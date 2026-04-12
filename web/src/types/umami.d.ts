interface Umami {
  track(event: string, data?: Record<string, unknown>): void
}

interface Window {
  umami?: Umami
}
