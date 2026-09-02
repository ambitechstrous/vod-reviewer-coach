# vod-reviewer-coach

An AI pipeline that reviews gameplay VODs and coaches players on habits and decisions across games. A video is uploaded, processed through three sequential services, and the player receives structured feedback on how to improve.

## Architecture

The backend is split across four independent Go services under `cmd/`, sharing a single Go module and common packages under `internal/`. `sampler`, `extractor`, and `analyzer` are event-driven and each can run as an AWS Lambda or a standalone binary — the entry point detects the runtime automatically. `server` is the one persistent HTTP service; it's not a pipeline stage itself, it's the API the frontend talks to directly (auth, uploads, video listing), and it's what accepts the initial upload and will eventually serve the finished feedback back to the player.

```
Upload → Sampler → Extractor → Analyzer → Feedback
```

---

## Services

### Server (`cmd/server`)

**Runtime:** persistent HTTP service (chi router)

The backend API the frontend talks to directly. Not itself a pipeline stage — it's the entry/exit point around it:

- **Auth** — `POST /auth/login` issues a session token; `GET /auth/verify` confirms one is still valid
- **Uploads** — `POST /uploads`, `/uploads/complete`, `/uploads/abort` drive the multipart upload flow (presigned S3 part URLs); completing an upload writes a `metadata.json` sidecar alongside the video
- **Videos** — `GET /videos` and `GET /videos/{videoID}` list and fetch a user's videos from S3, reading each video's `metadata.json`

---

### Sampler (`cmd/sampler`)

**Input:** a video file reference (S3 key + video ID)

Takes a raw uploaded VOD and prepares it for AI processing. Video files are too large and dense to feed directly into multimodal models, so the sampler breaks them down into two things:

- **Audio track** — extracted from the video and stored as a standalone audio file, to be used for communication analysis (voice comms, callouts, etc.)
- **Frame samples** — a set of images sampled at a regular interval across the video, giving the downstream model a visual timeline of the game without processing every frame

**Output:** an audio file and a collection of sampled frames, written back to S3.

---

### Extractor (`cmd/extractor`)

**Input:** the audio file and sampled frames produced by the Sampler

Takes the Sampler's output and extracts structured, timestamped information from it:

- **Transcript** — the audio file is run through a speech-to-text model to produce a timestamped transcript of all comms during the match
- **Game state stream** — the sampled frames are fed into a multimodal model to produce a timestamped description of the game: player and teammate positions, rotations, and whatever enemy position information is visible on screen

**Output:** a timestamped transcript and a timestamped game state stream, passed to the Analyzer.

---

### Analyzer (`cmd/analyzer`)

**Input:** `{"video_key" | "file_path", "prompt"}` — an S3 key (prod) or local path (testing), plus the prompt to analyze the video against

**Runtime:** event-driven — Lambda or standalone binary (like `sampler`/`extractor`)

The final stage. Takes a video and a prompt, and feeds it into Gemini to produce coaching feedback:

