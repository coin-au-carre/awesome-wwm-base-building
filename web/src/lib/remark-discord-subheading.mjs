/**
 * Remark plugin: transforms Discord's `-# text` small subheading syntax
 * into a paragraph with class "prose-subtext" for subtle styling.
 */
export function remarkDiscordSubheading() {
  return (tree) => {
    function visit(node) {
      if (node.type === "paragraph" && node.children?.length > 0) {
        const first = node.children[0]
        if (first.type === "text" && first.value.startsWith("-# ")) {
          first.value = first.value.slice(3)
          node.data = node.data ?? {}
          node.data.hProperties = { ...node.data.hProperties, className: "prose-subtext" }
        }
      }
      node.children?.forEach(visit)
    }
    visit(tree)
  }
}
