import { Link, useParams } from 'react-router-dom'
import { StatusBadge } from '../components/StatusBadge'
import { mockVideos } from '../data/mockVideos'

export function VideoDetailPage() {
  const { videoId } = useParams<{ videoId: string }>()
  const video = mockVideos.find((v) => v.id === videoId)

  if (!video) {
    return (
      <div className="flex flex-col items-start gap-3">
        <p className="text-slate-300">Video not found.</p>
        <Link to="/" className="text-sm text-indigo-400 hover:underline">
          Back to videos
        </Link>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-5">
      <Link
        to="/"
        className="w-fit text-sm text-slate-400 transition hover:text-white"
      >
        &larr; Back to videos
      </Link>

      <h1 className="text-xl font-semibold text-white">{video.title}</h1>

      <div className="overflow-hidden rounded-xl border border-white/5 bg-black">
        {video.videoUrl ? (
          <video
            src={video.videoUrl}
            controls
            className="aspect-video w-full"
          />
        ) : (
          <div
            className="flex aspect-video w-full items-center justify-center"
            style={{
              background: `linear-gradient(135deg, hsl(${video.thumbnailHue} 70% 16%), hsl(${video.thumbnailHue} 70% 8%))`,
            }}
          >
            <p className="text-sm text-white/50">
              {video.status === 'analyzing'
                ? 'Video will be viewable once analysis finishes.'
                : 'Video uploaded — waiting to start analysis.'}
            </p>
          </div>
        )}
      </div>

      <div>
        <StatusBadge status={video.status} />
      </div>

      {video.status === 'processed' && video.summary && (
        <div className="rounded-xl border border-white/5 bg-white/[0.03] p-5">
          <h2 className="mb-2 font-medium text-white">Coaching summary</h2>
          <p className="text-sm leading-relaxed text-slate-300">
            {video.summary}
          </p>
        </div>
      )}

      {video.status === 'analyzing' && (
        <div className="rounded-xl border border-white/5 bg-white/[0.03] p-5">
          <p className="text-sm text-slate-400">
            We're analyzing this match — transcript and gameplay review are
            in progress. This page will update once feedback is ready.
          </p>
        </div>
      )}

      {video.status === 'uploaded' && (
        <div className="rounded-xl border border-white/5 bg-white/[0.03] p-5">
          <p className="text-sm text-slate-400">
            This video is queued for analysis. Check back shortly.
          </p>
        </div>
      )}
    </div>
  )
}
