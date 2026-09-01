import { useEffect, useRef, useState, type DragEvent } from 'react'
import { useAuth } from '../lib/auth'
import {
  abortUploadSession,
  createUploadSession,
  finishUploadSession,
  type UploadSession,
} from '../lib/upload'

type Status = 'idle' | 'creating-session' | 'ready' | 'uploading' | 'done' | 'error'

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB']
  let value = bytes
  let unitIndex = -1
  do {
    value /= 1024
    unitIndex++
  } while (value >= 1024 && unitIndex < units.length - 1)
  return `${value.toFixed(1)} ${units[unitIndex]}`
}

export function UploadVideoPanel({ onClose }: { onClose: () => void }) {
  const { user } = useAuth()
  const [file, setFile] = useState<File | null>(null)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [session, setSession] = useState<UploadSession | null>(null)
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState<string | null>(null)
  const [isDraggingOver, setIsDraggingOver] = useState(false)
  const [progress, setProgress] = useState(0)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Mirror the latest session/status in refs so the unmount cleanup below
  // (which only runs once) can see current values instead of stale closures.
  const sessionRef = useRef(session)
  const statusRef = useRef(status)
  useEffect(() => {
    sessionRef.current = session
    statusRef.current = status
  }, [session, status])

  // Abort any session that's still open if the panel is closed without the
  // user explicitly clearing or finishing the upload.
  useEffect(() => {
    return () => {
      if (sessionRef.current && statusRef.current !== 'done' && user) {
        void abortUploadSession(sessionRef.current, user.token)
      }
    }
  }, [user])

  useEffect(() => {
    return () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl)
    }
  }, [previewUrl])

  async function selectFile(newFile: File) {
    if (!user) return
    if (previewUrl) URL.revokeObjectURL(previewUrl)

    // A new drop replaces the in-progress session without aborting the old
    // one — it's simply abandoned and left for S3 to garbage-collect.
    setFile(newFile)
    setPreviewUrl(URL.createObjectURL(newFile))
    setSession(null)
    setError(null)
    setProgress(0)
    setStatus('creating-session')

    try {
      const newSession = await createUploadSession(crypto.randomUUID(), newFile, user.token)
      setSession(newSession)
      setStatus('ready')
    } catch (err) {
      setStatus('error')
      setError(err instanceof Error ? err.message : 'Failed to start upload')
    }
  }

  function handleDragOver(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    if (status !== 'uploading') setIsDraggingOver(true)
  }

  function handleDragLeave() {
    setIsDraggingOver(false)
  }

  function handleDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setIsDraggingOver(false)
    if (status === 'uploading') return

    const dropped = e.dataTransfer.files[0]
    if (!dropped) return
    if (!dropped.type.startsWith('video/')) {
      setError('Please drop a video file')
      return
    }
    void selectFile(dropped)
  }

  function handleClear() {
    if (session && user) void abortUploadSession(session, user.token)
    if (previewUrl) URL.revokeObjectURL(previewUrl)
    setFile(null)
    setPreviewUrl(null)
    setSession(null)
    setStatus('idle')
    setError(null)
    setProgress(0)
  }

  async function handleSubmit() {
    if (!file || !session || !user) return
    setStatus('uploading')
    setError(null)

    try {
      await finishUploadSession(session, file, user.token, {
        onProgress: ({ uploadedBytes, totalBytes }) =>
          setProgress(Math.round((uploadedBytes / totalBytes) * 100)),
      })
      setStatus('done')
    } catch (err) {
      // finishUploadSession already aborted the session on failure.
      setSession(null)
      setStatus('error')
      setError(err instanceof Error ? err.message : 'Upload failed')
    }
  }

  const isBusy = status === 'creating-session' || status === 'uploading'

  return (
    <div className="rounded-2xl border border-white/5 bg-white/[0.03] p-5">
      <div className="mb-4 flex items-center justify-between">
        <p className="font-medium text-white">Upload a video</p>
        <button
          onClick={onClose}
          className="text-sm text-slate-400 transition hover:text-white"
        >
          Close
        </button>
      </div>

      {status === 'done' ? (
        <div className="rounded-lg border border-emerald-400/30 bg-emerald-500/10 p-4 text-sm text-emerald-300">
          Upload complete — analysis will start shortly.
        </div>
      ) : (
        <div
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          className={`flex flex-col items-center gap-3 rounded-xl border-2 border-dashed p-6 text-center transition ${
            isDraggingOver ? 'border-indigo-400 bg-indigo-500/5' : 'border-white/10'
          }`}
        >
          {file && previewUrl ? (
            <div className="flex w-full items-center gap-4">
              <video
                src={previewUrl}
                muted
                preload="metadata"
                className="h-20 w-32 shrink-0 rounded-lg bg-black object-cover"
              />
              <div className="min-w-0 flex-1 text-left">
                <p className="truncate text-sm font-medium text-white">{file.name}</p>
                <p className="text-xs text-slate-400">{formatBytes(file.size)}</p>
                {status === 'creating-session' && (
                  <p className="mt-1 text-xs text-slate-400">Preparing upload&hellip;</p>
                )}
                {status === 'uploading' && (
                  <p className="mt-1 text-xs text-slate-400">Uploading&hellip; {progress}%</p>
                )}
              </div>
              <button
                onClick={handleClear}
                disabled={status === 'uploading'}
                className="shrink-0 rounded-md border border-white/10 px-2.5 py-1.5 text-xs text-slate-300 transition hover:bg-white/5 disabled:cursor-not-allowed disabled:opacity-40"
              >
                Clear
              </button>
            </div>
          ) : (
            <>
              <p className="text-sm text-slate-300">Drag and drop a video here</p>
              <p className="text-xs text-slate-500">or</p>
              <button
                onClick={() => fileInputRef.current?.click()}
                className="rounded-lg border border-white/10 px-3 py-1.5 text-sm text-slate-200 transition hover:bg-white/5"
              >
                Browse files
              </button>
              <input
                ref={fileInputRef}
                type="file"
                accept="video/*"
                className="hidden"
                onChange={(e) => {
                  const picked = e.target.files?.[0]
                  if (picked) void selectFile(picked)
                  e.target.value = ''
                }}
              />
            </>
          )}
        </div>
      )}

      {error && <p className="mt-3 text-sm text-red-400">{error}</p>}

      {status !== 'done' && (
        <div className="mt-4 flex justify-end">
          <button
            onClick={handleSubmit}
            disabled={!session || isBusy}
            className="rounded-lg bg-indigo-500 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-400 disabled:cursor-not-allowed disabled:opacity-40"
          >
            {status === 'uploading' ? 'Uploading…' : 'Submit'}
          </button>
        </div>
      )}
    </div>
  )
}
