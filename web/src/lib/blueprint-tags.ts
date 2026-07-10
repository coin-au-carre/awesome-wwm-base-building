/** Discord forum tags that describe what the diagram is for, shown as a distinct badge on the frontend. */
export const DIAGRAM_TYPES = ["Homestead Diagram", "Small Solo Diagram", "Small Guild Diagram", "Large Guild Diagram"] as const

/** Discord forum tags that duplicate the isFree/isPayToBuild price badge — dropped from generic tag pills/filters. */
const PRICE_TAGS = ["Free", "Paid"] as const

export function getDiagramType(tags?: string[]): string | undefined {
  return tags?.find((t) => (DIAGRAM_TYPES as readonly string[]).includes(t))
}

/** Tags safe to render as generic pills: excludes the diagram-type tag (shown as its own badge) and price tags (shown as the price badge). */
export function displayTags(tags?: string[]): string[] {
  return (tags ?? []).filter(
    (t) => !(DIAGRAM_TYPES as readonly string[]).includes(t) && !(PRICE_TAGS as readonly string[]).includes(t)
  )
}
