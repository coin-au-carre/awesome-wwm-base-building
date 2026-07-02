// cmd/tutorial-gdoc/main.go — fetch Google Doc → web/src/content/articles/*.md
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ruby/internal/cmdutil"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/chai2010/webp"
)

var reDocID = regexp.MustCompile(`/document/d/([a-zA-Z0-9_-]+)`)
var reSlug = regexp.MustCompile(`[^\p{L}\p{N}]+`)
var reGdocIDField = regexp.MustCompile(`(?m)^gdocID:\s*"([^"]+)"`)
var reUpdatedDate = regexp.MustCompile(`(?m)^updatedDate:.*\n?`)
var reTitleTag = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
var reFirstH1 = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
var reGdocTitleP = regexp.MustCompile(`(?is)<p[^>]+\bclass="[^"]*\btitle\b[^"]*"[^>]*>.*?</p>`)
var reHeading = regexp.MustCompile(`(?m)^(#{1,5}) `)
var reDataURI = regexp.MustCompile(`src="data:image/([^;]+);base64,([^"]+)"`)

func slugify(s string) string {
	return strings.Trim(reSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func main() {
	root := flag.String("root", cmdutil.RootDir(), "repository root directory")
	list := flag.String("list", "", "file with one Google Doc URL per line")
	flag.Parse()

	cmdutil.LoadEnv(*root)

	var docURLs []string
	if *list != "" {
		f, err := os.Open(*list)
		if err != nil {
			slog.Error("opening list file", "err", err)
			os.Exit(1)
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			docURLs = append(docURLs, line)
		}
		f.Close()
	}
	docURLs = append(docURLs, flag.Args()...)

	// Also pick up docs already tracked in article frontmatters.
	articlesDir := filepath.Join(*root, "web", "src", "content", "articles")
	docURLs = append(docURLs, collectFrontmatterGdocURLs(articlesDir)...)
	docURLs = dedupByDocID(docURLs)

	if len(docURLs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: tutorial-gdoc [-list <file>] <google-doc-url>...")
		os.Exit(1)
	}

	for _, u := range docURLs {
		if err := syncDoc(*root, u); err != nil {
			slog.Error("syncing doc", "url", u, "err", err)
		}
	}
}

func syncDoc(root, docURL string) error {
	m := reDocID.FindStringSubmatch(docURL)
	if m == nil {
		return fmt.Errorf("invalid Google Doc URL: %s", docURL)
	}
	docID := m[1]

	exportURL := fmt.Sprintf("https://docs.google.com/document/d/%s/export?format=html", docID)
	resp, err := http.Get(exportURL) //nolint:noctx
	if err != nil {
		return fmt.Errorf("fetching doc: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("fetching doc: HTTP %d", resp.StatusCode)
	}
	htmlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading doc: %w", err)
	}
	htmlStr := string(htmlBytes)

	title := extractTitle(htmlStr)
	if title == "" {
		title = docID
	}

	articlesDir := filepath.Join(root, "web", "src", "content", "articles")

	// Use existing article filename as slug so image paths stay stable across title changes.
	outPath := findArticleByGdocID(articlesDir, docID)
	var slug string
	if outPath != "" {
		slug = strings.TrimSuffix(filepath.Base(outPath), ".md")
	} else {
		slug = slugify(title)
		outPath = filepath.Join(articlesDir, slug+".md")
	}

	imagesDir := filepath.Join(root, "web", "public", "tutorials", slug)
	manifestPath := filepath.Join(imagesDir, ".manifest.json")
	manifest := loadImageManifest(manifestPath)
	nextNum := nextImageNum(manifest)
	counter := 0
	htmlStr = reDataURI.ReplaceAllStringFunc(htmlStr, func(match string) string {
		sub := reDataURI.FindStringSubmatch(match)
		_, data := sub[1], sub[2]
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return match
		}
		// Hash the raw image bytes so a filename stays attached to the same image
		// regardless of where it moves in the doc (reordering/insertion doesn't
		// shift numbering out from under existing width customizations).
		hash := sha256.Sum256(decoded)
		key := hex.EncodeToString(hash[:])
		filename, known := manifest[key]
		if !known {
			filename = fmt.Sprintf("img-%d.webp", nextNum)
			nextNum++
		}
		if err := os.MkdirAll(imagesDir, 0755); err == nil {
			if webpBytes, err := toWebP(decoded); err == nil {
				_ = os.WriteFile(filepath.Join(imagesDir, filename), webpBytes, 0644)
			} else {
				slog.Warn("webp encode failed, skipping image", "err", err)
				return match
			}
		}
		manifest[key] = filename
		counter++
		return fmt.Sprintf(`src="/tutorials/%s/%s"`, slug, filename)
	})
	if counter > 0 {
		slog.Info("extracted images", "count", counter, "dir", imagesDir)
		saveImageManifest(manifestPath, manifest)
	}

	// Strip the Google Docs Title-style paragraph from the body (duplicates frontmatter title).
	// H1 tags are Heading 1 content and must be preserved.
	body := reGdocTitleP.ReplaceAllLiteralString(htmlStr, "")

	markdown, err := htmltomarkdown.ConvertString(body)
	if err != nil {
		return fmt.Errorf("converting to markdown: %w", err)
	}
	markdown = strings.TrimSpace(markdown)
	// Shift all headings down one level (H1→H2 … H5→H6) so the page <h1> stays the frontmatter title.
	markdown = reHeading.ReplaceAllString(markdown, "$1# ")

	var content string
	if existing, err := os.ReadFile(outPath); err == nil {
		markdown = preserveImageCustomizations(markdown, string(existing), slug)
		if fm := extractFrontmatter(string(existing)); fm != "" {
			fm = setUpdatedDate(fm)
			content = fmt.Sprintf("---\n%s---\n\n%s\n", fm, markdown)
		} else {
			content = buildMarkdown(title, docID, markdown)
		}
	} else {
		content = buildMarkdown(title, docID, markdown)
	}

	if err := os.MkdirAll(articlesDir, 0755); err != nil {
		return fmt.Errorf("creating articles dir: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing article: %w", err)
	}

	fmt.Printf("wrote  %s\n", outPath)
	return nil
}

// group 1 = full path, group 2 = filename
var reImgNum = regexp.MustCompile(`^img-(\d+)\.webp$`)

// loadImageManifest reads the hash→filename map for a tutorial's images, if any.
func loadImageManifest(path string) map[string]string {
	m := make(map[string]string)
	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, &m)
	return m
}

