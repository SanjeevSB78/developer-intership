# Learn this project as you build it

This guide explains the project in everyday terms. You do not need to memorize the code. Aim to understand what each part is responsible for and be able to trace one request through it.

## The whole app in one sentence

React displays the screens; the Go server checks requests and applies the rules; MongoDB keeps the lasting records; Redis helps open poll pages hear about new votes quickly.

## Folder map

| Place | What it contains |
| --- | --- |
| `frontend/src/App.jsx` | Page components, forms, and calls to the API |
| `frontend/src/styles.css` | Colors, spacing, typography, responsive layouts, and motion |
| `backend/main.go` | Connects to MongoDB and Redis, then registers the web routes |
| `backend/models.go` | The shapes of accounts, polls, choices, and votes |
| `backend/auth.go` | Signup, login, signed login tokens, and the login check |
| `backend/polls.go` | Poll creation, vote validation, result counts, and live events |
| `docker-compose.yml` | Starts MongoDB, Redis, the API, and the website together |

## Follow one example: casting a vote

1. A voter clicks a choice in `PollPage` in `App.jsx` and presses **Cast your vote**.
2. React sends an HTTP request to `POST /api/polls/{id}/votes`. The request contains the choice ID and a random browser ID.
3. Gin matches that URL to `createVote` in `backend/polls.go`.
4. Go checks that the poll exists, that the choice belongs to that poll, and that the request has a reasonable browser ID.
5. MongoDB saves the vote. Its unique index rejects a second vote from the same browser ID for that poll.
6. Redis increases the selected choice's count and publishes a small vote event.
7. Every open poll page has an SSE connection listening to Redis events. React receives the event and updates the visible results without reloading.

If you can explain those seven steps in your own words, you understand the central feature.

## Plain-language glossary

- **Server/API:** a program that waits for requests from the website and sends back data.
- **Route:** a named URL and action, such as `POST /api/polls` to create a poll.
- **Handler:** the Go function that runs for a route, such as `createPoll`.
- **Middleware:** a check that runs before a handler. `requireUser` checks the login token before poll creation.
- **JSON:** a text format used to send structured data between the browser and the server.
- **Database/document:** MongoDB stores each account, poll, and vote as a record. It is the lasting copy.
- **Cache:** a fast temporary copy. Redis holds the counts used by the live results.
- **Pub/Sub:** Redis sends a message to listeners when a vote arrives. It is like a radio announcement.
- **SSE:** Server-Sent Events keep a browser connection open so the server can send small updates one way. Voting itself still uses a normal HTTP request.
- **Unique index:** a database rule that refuses two records with the same key. Here it limits one browser ID to one saved vote per poll.
- **JWT:** a signed token the browser sends after login to prove who is making a protected request.

## Small Go ideas used here

- `type Poll struct { ... }` defines the fields for one poll. The `bson` labels tell MongoDB how to store fields; `json` labels tell the API what to send to React.
- `func (a *App) createPoll(c *gin.Context)` is a method attached to `App`. `a` gives it access to MongoDB and Redis; `c` contains the request and response.
- `if err != nil { ... }` is Go's usual error handling. Go makes the programmer check errors explicitly.
- `defer thing.Close()` schedules cleanup for when the current function finishes.
- `context.Context` carries cancellation and time limits. If a browser disconnects from the event stream, its context ends and the Redis subscription is closed.

## Why use both MongoDB and Redis?

MongoDB is the durable record: it remembers who voted and lets the unique index reject duplicates. Redis is optimized for the changing live count and broadcasting updates. If Redis is empty for a poll, the API rebuilds its initial counts from MongoDB votes. This is a small demonstration design; a high-traffic production service would also need a more robust recovery mechanism for a MongoDB save followed by a Redis outage.

## What to learn first

1. Read the vote walkthrough above and point at each corresponding file/function.
2. Read `backend/main.go`: find how a route connects to its handler.
3. Read `backend/polls.go`: start at `createVote` and follow the comments and helper calls.
4. In `frontend/src/App.jsx`, find the `vote` function and compare its URL/body with Go's `voteInput` structure.
5. Read `backend/auth.go`: explain why the browser token is checked before `createPoll` can run.
6. Change one small UI detail yourself, such as the landing-page introduction, and rebuild the frontend.

## Interview practice prompts

Try answering these without reading the guide, then check your explanation:

1. What is the difference between MongoDB and Redis in this app?
2. How does the API stop someone from creating a poll while signed out?
3. How does a vote reach other browsers without them refreshing?
4. Why is the unique vote index in MongoDB important even though the browser also stores an ID?
5. What happens if Redis loses its counts?
6. What would you improve before using this with a large public audience?

Be honest about tradeoffs. The browser ID is a lightweight duplicate-vote limit, not strong identity verification: someone can clear browser storage or use another browser. The brief asks for a simple voting flow; stronger anti-abuse controls would need a different product decision.

## The required walkthrough video

The brief asks for a 3–5 minute video. Show the create/share/vote/live flow, explain one challenge and how you solved it, and say which AI tools you used and what help they provided. Review every part of the project before recording so you can explain it in your own words.
