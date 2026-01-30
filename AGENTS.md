# PROJECT KNOWLEDGE BASE

**Overview:** Single-binary Go mock for Xendit disbursement flows with configurable retry scenarios.

## STRUCTURE
```
./
├── main.go               # HTTP server + mock behavior
├── scenario.sample.json  # Scenario examples
├── Dockerfile            # Container build
├── docker-compose.yml    # Local compose run
├── railway.toml          # Railway deployment config
└── README.md             # Usage and env docs
```

## WHERE TO LOOK
- Core behavior and handlers: `main.go`
- Scenario rules and outcome logic: `main.go`
- Runtime/env setup: `README.md`
- Container/runtime: `Dockerfile`, `docker-compose.yml`, `railway.toml`

## CONVENTIONS
- Keep the service single-file unless complexity grows.
- Load env from `.env` locally; do not commit `.env`.
- Callback posts to `$CALLBACK_BASE_URL/api/v1/it/xendit/disbursement/callback`.

## PULL REQUEST RULES
- PR titles and commit messages must follow Conventional Commits.
  Examples: `feat: add scenario override`, `fix: handle missing callback base url`.

## COMMANDS
```bash
go run .
docker build -t xendit-api-mock .
docker run --env-file .env -p 8080:8080 xendit-api-mock
docker compose up --build
```

## NOTES
- In-memory state resets on restart; use `POST /reset` to clear state.
