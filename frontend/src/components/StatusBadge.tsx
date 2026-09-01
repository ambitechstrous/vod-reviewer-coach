import type { VideoStatus } from '../types'

const STATUS_STYLES: Record<VideoStatus, string> = {
  uploaded: 'bg-slate-700/60 text-slate-200 ring-slate-500/40',
  analyzing: 'bg-amber-500/15 text-amber-300 ring-amber-400/40',
  processed: 'bg-emerald-500/15 text-emerald-300 ring-emerald-400/40',
}

const STATUS_LABELS: Record<VideoStatus, string> = {
  uploaded: 'Uploaded',
  analyzing: 'Analyzing',
  processed: 'Processed',
}

export function StatusBadge({ status }: { status: VideoStatus }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ring-1 ring-inset ${STATUS_STYLES[status]}`}
    >
      {status === 'analyzing' && (
        <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-amber-300" />
      )}
      {STATUS_LABELS[status]}
    </span>
  )
}
