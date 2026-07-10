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
  breaks: true,
  extensions: [spoilerExtension],
  renderer: {
    link(token) {
      const isExternal = token.href?.startsWith("http")
      return `<a href="${token.href ?? ""}"${isExternal ? ` target="_blank" rel="noopener noreferrer"` : ""} class="underline hover:opacity-75 transition-opacity">${token.text}</a>`
    },
    checkbox({ checked }: { checked: boolean }) {
      return `<input type="checkbox"${checked ? ' checked=""' : ""}> `
    },
  },
})

// Normalize Discord-style loose bold markers: ** text ** → **text**
const reLooseBold = /\*\* ?(.+?) ?\*\*/g

// Strip "**" glued directly onto a code fence (e.g. "**```" or "```**") — bold can't wrap a
// fenced code block, and the trailing "**" makes the closing fence invalid, swallowing the
// rest of the text into the code block. Discord tolerates this typo; marked does not.
const reBoldOnFence = /\*\*(`+)|(`+)\*\*/g

export function renderMarkdown(text: string): string {
  const normalized = text.replace(reBoldOnFence, "$1$2").replace(reLooseBold, "**$1**")
  return instance.parse(normalized, { async: false }) as string
}