func saveImageManifest(path string, m map[string]string) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0644)
}

// nextImageNum finds the smallest unused img-N.webp number so new images
// don't collide with filenames already claimed by known hashes.
func nextImageNum(manifest map[string]string) int {
	max := 0
	for _, filename := range manifest {
		if m := reImgNum.FindStringSubmatch(filename); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n > max {
				max = n
			}
		}
	}
	return max + 1
}

// group 1 = full path, group 2 = filename
var reNewMDImg = regexp.MustCompile(`!\[[^\]]*\]\((/tutorials/[^/]+/([^)]+))\)`)

// preserveImageCustomizations keeps custom img markup (e.g. <img width="50%">) from the
// existing article instead of the plain ![](…) that the sync would regenerate.
func preserveImageCustomizations(newMD, oldContent, slug string) string {
	q := regexp.QuoteMeta(slug)
	reOldMD := regexp.MustCompile(`!\[[^\]]*\]\(/tutorials/` + q + `/([^)]+)\)`)
	reOldHTML := regexp.MustCompile(`<img[^>]+src="/tutorials/` + q + `/([^"]+)"[^>]*(?:/>|>)`)

	existing := make(map[string]string)
	for _, m := range reOldMD.FindAllStringSubmatch(oldContent, -1) {
		existing[m[1]] = m[0]
	}
	for _, m := range reOldHTML.FindAllStringSubmatch(oldContent, -1) {
		existing[m[1]] = m[0]
	}
	if len(existing) == 0 {
		return newMD
	}
	return reNewMDImg.ReplaceAllStringFunc(newMD, func(match string) string {
		sub := reNewMDImg.FindStringSubmatch(match)
		fullPath, filename := sub[1], sub[2]
		if custom, ok := existing[filename]; ok {
			return custom
		}
		return fmt.Sprintf(`<img src="%s" style="width: 80%%" />`, fullPath)
	})
}

func extractTitle(htmlStr string) string {
	// Prefer the Google Docs "Title" paragraph style (<p class="...title...">),
	// then the HTML <title> tag, then first <h1>.
	if m := reGdocTitleP.FindStringSubmatch(htmlStr); m != nil {
		if t := strings.TrimSpace(stripTags(m[0])); t != "" {
			return t
		}
	}
	if m := reTitleTag.FindStringSubmatch(htmlStr); m != nil {
		if t := strings.TrimSpace(m[1]); t != "" {
			return t
		}
	}
	if m := reFirstH1.FindStringSubmatch(htmlStr); m != nil {
		return strings.TrimSpace(stripTags(m[1]))
	}
	return ""
}

func toWebP(raw []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: 80}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var reTagStrip = regexp.MustCompile(`<[^>]+>`)

func stripTags(s string) string {
	return reTagStrip.ReplaceAllString(s, "")
}

func setUpdatedDate(fm string) string {
	today := "updatedDate: " + time.Now().Format("2006-01-02") + "\n"
	if reUpdatedDate.MatchString(fm) {
		return reUpdatedDate.ReplaceAllString(fm, today)
	}
	return fm + today
}

func extractFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return ""
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return ""
	}
	return strings.TrimSpace(parts[1]) + "\n"
}

func buildMarkdown(title, docID, body string) string {
	return fmt.Sprintf("---\ntitle: %q\ndescription: \"\"\ntags: []\nauthors: []\ndate: %s\norder: 99\ngdocID: %q\n---\n\n%s\n",
		title, time.Now().Format("2006-01-02"), docID, body)
}

func collectFrontmatterGdocURLs(articlesDir string) []string {
	entries, _ := os.ReadDir(articlesDir)
	var urls []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(articlesDir, e.Name()))
		if err != nil {
			continue
		}
		fm := extractFrontmatter(string(data))
		if m := reGdocIDField.FindStringSubmatch(fm); m != nil {
			urls = append(urls, "https://docs.google.com/document/d/"+m[1])
		}
	}
	return urls
}

func findArticleByGdocID(articlesDir, docID string) string {
	entries, _ := os.ReadDir(articlesDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(articlesDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if fm := extractFrontmatter(string(data)); strings.Contains(fm, docID) {
			return path
		}
	}
	return ""
}

// dedupByDocID keeps the first URL seen for each Google Doc ID.
func dedupByDocID(urls []string) []string {
	seen := make(map[string]bool, len(urls))
	var out []string
	for _, u := range urls {
		m := reDocID.FindStringSubmatch(u)
		if m == nil {
			continue
		}
		id := m[1]
		if !seen[id] {
			seen[id] = true
			out = append(out, u)
		}
	}
	return out
}
