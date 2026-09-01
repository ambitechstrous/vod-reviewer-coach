import { useEffect, useState } from 'react'
import { ChatPanel } from '../components/ChatPanel'
import { UploadVideoPanel } from '../components/UploadVideoPanel'
import { VideoTile } from '../components/VideoTile'
import { SessionExpiredError } from '../lib/api'
import { useAuth } from '../lib/auth'
import type { Video } from '../types'
import { fetchVideos } from '../lib/videos'

export function LandingPage() {
  const { user, logout } = useAuth()
  const [isUploadOpen, setIsUploadOpen] = useState(false)
  const [videos, setVideos] = useState<Video[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!user) return
    const controller = new AbortController()

    fetchVideos(user.token, controller.signal)
      .then(setVideos)
      .catch((err) => {
        if (controller.signal.aborted) return
        if (err instanceof SessionExpiredError) {
          logout()
          return
        }
        setError('Could not load your videos.')
      })
      .finally(() => {
        if (!controller.signal.aborted) setIsLoading(false)
      })

    return () => controller.abort()
  }, [user, logout])

  return (
    <div className="flex flex-col gap-10">
      <div>
        <h1 className="text-xl font-semibold text-white">Your videos</h1>
        <p className="mt-1 text-sm text-slate-400">
          Uploads are analyzed automatically. Click a video once it's
          processed to see your coaching report.
        </p>

        <div className="mt-5 flex flex-col gap-2.5">
          {isLoading && <p className="text-sm text-slate-400">Loading your videos&hellip;</p>}
          {error && <p className="text-sm text-red-400">{error}</p>}
          {!isLoading && !error && videos.length === 0 && (
            <p className="text-sm text-slate-400">
              No videos yet — upload one below to get started.
            </p>
          )}
          {videos.map((video) => (
            <VideoTile key={video.id} video={video} />
          ))}
        </div>

        <div className="mt-4">
          {isUploadOpen ? (
            <UploadVideoPanel onClose={() => setIsUploadOpen(false)} />
          ) : (
            <button
              onClick={() => setIsUploadOpen(true)}
              className="w-full rounded-xl border border-dashed border-white/10 py-3 text-sm text-slate-400 transition hover:border-white/20 hover:text-white"
            >
              + Upload video
            </button>
          )}
        </div>
      </div>

      <ChatPanel />
    </div>
  )
}
