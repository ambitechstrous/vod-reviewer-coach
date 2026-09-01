import { Link } from 'react-router-dom'
import type { Video } from '../types'
import { StatusBadge } from './StatusBadge'
import { VideoThumbnail } from './VideoThumbnail'

function formatUploadedAt(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })
}

export function VideoTile({ video }: { video: Video }) {
  return (
    <Link
      to={`/videos/${video.id}`}
      className="flex items-center gap-4 rounded-xl border border-white/5 bg-white/[0.03] p-3 transition hover:border-white/10 hover:bg-white/[0.06]"
    >
      <VideoThumbnail
        hue={video.thumbnailHue}
        status={video.status}
        durationLabel={video.durationLabel}
        className="h-16 w-28"
      />

      <div className="min-w-0 flex-1">
        <p className="truncate font-medium text-white">{video.title}</p>
        <p className="mt-0.5 text-sm text-slate-400">
          {video.game} &middot; {formatUploadedAt(video.uploadedAt)}
        </p>
      </div>

      <StatusBadge status={video.status} />
    </Link>
  )
}
