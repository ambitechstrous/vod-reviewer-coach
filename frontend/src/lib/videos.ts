import type { Video } from '../types'
import { API_BASE_URL, SessionExpiredError } from './api'

/** Fetches the authenticated user's videos from the backend. */
export async function fetchVideos(token: string, signal?: AbortSignal): Promise<Video[]> {
  const res = await fetch(`${API_BASE_URL}/videos`, {
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })

  if (res.status === 401) throw new SessionExpiredError()
  if (!res.ok) {
    throw new Error(`failed to fetch videos: ${res.status} ${await res.text()}`)
  }

  return res.json()
}

/** Fetches a single video's details, or null if no video exists with that ID. */
export async function fetchVideoDetails(
  videoId: string,
  token: string,
  signal?: AbortSignal,
): Promise<Video | null> {
  const res = await fetch(`${API_BASE_URL}/videos/${encodeURIComponent(videoId)}`, {
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })

  if (res.status === 401) throw new SessionExpiredError()
  if (res.status === 404) return null
  if (!res.ok) {
    throw new Error(`failed to fetch video: ${res.status} ${await res.text()}`)
  }

  return res.json()
}
