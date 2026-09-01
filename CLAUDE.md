# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build all services
go build ./...

# Vet
go vet ./...

# Run tests
go test ./...

# Run sampler (standalone, no payload)
go run ./cmd/sampler

# Run sampler with a payload
echo '{"video_id":"abc123"}' | go run ./cmd/sampler

# Run extractor with a payload
echo '{"match_id":"abc123"}' | go run ./cmd/extractor

# Run analyzer (HTTP, default :8080)
go run ./cmd/analyzer

# Run analyzer on a custom port
PORT=9090 go run ./cmd/analyzer
```

```bash
# Frontend (run from frontend/)
cd frontend
npm install
npm run dev      # Vite dev server, http://localhost:5173
npm run build    # tsc -b && vite build
npm run lint     # oxlint
```

## Architecture

The pipeline is `Upload → Sampler → Extractor → Analyzer → Feedback`. Three independent services share a single Go module (`github.com/ambitechstrous/vod-reviewer-coach`) and common packages under `internal/`.

### Runtime detection pattern

`sampler` and `extractor` are dual-mode: they run as AWS Lambda functions or standalone binaries. The entry point is `handlers.RunHandler` (`internal/handlers/handler.go`), which checks `AWS_LAMBDA_FUNCTION_NAME` at startup — if set, it calls `lambda.Start`; otherwise it reads a JSON payload from stdin and runs synchronously. All business logic lives in a `Handler` interface (`Run(ctx, Event) error`), so neither main nor the handler cares about the runtime.

`analyzer` is a persistent HTTP service (chi router) and does not use `RunHandler`. It listens on `PORT` (default `8080`) with graceful shutdown on `SIGINT`/`SIGTERM`.

### Key packages

- `internal/handlers/` — the `Handler` interface, `RunHandler` dispatcher, and per-service implementations (`SamplerHandler`, `ExtractorHandler`, `Analyze` HTTP handler).
- `internal/storage/` — `S3Client` wrapping AWS SDK v2. Hardcoded bucket is `"user-vods"`. Provides `GetVideo` (streaming), `PutAudio`, `PutImage`, and `PresignGetVideo`.
- `internal/client/` — stub `GeminiClient` for Gemini API interactions (multimodal frame analysis, speech-to-text). Not yet implemented.

### Data flow between services

- **Sampler** receives `{"video_id": "..."}`, streams the video from S3, and is expected to write an audio track and sampled frames back to S3.
- **Extractor** receives `{"match_id": "..."}`, reads the Sampler's outputs, and is expected to produce a timestamped transcript (speech-to-text via Gemini) and a game state stream (multimodal frame analysis via Gemini).
- **Analyzer** receives the transcript + game state stream over HTTP `POST /analyze` and is expected to call an LLM to produce structured coaching feedback.

### Frontend (`frontend/`)

A separate npm project (not part of the Go module) — React 19 + TypeScript, Vite, Tailwind v4, React Router 7. Currently built against mock data (`frontend/src/data/mockVideos.ts`) since there's no video/user API to call yet.

- `src/pages/` — `LandingPage` (video list + chat), `VideoDetailPage`, `LoginPage`.
- `src/components/` — `VideoTile`, `VideoThumbnail`, `StatusBadge`, `ChatPanel`, `Layout`, `ProtectedRoute`.
- `src/lib/auth.tsx` — mock `AuthProvider`/`useAuth` (any email logs in, session kept in `localStorage`); routes are gated via `ProtectedRoute`.
- `src/types.ts` — shared `Video` (`status: 'uploaded' | 'analyzing' | 'processed'`) and `ChatMessage` types.

When the real backend endpoints exist, mock data and the mock auth provider are the pieces to swap out first.
