import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { StatusBadge } from '../components/StatusBadge'
import { SessionExpiredError } from '../lib/api'
import { useAuth } from '../lib/auth'
import { fetchVideoDetails } from '../lib/videos'
import type { Video } from '../types'

export function VideoDetailPage() {
  const { videoId } = useParams<{ videoId: string }>()
  const { user, logout } = useAuth()
  const [video, setVideo] = useState<Video | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!user || !videoId) return
    const controller = new AbortController()
    setIsLoading(true)
    setError(null)

    fetchVideoDetails(videoId, user.token, controller.signal)
      .then(setVideo)
      .catch((err) => {
        if (controller.signal.aborted) return
        if (err instanceof SessionExpiredError) {
          logout()
          return
        }
        setError('Could not load this video.')
      })
      .finally(() => {
        if (!controller.signal.aborted) setIsLoading(false)
      })

    return () => controller.abort()
  }, [videoId, user, logout])

  if (isLoading) {
    return <p className="text-sm text-slate-400">Loading&hellip;</p>
  }

  if (error || !video) {
    return (
      <div className="flex flex-col items-start gap-3">
        <p className={error ? 'text-sm text-red-400' : 'text-slate-300'}>
          {error ?? 'Video not found.'}
        </p>
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
            <p className="text-sm text-white/50">Video preview unavailable.</p>
          </div>
        )}
      </div>

      <div>
        <StatusBadge status={video.status} />
      </div>

      {video.status === 'processed' && video.summary && (
        <div className="rounded-xl border border-white/5 bg-white/[0.03] p-5">
          <h2 className="mb-2 font-medium text-white">Coaching summary</h2>
          <div className="prose prose-invert prose-sm max-w-none prose-headings:font-semibold prose-a:text-indigo-400">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
              {video.summary}
            </ReactMarkdown>
          </div>
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
