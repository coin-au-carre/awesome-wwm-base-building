export type ItemCategory =
  | "Floor"
  | "Wall"
  | "Roof"
  | "Door & Window"
  | "Pillar"
  | "Decoration"
  | "Furniture"
  | "Landscape"

export type ItemMode = "guild" | "solo" | "both"

export interface BuildItem {
  id: string
  name: string
  category: ItemCategory
  imageUrl?: string
  mode: ItemMode
  tags?: string[]
}
