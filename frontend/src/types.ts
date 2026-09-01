export type VideoStatus = 'uploaded' | 'analyzing' | 'processed'

export interface Video {
  id: string
  title: string
  game: string
  status: VideoStatus
  uploadedAt: string
  durationLabel: string
  thumbnailHue: number
  videoUrl?: string
  summary?: string
}

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  text: string
}
