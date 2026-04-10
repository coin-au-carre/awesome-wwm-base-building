import { readFileSync } from "fs"

export type EventStatus = "scheduled" | "active" | "completed" | "canceled"
export type EventType = "tour" | "pvp" | "marriage" | "dancing" | "fashion" | "contest" | "other"

export interface Event {
  id: string
  name: string
  description?: string
  guildName?: string
  guildId?: string
  type?: EventType
  scheduledStart: string   // ISO 8601
  scheduledEnd?: string
  location?: string
  status: EventStatus
  subscriberCount: number
  discordUrl: string
}

export const EVENT_TYPE_LABELS: Record<EventType, string> = {
  tour:     "Tour",
  pvp:      "PvP",
  marriage: "Marriage",
  dancing:  "Dancing",
  fashion:  "Fashion",
  contest:  "Contest",
  other:    "Other",
}

export const EVENT_TYPE_EMOJI: Record<EventType, string> = {
  tour:     "🗺️",
  pvp:      "⚔️",
  marriage: "💍",
  dancing:  "💃",
  fashion:  "👗",
  contest:  "🏆",
  other:    "📅",
}

function loadEvents(): Event[] {
  try {
    const raw = readFileSync(new URL("../../../data/events.json", import.meta.url), "utf-8")
    return JSON.parse(raw)
  } catch {
    return []
  }
}

const ALL_EVENTS: Event[] = loadEvents()

export function getUpcomingEvents(): Event[] {
  const now = new Date()
  return ALL_EVENTS
    .filter((e) => e.status === "scheduled" || e.status === "active")
    .filter((e) => !e.scheduledEnd || new Date(e.scheduledEnd) > now)
    .sort((a, b) => new Date(a.scheduledStart).getTime() - new Date(b.scheduledStart).getTime())
}

export function getAllEvents(): Event[] {
  return ALL_EVENTS
}
