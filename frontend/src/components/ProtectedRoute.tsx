import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '../lib/auth'

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { user, isReady } = useAuth()
  const location = useLocation()

  // Wait for the stored session to be checked against the backend before
  // deciding whether to redirect, so a still-valid session doesn't flash
  // through the login page and a stale one doesn't briefly render protected
  // content.
  if (!isReady) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-slate-400">
        Loading&hellip;
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location }} />
  }

  return <>{children}</>
}
