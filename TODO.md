# TODO

Tracks near-term implementation work on the VOD pipeline outside of an external issue tracker. Update as items complete or scope shifts.

## Done

- [x] Backend + frontend core integration — auth (login/verify), multipart upload flow, video list/detail wired to the backend

## Now

- [ ] Send SQS messages on upload (trigger downstream processing once a video finishes uploading)
- [ ] Add analyzer Lambda logic
- [ ] Add a local Go poller for testing SQS → Lambda analysis end to end

## Next

- [ ] Add databases for video/user metadata (after the core Lambda logic and setup above are done)
- [ ] Integrate MCP tooling for analyzing patterns across multiple videos
