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

## Project Structure

```
cmd/
  analyzer/     # HTTP service — chi router
  extractor/    # Event-driven — Lambda or standalone binary
  sampler/      # Event-driven — Lambda or standalone binary
internal/
  handlers/     # Handler interface, per-service business logic
  storage/      # S3 client (video streaming, presigned URLs, audio/image uploads)
```

## Running Locally

Event-driven services read a JSON payload from stdin when run outside Lambda:

```bash
# No payload
go run ./cmd/sampler

# With a payload
echo '{"video_id":"abc123"}' | go run ./cmd/sampler
```

The HTTP service:

```bash
go run ./cmd/analyzer        # listens on :8080
PORT=9090 go run ./cmd/analyzer
```
