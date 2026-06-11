import { Marked } from "marked"

const spoilerExtension = {
  name: "spoiler",
  level: "inline" as const,
  start(src: string) {
    return src.indexOf("||")
  },
  tokenizer(src: string) {
    const match = /^\|\|(.+?)\|\|/.exec(src)
    if (match) return { type: "spoiler", raw: match[0], text: match[1] }
  },
  renderer(token: { text: string }) {
    return `<span class="spoiler cursor-pointer select-none rounded bg-foreground/10 px-0.5 text-transparent [&.revealed]:text-inherit transition-colors" onclick="this.classList.toggle('revealed')" title="Click to reveal">${token.text}</span>`
  },
}

const instance = new Marked({
  extensions: [spoilerExtension],
  renderer: {
    link(token) {
      const isExternal = token.href?.startsWith("http")
      return `<a href="${token.href ?? ""}"${isExternal ? ` target="_blank" rel="noopener noreferrer"` : ""} class="underline hover:opacity-75 transition-opacity">${token.text}</a>`
    },
    listitem(token) {
      return `<li><span>${token.text}</span></li>\n`
    },
  },
})

export function renderMarkdown(text: string): string {
  return instance.parse(text, { async: false }) as string
}
