import { useState, type SubmitEvent } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../lib/auth'

export function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const from = (location.state as { from?: Location })?.from?.pathname ?? '/'

  async function handleSubmit(e: SubmitEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!email.trim() || isSubmitting) return

    setIsSubmitting(true)
    setError(null)
    try {
      await login(email.trim())
      navigate(from, { replace: true })
    } catch {
      setError('Could not log in — is the backend running?')
      setIsSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-6">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <p className="text-xl font-semibold text-white">VOD Coach</p>
          <p className="mt-1 text-sm text-slate-400">
            Sign in to see your matches and feedback.
          </p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="flex flex-col gap-4 rounded-2xl border border-white/5 bg-white/[0.03] p-6"
        >
          <div className="flex flex-col gap-1.5">
            <label htmlFor="email" className="text-sm text-slate-300">
              Email
            </label>
            <input
              id="email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              className="rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-indigo-400/50 focus:outline-none"
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <label htmlFor="password" className="text-sm text-slate-300">
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              className="rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-indigo-400/50 focus:outline-none"
            />
          </div>

          {error && <p className="text-sm text-red-400">{error}</p>}

          <button
            type="submit"
            disabled={isSubmitting}
            className="mt-2 rounded-lg bg-indigo-500 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-400 disabled:cursor-not-allowed disabled:opacity-40"
          >
            {isSubmitting ? 'Logging in…' : 'Log in'}
          </button>

          <p className="text-center text-xs text-slate-500">
            Demo login — any email works, no account needed yet.
          </p>
        </form>
      </div>
    </div>
  )
}
