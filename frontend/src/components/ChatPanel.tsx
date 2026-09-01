import { useRef, useState, type SubmitEvent } from 'react'
import type { ChatMessage } from '../types'

const MOCK_REPLIES = [
  "Looking at your recent VODs, your biggest recurring issue is over-peeking after a teammate dies nearby. Try holding the angle instead and letting them re-engage.",
  "Your communication drops off noticeably in the second half of matches. Calling out rotations even when unsure helps your team react faster.",
  "Economy management has been inconsistent — you've force-bought after lost pistols in two of your last three matches. Saving there would set up a stronger bonus round.",
  "That's a good question. Once more VODs are processed I'll be able to compare trends across matches, not just within one.",
]

let nextReplyIndex = 0

export function ChatPanel() {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [draft, setDraft] = useState('')
  const [isThinking, setIsThinking] = useState(false)
  const idRef = useRef(0)

  function nextId() {
    idRef.current += 1
    return `msg-${idRef.current}`
  }

  function handleSubmit(e: SubmitEvent<HTMLFormElement>) {
    e.preventDefault()
    const text = draft.trim()
    if (!text) return

    setMessages((prev) => [...prev, { id: nextId(), role: 'user', text }])
    setDraft('')
    setIsThinking(true)

    setTimeout(() => {
      const reply = MOCK_REPLIES[nextReplyIndex % MOCK_REPLIES.length]
      nextReplyIndex += 1
      setMessages((prev) => [
        ...prev,
        { id: nextId(), role: 'assistant', text: reply },
      ])
      setIsThinking(false)
    }, 700)
  }

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col rounded-2xl border border-white/5 bg-white/[0.03]">
      <div className="border-b border-white/5 px-5 py-3">
        <p className="font-medium text-white">Ask your coach</p>
        <p className="text-sm text-slate-400">
          Ask about habits, patterns, or specific matches.
        </p>
      </div>

      <div className="flex max-h-80 min-h-24 flex-col gap-3 overflow-y-auto px-5 py-4">
        {messages.length === 0 && !isThinking && (
          <p className="text-sm text-slate-500">
            e.g. "What's the one thing I should fix this week?"
          </p>
        )}

        {messages.map((m) => (
          <div
            key={m.id}
            className={`max-w-[85%] rounded-xl px-3.5 py-2 text-sm ${
              m.role === 'user'
                ? 'ml-auto bg-indigo-500/90 text-white'
                : 'bg-slate-800 text-slate-100'
            }`}
          >
            {m.text}
          </div>
        ))}

        {isThinking && (
          <div className="max-w-[85%] rounded-xl bg-slate-800 px-3.5 py-2 text-sm text-slate-400">
            Thinking&hellip;
          </div>
        )}
      </div>

      <form
        onSubmit={handleSubmit}
        className="flex items-center gap-2 border-t border-white/5 p-3"
      >
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="Ask about your gameplay..."
          className="flex-1 rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-indigo-400/50 focus:outline-none"
        />
        <button
          type="submit"
          disabled={!draft.trim()}
          className="rounded-lg bg-indigo-500 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-400 disabled:cursor-not-allowed disabled:opacity-40"
        >
          Send
        </button>
      </form>
    </div>
  )
}
