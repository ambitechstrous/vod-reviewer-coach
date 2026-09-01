# vod-reviewer-coach

An AI pipeline that reviews gameplay VODs and coaches players on habits and decisions across games. A video is uploaded, processed through three sequential services, and the player receives structured feedback on how to improve.

## Architecture

The pipeline is built as three independent Go services under `cmd/`. They share a single Go module and common packages under `internal/`. Each event-driven service (`sampler`, `extractor`) can run as an AWS Lambda or a standalone binary — the entry point detects the runtime automatically. The `analyzer` runs as a persistent HTTP service.

```
Upload → Sampler → Extractor → Analyzer → Feedback
```

---

## Services

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

**Input:** the transcript and game state stream from the Extractor

**Runtime:** HTTP service (chi router)

The final stage. Takes the full picture of what happened in the match — what was said, where everyone was, and how the game unfolded — and feeds it into an LLM to produce structured coaching feedback:

- Evaluates decision-making, positioning, and communication patterns
- Identifies recurring habits (positive and negative) across the match
- Produces concrete, actionable suggestions for improvement

**Output:** a coaching report delivered back to the player.

---

### Frontend (`frontend/`)

A React + TypeScript single-page app (Vite, Tailwind v4, React Router) that will sit in front of the pipeline. It is currently scaffolded against mock data while the backend's video/user schemas are still being built out.

- **Landing page** — list of uploaded videos (thumbnail + status: `uploaded` / `analyzing` / `processed`) with a chat panel below for asking an AI coach about gameplay.
- **Video detail page** — video player, title, status, and the coaching summary once analysis is `processed`.
- **Login** — mock auth (any email logs in) gating the app, standing in for real user/video mapping.

---

## Project Structure

```
cmd/
  server/       # HTTP service — chi router. Primary backend server for application.
  analyzer/     # Event-driven — Lambda or standalone binary. Runs core analysis logic.
  extractor/    # Event-driven — Lambda or standalone binary
  sampler/      # Event-driven — Lambda or standalone binary
internal/
  handlers/     # Handler interface, per-service business logic
  storage/      # S3 client (video streaming, presigned URLs, audio/image uploads)
frontend/       # React + Vite + TypeScript SPA (mock data for now)
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
# Gemini — required by the analyzer's /analyze endpoint
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
go run ./cmd/analyzer        # listens on :8080
PORT=9090 go run ./cmd/analyzer   # or a different port
```

`sampler` and `extractor` aren't wired into the frontend yet, but run the same way, standalone, reading a JSON payload from stdin:

```bash
echo '{"video_id":"abc123"}' | go run ./cmd/sampler
echo '{"match_id":"abc123"}' | go run ./cmd/extractor
```

### 3. MinIO (S3 emulation)

The backend talks to storage through the AWS S3 SDK, which MinIO also speaks, so pointing it at a local MinIO instance instead of real AWS is just the `AWS_ENDPOINT_URL` and `S3_FORCE_PATH_STYLE` env vars already shown above — no code changes needed. (`S3_FORCE_PATH_STYLE=true` matters: MinIO expects path-style requests — `host/bucket/key` — rather than the virtual-hosted-style, `bucket.host/key`, the AWS SDK defaults to.)

Run MinIO via Docker:

```bash
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=<choose your password> \
  minio/minio server /data --console-address ":9001"
```

- API: `http://localhost:9000` — this is what `AWS_ENDPOINT_URL` points at
- Console: `http://localhost:9001` — log in with `minioadmin` / `minioadmin`

Create the bucket the backend expects. The bucket name (`user-vods`) is hardcoded in `backend/internal/storage/s3.go`:

```bash
mc alias set local http://localhost:9000 minioadmin minioadmin
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
