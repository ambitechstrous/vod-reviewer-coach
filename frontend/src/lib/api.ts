export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

/** Thrown when the backend rejects a request's token (missing, expired, or invalid). */
export class SessionExpiredError extends Error {
  constructor() {
    super('Session expired — please log in again.')
    this.name = 'SessionExpiredError'
  }
}
