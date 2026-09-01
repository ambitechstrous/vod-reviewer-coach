const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

// How many parts to upload to S3 in parallel. Multipart upload's main
// benefit over a single PUT is that parts can go out concurrently.
const MAX_CONCURRENT_PARTS = 4

interface UploadPart {
  part_number: number
  url: string
}

interface InitiateUploadResponse {
  key: string
  upload_id: string
  part_size: number
  parts: UploadPart[]
}

export interface UploadProgress {
  uploadedBytes: number
  totalBytes: number
}

export interface UploadVideoOptions {
  onProgress?: (progress: UploadProgress) => void
  signal?: AbortSignal
}

/**
 * Uploads a video file to S3 via the backend's multipart upload endpoints:
 * initiate (get one presigned URL per part) -> PUT each part directly to S3
 * -> complete. If anything fails partway through, the upload is aborted on
 * the backend so S3 doesn't keep billing for orphaned parts.
 *
 * `videoId` becomes the video's S3 key, so it must be unique per upload.
 */
export async function uploadVideo(
  videoId: string,
  file: File,
  options: UploadVideoOptions = {},
): Promise<void> {
  const { onProgress, signal } = options

  const init = await initiateUpload(videoId, file, signal)

  let uploadedBytes = 0
  const reportProgress = (partBytes: number) => {
    uploadedBytes += partBytes
    onProgress?.({ uploadedBytes, totalBytes: file.size })
  }

  try {
    const completedParts = await runWithConcurrency(
      init.parts,
      MAX_CONCURRENT_PARTS,
      (part) => uploadPart(init, file, part, reportProgress, signal),
    )

    await completeUpload(videoId, init.upload_id, completedParts, signal)
  } catch (err) {
    await abortUpload(videoId, init.upload_id)
    throw err
  }
}

async function initiateUpload(
  videoId: string,
  file: File,
  signal?: AbortSignal,
): Promise<InitiateUploadResponse> {
  const res = await fetch(`${API_BASE_URL}/uploads`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      video_id: videoId,
      content_type: file.type || 'application/octet-stream',
      file_size: file.size,
    }),
    signal,
  })

  if (!res.ok) {
    throw new Error(`failed to initiate upload: ${res.status} ${await res.text()}`)
  }

  return res.json()
}

async function uploadPart(
  init: InitiateUploadResponse,
  file: File,
  part: UploadPart,
  reportProgress: (bytes: number) => void,
  signal?: AbortSignal,
): Promise<{ part_number: number; etag: string }> {
  const start = (part.part_number - 1) * init.part_size
  const end = Math.min(start + init.part_size, file.size)
  const chunk = file.slice(start, end)

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

async function completeUpload(
  videoId: string,
  uploadId: string,
  parts: { part_number: number; etag: string }[],
  signal?: AbortSignal,
): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/uploads/complete`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      video_id: videoId,
      upload_id: uploadId,
      parts: parts.sort((a, b) => a.part_number - b.part_number),
    }),
    signal,
  })

  if (!res.ok) {
    throw new Error(`failed to complete upload: ${res.status} ${await res.text()}`)
  }
}

async function abortUpload(videoId: string, uploadId: string): Promise<void> {
  try {
    await fetch(`${API_BASE_URL}/uploads/abort`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ video_id: videoId, upload_id: uploadId }),
    })
  } catch {
    // Best-effort cleanup — the original error is what the caller sees.
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
