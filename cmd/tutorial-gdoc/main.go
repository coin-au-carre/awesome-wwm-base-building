// cmd/tutorial-gdoc/main.go — fetch Google Doc → web/src/content/articles/*.md
package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ruby/internal/cmdutil"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)


var reDocID = regexp.MustCompile(`/document/d/([a-zA-Z0-9_-]+)`)
var reSlug = regexp.MustCompile(`[^\p{L}\p{N}]+`)
var reGdocIDField = regexp.MustCompile(`(?m)^gdocID:\s*"([^"]+)"`)
var reTitleTag = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
var reFirstH1 = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
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
	docURLs = dedup(docURLs)

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
	slug := slugify(title)

	articlesDir := filepath.Join(root, "web", "src", "content", "articles")
	imagesDir := filepath.Join(root, "web", "public", "tutorials", slug)

	// Decode base64 data URI images → local files, rewrite src to public path.
	counter := 0
	htmlStr = reDataURI.ReplaceAllStringFunc(htmlStr, func(match string) string {
		sub := reDataURI.FindStringSubmatch(match)
		ext, data := sub[1], sub[2]
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return match
		}
		counter++
		filename := fmt.Sprintf("img-%d.%s", counter, ext)
		if err := os.MkdirAll(imagesDir, 0755); err == nil {
			_ = os.WriteFile(filepath.Join(imagesDir, filename), decoded, 0644)
		}
		return fmt.Sprintf(`src="/tutorials/%s/%s"`, slug, filename)
	})
	if counter > 0 {
		slog.Info("extracted images", "count", counter, "dir", imagesDir)
	}

	// Strip the first H1 (it duplicates the frontmatter title).
	stripped := false
	body := reFirstH1.ReplaceAllStringFunc(htmlStr, func(s string) string {
		if !stripped {
			stripped = true
			return ""
		}
		return s
	})

	markdown, err := htmltomarkdown.ConvertString(body)
	if err != nil {
		return fmt.Errorf("converting to markdown: %w", err)
	}
	markdown = strings.TrimSpace(markdown)

	outPath := findArticleByGdocID(articlesDir, docID)
	if outPath == "" {
		outPath = filepath.Join(articlesDir, slug+".md")
	}

	var content string
	if existing, err := os.ReadFile(outPath); err == nil {
		// Preserve any image customizations (size, style) from the existing file.
		markdown = preserveImageCustomizations(markdown, string(existing), slug)
		if fm := extractFrontmatter(string(existing)); fm != "" {
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


func extractTitle(htmlStr string) string {
	// Try <title> first, fall back to first <h1>.
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

var reTagStrip = regexp.MustCompile(`<[^>]+>`)
var reNewMDImg = regexp.MustCompile(`!\[[^\]]*\]\(/tutorials/[^/]+/([^)]+)\)`)

// preserveImageCustomizations replaces auto-generated ![](/tutorials/slug/img-N.ext) in newMD
// with whatever representation exists in oldContent for that filename (e.g. <img width="500">).
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
		filename := reNewMDImg.FindStringSubmatch(match)[1]
		if custom, ok := existing[filename]; ok {
			return custom
		}
		return match
	})
}

func stripTags(s string) string {
	return reTagStrip.ReplaceAllString(s, "")
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

func dedup(urls []string) []string {
	seen := make(map[string]bool, len(urls))
	var out []string
	for _, u := range urls {
		if !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}
	return out
}
