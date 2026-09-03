import { API_BASE_URL, SessionExpiredError } from './api'

interface ChatResponse {
  answer: string
}

/** Asks the AI coach a question across all of the user's analyzed videos. */
export async function askCoach(
  question: string,
  token: string,
  signal?: AbortSignal,
): Promise<string> {
  const res = await fetch(`${API_BASE_URL}/chat`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ question }),
    signal,
  })

  if (res.status === 401) throw new SessionExpiredError()
  if (!res.ok) {
    throw new Error(`failed to ask coach: ${res.status} ${await res.text()}`)
  }

  const data: ChatResponse = await res.json()
  return data.answer
}
