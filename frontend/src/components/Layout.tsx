import { Link, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../lib/auth'

export function Layout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  function handleLogout() {
    logout()
    navigate('/login')
  }

  return (
    <div className="min-h-screen">
      <header className="border-b border-white/5">
        <div className="mx-auto flex max-w-4xl items-center justify-between px-6 py-4">
          <Link to="/" className="font-semibold tracking-tight text-white">
            VOD Coach
          </Link>

          {user && (
            <div className="flex items-center gap-3 text-sm text-slate-400">
              <span>{user.email}</span>
              <button
                onClick={handleLogout}
                className="rounded-md border border-white/10 px-3 py-1.5 text-slate-200 transition hover:bg-white/5"
              >
                Log out
              </button>
            </div>
          )}
        </div>
      </header>

      <main className="mx-auto max-w-4xl px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}
