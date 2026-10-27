# Pulse Polls

A live polling app built for the developer internship brief. The React client talks to a Go/Gin API; MongoDB stores accounts, polls, and votes, while Redis maintains live counts and fans out updates over server-sent events (SSE).

## Local development

Requirements: Docker Compose, Go 1.27+, Node.js 24+.

1. Copy `.env.example` to `.env` and change `JWT_SECRET`.
2. Start data services: `docker compose up -d mongo redis`.
3. In `backend`, run `go mod tidy` and `go run .`. The API loads settings from the root `.env` file.
4. In `frontend`, run `npm install` then `npm run dev`. The React app reads `VITE_API_URL` from the root `.env` file.
5. Visit http://localhost:5173.

Or run the full stack with `docker compose up --build` and visit http://localhost:8080.

## Product flow

Create an account, sign in, create a question with 2–6 choices, then share the poll URL. Anyone with the URL can vote. Results stream to open browsers without refreshing. Votes are anonymous and one vote per browser identity per poll; the identity is stored locally by the browser.

## Design decisions

- The API validates and normalizes every submitted field. Only authenticated users can create polls.
- Passwords use bcrypt. API sessions use signed JWT bearer tokens.
- MongoDB is the durable source of truth, with a unique `(pollId, voterId)` index to prevent duplicate votes.
- Redis hashes hold option counts and Redis Pub/Sub carries vote events to SSE connections. Redis count keys are initialized from MongoDB on first access and can be rebuilt from stored votes.
- SSE is used for one-way live results, which suits a vote feed without requiring a bidirectional socket protocol.
- This project is not deployed yet. Add hosting-specific secrets and a public MongoDB/Redis deployment before submitting.

## Deployment checklist

- Deploy the Go API with `MONGODB_URI`, `MONGODB_DATABASE`, `REDIS_URL` (or `REDIS_ADDR`), `JWT_SECRET`, and `FRONTEND_ORIGIN` configured.
- Deploy the static React build and set `VITE_API_URL` to the public API URL. Configure HTTPS and CORS for the exact frontend origin.
- Verify signup, poll creation, cross-browser voting, live updates, and duplicate-vote rejection on the public URLs.
- Record the required 3–5 minute walkthrough: toughest challenge and solution, plus honest disclosure of AI tools used.
- Submit the public repository, live link, and video to the email address in the brief. Do this yourself after reviewing the submission.

## API overview

- `POST /api/auth/signup`, `POST /api/auth/login`, `GET /api/auth/me`
- `POST /api/polls` (authenticated), `GET /api/polls/:id`
- `POST /api/polls/:id/votes`, `GET /api/polls/:id/events` (SSE)

## Next steps

1. Run the app locally and review the full create/share/vote/live flow.
2. Add automated Go and frontend tests, then run them.
3. Deploy the API and frontend, exercise the live path, and record the required video.

## Learning guide

If Go or Gin are new to you, start with [docs/learning-guide.md](docs/learning-guide.md). It follows one vote from the React button through Gin, MongoDB, Redis, and back to the live browser updates.
