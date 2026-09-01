import { API_BASE_URL } from './api'

// How many parts to upload to S3 in parallel. Multipart upload's main
// benefit over a single PUT is that parts can go out concurrently.
const MAX_CONCURRENT_PARTS = 4

interface UploadPartURL {
  part_number: number
  url: string
}

interface CreateUploadSessionResponse {
  key: string
  upload_id: string
  part_size: number
  parts: UploadPartURL[]
}

/** An in-progress multipart upload. No file bytes have been sent yet. */
export interface UploadSession {
  videoName: string
  uploadId: string
  partSize: number
  parts: UploadPartURL[]
}

export interface UploadProgress {
  uploadedBytes: number
  totalBytes: number
}

/** Thrown when the backend rejects a request's token (missing, expired, or invalid). */
export class SessionExpiredError extends Error {
  constructor() {
    super('Session expired — please log in again.')
    this.name = 'SessionExpiredError'
  }
}

/**
 * Starts a multipart upload session and returns a presigned URL for every
 * part. `videoName` is namespaced by the authenticated user server-side, so
 * it only needs to be unique per-user, not globally. `token` is the caller's
 * session token (see `useAuth`). No file bytes are transferred by this call.
 */
export async function createUploadSession(
  videoName: string,
  file: File,
  token: string,
  signal?: AbortSignal,
): Promise<UploadSession> {
  const res = await fetch(`${API_BASE_URL}/uploads`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      video_name: videoName,
      content_type: file.type || 'application/octet-stream',
      file_size: file.size,
    }),
    signal,
  })

  if (res.status === 401) throw new SessionExpiredError()
  if (!res.ok) {
    throw new Error(`failed to create upload session: ${res.status} ${await res.text()}`)
  }

  const data: CreateUploadSessionResponse = await res.json()
  return {
    videoName,
    uploadId: data.upload_id,
    partSize: data.part_size,
    parts: data.parts,
  }
}

export interface FinishUploadOptions {
  onProgress?: (progress: UploadProgress) => void
  signal?: AbortSignal
}

/**
 * Uploads every part of `file` directly to S3 using `session`'s presigned
 * URLs, then completes the multipart upload. Aborts the session on the
 * backend if any part fails or the upload is cancelled.
 */
export async function finishUploadSession(
  session: UploadSession,
  file: File,
  token: string,
  options: FinishUploadOptions = {},
): Promise<void> {
  const { onProgress, signal } = options

  let uploadedBytes = 0
  const reportProgress = (partBytes: number) => {
    uploadedBytes += partBytes
    onProgress?.({ uploadedBytes, totalBytes: file.size })
  }

  try {
    const completedParts = await runWithConcurrency(
      session.parts,
      MAX_CONCURRENT_PARTS,
      (part) => uploadPart(session, file, part, reportProgress, signal),
    )

    await completeUploadSession(session, completedParts, token, signal)
  } catch (err) {
    await abortUploadSession(session, token)
    throw err
  }
}

/** Cancels an in-progress upload session, releasing any parts already uploaded to S3. */
export async function abortUploadSession(session: UploadSession, token: string): Promise<void> {
  try {
    await fetch(`${API_BASE_URL}/uploads/abort`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ video_name: session.videoName, upload_id: session.uploadId }),
    })
  } catch {
    // Best-effort cleanup — the caller has already moved on.
  }
}

async function uploadPart(
  session: UploadSession,
  file: File,
  part: UploadPartURL,
  reportProgress: (bytes: number) => void,
  signal?: AbortSignal,
): Promise<{ part_number: number; etag: string }> {
  const start = (part.part_number - 1) * session.partSize
  const end = Math.min(start + session.partSize, file.size)
  const chunk = file.slice(start, end)

  // No Authorization header here — the presigned URL itself carries the
  // credentials for this specific part, and this PUT goes straight to
  // S3/MinIO, not through our backend.
  const res = await fetch(part.url, { method: 'PUT', body: chunk, signal })

  if (!res.ok) {
    throw new Error(`failed to upload part ${part.part_number}: ${res.status}`)
  }

  // S3's bucket CORS config must include ExposeHeaders: ["ETag"], or this
  // will be null and CompleteUpload will fail server-side.
  const etag = res.headers.get('ETag')
  if (!etag) {
    throw new Error(`missing ETag for part ${part.part_number}`)
  }

  reportProgress(end - start)
  return { part_number: part.part_number, etag }
}

async function completeUploadSession(
  session: UploadSession,
  parts: { part_number: number; etag: string }[],
  token: string,
  signal?: AbortSignal,
): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/uploads/complete`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      video_name: session.videoName,
      upload_id: session.uploadId,
      parts: parts.sort((a, b) => a.part_number - b.part_number),
    }),
    signal,
  })

  if (res.status === 401) throw new SessionExpiredError()
  if (!res.ok) {
    throw new Error(`failed to complete upload: ${res.status} ${await res.text()}`)
  }
}

/** Runs `worker` over `items` with at most `limit` calls in flight at once. */
async function runWithConcurrency<T, R>(
  items: T[],
  limit: number,
  worker: (item: T) => Promise<R>,
): Promise<R[]> {
  const results = new Array<R>(items.length)
  let nextIndex = 0

  async function runNext(): Promise<void> {
    const index = nextIndex++
    if (index >= items.length) return
    results[index] = await worker(items[index])
    await runNext()
  }

  await Promise.all(Array.from({ length: Math.min(limit, items.length) }, runNext))
  return results
}
