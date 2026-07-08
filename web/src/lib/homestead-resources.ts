export interface HomesteadSheet {
  title: string
  description: string
  href: string
  by: string
  column?: "guide" | "sheet"
}

export const HOMESTEAD_SHEETS: HomesteadSheet[] = [
  {
    title: "Angel Retainer Guide",
    description: "Retainer planner",
    href: "https://docs.google.com/spreadsheets/d/1cKG4sqd7NNFWWqxfXkrAncJBYfbAULT9JcHKZ-qUtYA/edit?usp=sharing",
    by: "LittleMissAngel♡",
    column: "guide",
  },
  {
    title: "Simple Farming & Specialties",
    description: "Farming & specialties consumption tracker",
    href: "https://docs.google.com/spreadsheets/d/1RnfMBbyvffaij8JeBIoMJQlQHI9S15qAmxZzDwZsoSE/edit?usp=sharing",
    by: "KCrazy",
    column: "guide",
  },
  {
    title: "Goose Calcs",
    description: "Recipes calculation & planner",
    href: "https://docs.google.com/spreadsheets/d/1NI0uJ9t-pEX2idqO1aUDRJgGZhxXPGyIiUbcbkgWiVA/edit?usp=sharing",
    by: "Goose",
  },
  {
    title: "Dravish Arbiter System",
    description: "Seller planner",
    href: "https://docs.google.com/spreadsheets/d/1WvhDRh5EIIH6LPvNXi_GbB0s56CpciFD9vHdmko2_4Q/edit?usp=sharing",
    by: "Dravish",
  },
  {
    title: "Goth Planner v2.0",
    description: "Retainers, crops, farms, weekly plan, queues, order planner",
    href: "https://docs.google.com/spreadsheets/d/1KqUKDaL74fDClVv-Ow-uNXcrXdX5q08uvy_KNqK0iJo/edit?usp=sharing",
    by: "Goth",
  },
]
