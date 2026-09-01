import { ChatPanel } from '../components/ChatPanel'
import { VideoTile } from '../components/VideoTile'
import { mockVideos } from '../data/mockVideos'

export function LandingPage() {
  return (
    <div className="flex flex-col gap-10">
      <div>
        <h1 className="text-xl font-semibold text-white">Your videos</h1>
        <p className="mt-1 text-sm text-slate-400">
          Uploads are analyzed automatically. Click a video once it's
          processed to see your coaching report.
        </p>

        <div className="mt-5 flex flex-col gap-2.5">
          {mockVideos.map((video) => (
            <VideoTile key={video.id} video={video} />
          ))}
        </div>
      </div>

      <ChatPanel />
    </div>
  )
}
