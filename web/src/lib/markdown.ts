export function renderMarkdown(text: string): string {
  const bold = (s: string) => s.replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
  const paragraphs = text.trim().split(/\n{2,}/)
  return paragraphs
    .map((block) => {
      const lines = block.split("\n").map((l) => l.trimEnd())
      const isList = lines.every((l) => /^[-*]\s/.test(l.trim()) || l.trim() === "")
      if (isList) {
        const items = lines
          .filter((l) => /^[-*]\s/.test(l.trim()))
          .map((l) => `<li>${bold(l.trim().replace(/^[-*]\s/, ""))}</li>`)
          .join("")
        return `<ul class="space-y-1.5 list-none">${items}</ul>`
      }
      return `<p>${bold(lines.join("<br>"))}</p>`
    })
    .join("\n")
}