- Uploads the video to Gemini's File API and polls until it's processed
- Generates a response from the given prompt (decision-making, positioning, communication patterns, etc. — the prompt itself isn't fixed yet)
- Currently just logs the resulting text; turning that into a structured report delivered back to the player, and eventually reading directly from the transcript + game state stream instead of a raw video, is tracked in `TODO.md`

**Output:** coaching feedback (currently logged, not yet persisted or delivered).

**Status:** the Lambda/standalone handler itself is implemented (`AnalyzerHandler` calls Gemini directly). Still open: wiring a real SQS trigger on upload, a local poller to test that flow end to end, and consuming the Extractor's transcript/game-state output instead of a raw video — see `TODO.md`.

---

### Frontend (`frontend/`)

A React + TypeScript single-page app (Vite, Tailwind v4, React Router) that sits in front of the pipeline, talking to `server`. Auth, uploads, and video list/detail are wired to real backend endpoints — most video metadata (game, duration, thumbnail) is still placeholder data pending the database work in `TODO.md`, since it isn't tracked anywhere yet.

- **Landing page** — list of uploaded videos (thumbnail + status: `uploaded` / `analyzing` / `processed`) with a chat panel below for asking an AI coach about gameplay.
- **Video detail page** — video player, title, status, and the coaching summary once analysis is `processed`.
- **Login** — real JWT session issued by `server` (any email logs in — there's no password check yet), gating the app and scoping S3 keys to that user.

---

## Project Structure

```
backend/
  cmd/
    server/       # HTTP service — chi router. Primary backend server for application.
    analyzer/     # Event-driven — Lambda or standalone binary. Runs core analysis logic.
    extractor/    # Event-driven — Lambda or standalone binary
    sampler/      # Event-driven — Lambda or standalone binary
  internal/
    api/          # server's HTTP handlers: auth, uploads, video list/detail, middleware
    auth/         # JWT session token issuing/verification
    client/       # GeminiClient — Gemini File API video analysis
    handlers/     # Handler interface + RunHandler dispatcher, pipeline handlers (sampler/extractor/analyzer)
    storage/      # S3 client (video streaming, presigned URLs, multipart uploads, audio/image uploads)
frontend/         # React + Vite + TypeScript SPA
```

## Local Setup

Everything below assumes you're starting from the repo root. There are three pieces to get running: the frontend, the backend, and (if you want uploads to actually work end to end) a local S3-compatible store via MinIO.

### 1. Frontend

```bash
cd frontend
npm install
npm run dev           # http://localhost:5173
```

By default it talks to the backend at `http://localhost:8080`. Override that with `frontend/.env`:

```
VITE_API_BASE_URL=http://localhost:8080
```

### 2. Backend

Requires Go 1.26+. Config comes from environment variables — create `backend/.env` (already gitignored):

```
# Gemini — required by cmd/analyzer
GEMINI_API_KEY=your-gemini-api-key

# S3 / MinIO — see the MinIO section below
AWS_REGION=us-east-1
ENVIRONMENT=development
MINIO_USER=admin
MINIO_PASSWORD=<your password here>

# Origins allowed to call the API (comma-separated)
CORS_ALLOWED_ORIGINS=http://localhost:5173

# Signs session tokens issued by /auth/login — any random string works
# locally, but keep it secret in real environments (anyone who has it can
# mint a valid token for any user).
JWT_SECRET=<a long random string>
```

Then run the HTTP service the frontend talks to:

```bash
cd backend
go run ./cmd/server        # listens on :8080
PORT=9090 go run ./cmd/server   # or a different port
```

`sampler`, `extractor`, and `analyzer` aren't wired into the frontend yet. All three can be run standalone, reading a JSON payload from stdin:

```bash
echo '{"video_id":"abc123"}' | go run ./cmd/sampler
echo '{"match_id":"abc123"}' | go run ./cmd/extractor
echo '{"file_path":"/path/to/video.mp4","prompt":"..."}' | go run ./cmd/analyzer   # requires GEMINI_API_KEY
```

`sampler` and `extractor` are still placeholder stubs (see `TODO.md`). `analyzer` actually calls Gemini to analyze the given video, but nothing upstream triggers it yet — no SQS wiring, and it isn't reading the Extractor's transcript/game-state output.

### 3. MinIO (S3 emulation)

The backend talks to storage through the AWS S3 SDK, which MinIO also speaks, so pointing it at a local MinIO instance instead of real AWS is just a matter of setting the `ENVIRONMENT` and `AWS_REGION` env vars to `DEVELOPMENT` and `us-east-1` respectively — no code changes needed. 

Run MinIO via Docker:

```bash
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=<choose-your-password> \
  minio/minio server /data --console-address ":9001"
```

- API: `http://localhost:9000` — this is what `AWS_ENDPOINT_URL` points at
- Console: `http://localhost:9001` — log in with `minioadmin` / `minioadmin`

Create the bucket the backend expects. The bucket name (`user-vods`) is hardcoded in `backend/internal/storage/s3.go`:

```bash
mc alias set local http://localhost:9000 minioadmin <choose-your-password>
mc mb local/user-vods
```

(Or create it from the console at `http://localhost:9001` instead of using `mc`.)

**Multipart uploads also need CORS enabled on the bucket.** The upload panel PUTs each part directly from the browser to S3/MinIO, bypassing the backend entirely, so the bucket itself — not the Go server — has to allow cross-origin requests from the frontend and expose the `ETag` header the frontend reads back after each part:

```json
{
  "CORSRules": [
    {
      "AllowedOrigins": ["http://localhost:5173"],
      "AllowedMethods": ["PUT", "GET"],
      "AllowedHeaders": ["*"],
      "ExposeHeaders": ["ETag"]
    }
  ]
}
```

```bash
mc cors set local/user-vods cors.json   # flags vary by mc version — check `mc cors --help`
```

After doing the above set up, you can finally run `docker start minio` to spin up the container. To stop minio, simply run `docker stop minio`.

With MinIO, the backend, and the frontend all running, dropping a video into the upload panel drives a real multipart upload against your local bucket.
