export const TAG_PALETTE = [
  { bg: "bg-violet-500/20", text: "text-violet-200", ring: "ring-1 ring-inset ring-violet-400/40" },
  { bg: "bg-sky-500/20", text: "text-sky-200", ring: "ring-1 ring-inset ring-sky-400/40" },
  { bg: "bg-emerald-500/20", text: "text-emerald-200", ring: "ring-1 ring-inset ring-emerald-400/40" },
  { bg: "bg-amber-500/20", text: "text-amber-200", ring: "ring-1 ring-inset ring-amber-400/40" },
  { bg: "bg-rose-500/20", text: "text-rose-200", ring: "ring-1 ring-inset ring-rose-400/40" },
  { bg: "bg-teal-500/20", text: "text-teal-200", ring: "ring-1 ring-inset ring-teal-400/40" },
  { bg: "bg-orange-500/20", text: "text-orange-200", ring: "ring-1 ring-inset ring-orange-400/40" },
  { bg: "bg-indigo-500/20", text: "text-indigo-200", ring: "ring-1 ring-inset ring-indigo-400/40" },
]

export function tagColor(tag: string): { bg: string; text: string; ring: string } {
  const hash = [...tag].reduce((a, c) => a + c.charCodeAt(0), 0)
  return TAG_PALETTE[hash % TAG_PALETTE.length]
}
