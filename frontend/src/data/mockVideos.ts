import type { Video } from '../types'

export const mockVideos: Video[] = [
  {
    id: 'vod-1042',
    title: 'Ranked Placements - Game 5',
    game: 'Valorant',
    status: 'processed',
    uploadedAt: '2026-08-29T18:04:00Z',
    durationLabel: '38:12',
    thumbnailHue: 265,
    videoUrl:
      'https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4',
    summary:
      "You held angles well in the first half but over-peeked mid on 4 of your 6 deaths, usually right after a teammate died nearby. Comms were sparse after the 8-minute mark — calling rotations even when you're not sure would help your team react faster. Your utility usage on site executes was clean and well-timed.",
  },
  {
    id: 'vod-1041',
    title: 'Scrim vs. Nova Esports',
    game: 'Valorant',
    status: 'processed',
    uploadedAt: '2026-08-28T21:30:00Z',
    durationLabel: '41:05',
    thumbnailHue: 190,
    videoUrl:
      'https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4',
    summary:
      "Strong entry fragging in the first 10 rounds, but your economy management slipped after two lost pistols — you force-bought three rounds in a row instead of saving. Positioning during retakes was consistently late; try rotating on the first callout instead of waiting for confirmation.",
  },
  {
    id: 'vod-1040',
    title: 'Solo Queue - Late Night Session',
    game: 'Apex Legends',
    status: 'analyzing',
    uploadedAt: '2026-09-01T02:15:00Z',
    durationLabel: '22:47',
    thumbnailHue: 25,
  },
  {
    id: 'vod-1039',
    title: 'Duo Ranked with Kess',
    game: 'Apex Legends',
    status: 'analyzing',
    uploadedAt: '2026-09-01T01:02:00Z',
    durationLabel: '19:33',
    thumbnailHue: 340,
  },
  {
    id: 'vod-1038',
    title: 'Tournament Qualifier - Map 2',
    game: 'Valorant',
    status: 'uploaded',
    uploadedAt: '2026-09-01T03:40:00Z',
    durationLabel: '35:20',
    thumbnailHue: 145,
  },
  {
    id: 'vod-1037',
    title: 'Warmup Aim Rounds',
    game: 'CS2',
    status: 'uploaded',
    uploadedAt: '2026-09-01T03:52:00Z',
    durationLabel: '12:08',
    thumbnailHue: 55,
  },
]
