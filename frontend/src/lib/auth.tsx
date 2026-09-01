import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { API_BASE_URL } from './api'

interface AuthUser {
  email: string
  token: string
}

interface AuthContextValue {
  user: AuthUser | null
  /** False until the stored session (if any) has been checked against the backend. */
  isReady: boolean
  login: (email: string) => Promise<void>
  logout: () => void
}

const STORAGE_KEY = 'vod-coach:user'

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

/** Reads the stored session, rejecting anything that isn't shaped like a
 * current AuthUser — e.g. one saved by an older version of this app before
 * tokens existed. */
function loadStoredUser(): AuthUser | null {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (!raw) return null

  try {
    const parsed = JSON.parse(raw) as Partial<AuthUser>
    if (!parsed.email || !parsed.token) return null
    return { email: parsed.email, token: parsed.token }
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(loadStoredUser)
  // Nothing to verify if there's no stored session, so it's ready immediately.
  const [isReady, setIsReady] = useState(() => loadStoredUser() === null)

  useEffect(() => {
    if (user) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(user))
    } else {
      localStorage.removeItem(STORAGE_KEY)
    }
  }, [user])

  // Confirm any stored session is still valid on load. This is what
  // actually clears out a session from an old flow (no token) or a
  // since-expired/invalidated token, instead of leaving it to fail on
  // every request with a 401.
  useEffect(() => {
    const stored = loadStoredUser()
    if (!stored) return // isReady already true from initial state

    let cancelled = false
    fetch(`${API_BASE_URL}/auth/verify`, {
      headers: { Authorization: `Bearer ${stored.token}` },
    })
      .then((res) => {
        if (cancelled) return
        if (res.status === 401) setUser(null)
        setIsReady(true)
      })
      .catch(() => {
        // Backend unreachable — don't punish the user for that by logging
        // them out; leave the session as-is and let a real request surface it.
        if (!cancelled) setIsReady(true)
      })

    return () => {
      cancelled = true
    }
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isReady,
      login: async (email: string) => {
        const res = await fetch(`${API_BASE_URL}/auth/login`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email }),
        })

        if (!res.ok) {
          throw new Error(`login failed: ${res.status}`)
        }

        const { token }: { token: string } = await res.json()
        setUser({ email, token })
      },
      logout: () => setUser(null),
    }),
    [user, isReady],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
