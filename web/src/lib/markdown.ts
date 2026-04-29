export function renderMarkdown(text: string): string {
  const inline = (s: string) =>
    s
      .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
      .replace(/~~(.+?)~~/g, "<del>$1</del>")
      .replace(/\*(.+?)\*/g, "<em>$1</em>")
      .replace(/_(.+?)_/g, "<em>$1</em>")
      .replace(/\[([^\]]+)\]\((https?:\/\/[^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer" class="underline hover:opacity-75 transition-opacity">$1</a>')
      .replace(/\|\|(.+?)\|\|/g, '<span class="spoiler cursor-pointer select-none rounded bg-foreground/10 px-0.5 text-transparent [&.revealed]:text-inherit transition-colors" onclick="this.classList.toggle(\'revealed\')" title="Click to reveal">$1</span>')
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
