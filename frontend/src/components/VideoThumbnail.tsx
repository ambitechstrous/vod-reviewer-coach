import type { VideoStatus } from '../types'

export function VideoThumbnail({
  hue,
  status,
  durationLabel,
  className = '',
}: {
  hue: number
  status: VideoStatus
  durationLabel: string
  className?: string
}) {
  return (
    <div
      className={`relative flex shrink-0 items-center justify-center overflow-hidden rounded-lg ${className}`}
      style={{
        background: `linear-gradient(135deg, hsl(${hue} 70% 22%), hsl(${hue} 70% 10%))`,
      }}
    >
      <svg
        viewBox="0 0 24 24"
        className={`h-8 w-8 ${status === 'processed' ? 'text-white/70' : 'text-white/30'}`}
        fill="currentColor"
      >
        <path d="M8 5v14l11-7z" />
      </svg>
      <span className="absolute bottom-1 right-1.5 rounded bg-black/60 px-1.5 py-0.5 text-[11px] font-medium text-white/80">
        {durationLabel}
      </span>
    </div>
  )
}
