export function renderMarkdown(text: string): string {
  const inline = (s: string) =>
    s
      .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
      .replace(/~~(.+?)~~/g, "<del>$1</del>")
      .replace(/\*(.+?)\*/g, "<em>$1</em>")
      .replace(/_(.+?)_/g, "<em>$1</em>")
  const paragraphs = text.trim().split(/\n{2,}/)
  return paragraphs
    .map((block) => {
      const lines = block.split("\n").map((l) => l.trimEnd())
      const isList = lines.every((l) => /^[-*]\s/.test(l.trim()) || l.trim() === "")
      if (isList) {
        const items = lines
          .filter((l) => /^[-*]\s/.test(l.trim()))
          .map((l) => `<li>${inline(l.trim().replace(/^[-*]\s/, ""))}</li>`)
          .join("")
        return `<ul class="space-y-1.5 list-none">${items}</ul>`
      }
      return `<p>${inline(lines.join("<br>"))}</p>`
    })
    .join("\n")
}
