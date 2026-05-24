import { readFileSync } from "fs"
import { resolve } from "path"

export type EventStatus = "scheduled" | "active" | "completed" | "canceled"
export type EventType = "tour" | "pvp" | "marriage" | "dancing" | "fashion" | "contest" | "race" | "streaming" | "other"
export type ChannelType = "voice" | "stage"

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
  channelType?: ChannelType
  channelName?: string
  status: EventStatus
  subscriberCount: number
  discordUrl: string
  image?: string
}

export const EVENT_TYPE_LABELS: Record<EventType, string> = {
  tour:     "Tour",
  pvp:      "PvP",
  marriage: "Marriage",
  dancing:  "Dancing",
  fashion:  "Fashion",
  contest:  "Contest",
  race:      "Race",
  streaming: "Streaming",
  other:     "Other",
}

export const EVENT_TYPE_EMOJI: Record<EventType, string> = {
  tour:     "🗺️",
  pvp:      "⚔️",
  marriage: "💍",
  dancing:  "💃",
  fashion:  "👗",
  contest:  "🏆",
  race:      "🏁",
  streaming: "🎥",
  other:     "📅",
}

function loadEvents(): Event[] {
  try {
    const raw = readFileSync(resolve(process.cwd(), "..", "data/events.json"), "utf-8")
    return JSON.parse(raw)
  } catch {
    return []
  }
}

const ALL_EVENTS: Event[] = loadEvents()

export function getUpcomingEvents(): Event[] {
  return ALL_EVENTS
    .filter((e) => e.status === "scheduled" || e.status === "active")
    .sort((a, b) => new Date(a.scheduledStart).getTime() - new Date(b.scheduledStart).getTime())
}

export function getAllEvents(): Event[] {
  return ALL_EVENTS
}
