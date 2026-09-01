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
  login: (email: string) => Promise<void>
  logout: () => void
}

const STORAGE_KEY = 'vod-coach:user'

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(() => {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? (JSON.parse(raw) as AuthUser) : null
  })

  useEffect(() => {
    if (user) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(user))
    } else {
      localStorage.removeItem(STORAGE_KEY)
    }
  }, [user])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
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
    [user],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
