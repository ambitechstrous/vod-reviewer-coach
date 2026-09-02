# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# All Go commands run from backend/ — that's where go.mod lives
cd backend

# Build all services
go build ./...

# Vet
go vet ./...

# Run tests
go test ./...

# Run the server (HTTP, default :8080) — the API the frontend talks to
go run ./cmd/server

# Run the server on a custom port
PORT=9090 go run ./cmd/server

# Run sampler (standalone, no payload)
go run ./cmd/sampler

# Run sampler with a payload
echo '{"video_id":"abc123"}' | go run ./cmd/sampler

# Run extractor with a payload
echo '{"match_id":"abc123"}' | go run ./cmd/extractor

# Run analyzer with a payload (local file, bypassing S3)
echo '{"file_path":"/path/to/video.mp4","prompt":"..."}' | go run ./cmd/analyzer
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

The Go module (`github.com/ambitechstrous/vod-reviewer-coach`) lives under `backend/`. Four independent services share it and common packages under `backend/internal/`: `server` is the persistent HTTP API the frontend talks to (auth, uploads, video listing); `sampler`, `extractor`, and `analyzer` are the event-driven pipeline stages.

```
Upload → Sampler → Extractor → Analyzer → Feedback
```

### Runtime detection pattern

`sampler`, `extractor`, and `analyzer` are dual-mode: they run as AWS Lambda functions or standalone binaries. The entry point is `handlers.RunHandler` (`backend/internal/handlers/handler.go`), which checks `AWS_LAMBDA_FUNCTION_NAME` at startup — if set, it calls `lambda.Start`; otherwise it reads a JSON payload from stdin and runs synchronously. All business logic lives in a `Handler` interface (`Run(ctx, Event) error`), so neither main nor the handler cares about the runtime.

`server` is the one persistent HTTP service (chi router) and does not use `RunHandler`. It listens on `PORT` (default `8080`) with graceful shutdown on `SIGINT`/`SIGTERM`.

### Key packages

- `internal/handlers/` — the `Handler` interface, `RunHandler` dispatcher, and the event-driven pipeline handlers: `SamplerHandler`, `ExtractorHandler`, `AnalyzerHandler`.
- `internal/api/` — the `server` HTTP handler: router setup (`http.go`), auth endpoints (`auth.go`), multipart upload endpoints (`upload.go`), video list/detail endpoints (`videos.go`), and the `requireAuth` middleware (`middleware.go`).
- `internal/auth/` — issues and verifies signed JWT session tokens (`IssueToken`/`VerifyToken`) and carries the authenticated user ID through request context. No password/identity check yet — `JWT_SECRET` env var required.
- `internal/storage/` — `S3Client` wrapping AWS SDK v2. Hardcoded bucket is `"user-vods"`. Videos are stored per-user as `{userID}/{videoID}/video.mp4` with a sidecar `{userID}/{videoID}/metadata.json`. Provides object CRUD/listing, `GetVideo` (streaming), `PutAudio`/`PutImage`, presigned GET URLs, and the multipart upload flow (create/presign-part/complete/abort).
- `internal/client/` — `GeminiClient` for Gemini API interactions. `AnalyzeVideo` uploads a video to Gemini's File API, polls until processed, and generates a response from a prompt. Speech-to-text / multimodal frame analysis for the extractor is not yet implemented.

### Data flow between services

- **Sampler** receives `{"video_id": "..."}`, streams the video from S3, and is expected to write an audio track and sampled frames back to S3.
- **Extractor** receives `{"match_id": "..."}`, reads the Sampler's outputs, and is expected to produce a timestamped transcript (speech-to-text via Gemini) and a game state stream (multimodal frame analysis via Gemini).
- **Analyzer** receives `{"video_key" | "file_path", "prompt"}`, and calls Gemini's video analysis to produce a text result (currently just logged — turning that into structured coaching feedback delivered back to the player is still open, see `TODO.md`).

### Frontend (`frontend/`)

A separate npm project (not part of the Go module) — React 19 + TypeScript, Vite, Tailwind v4, React Router 7. Auth, uploads, and video list/detail are wired to the real `server` API (no mock data remaining); most video metadata (game, duration, thumbnail) is still placeholder/hardcoded pending the database work in `TODO.md`.

- `src/pages/` — `LandingPage` (video list + chat), `VideoDetailPage`, `LoginPage`.
- `src/components/` — `VideoTile`, `VideoThumbnail`, `StatusBadge`, `ChatPanel`, `UploadVideoPanel`, `Layout`, `ProtectedRoute`.
- `src/lib/auth.tsx` — `AuthProvider`/`useAuth` backed by real `POST /auth/login` + `GET /auth/verify` calls (any email still logs in — no password check server-side yet); session (email + JWT) kept in `localStorage`. Routes are gated via `ProtectedRoute`.
- `src/lib/api.ts` — `API_BASE_URL` (from `VITE_API_BASE_URL`) and `SessionExpiredError`.
- `src/lib/videos.ts` / `src/lib/upload.ts` — fetch video list/details and drive the multipart upload flow against `server`.
- `src/types.ts` — shared `Video` (`status: 'uploaded' | 'analyzing' | 'processed'`) and `ChatMessage` types.
