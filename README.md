# Xendit API Mock

Simple mock for Xendit disbursement API to test retry behavior.

Behavior
- Default (no scenario file): first unique `external_id` returns `FAILED` once, then `COMPLETED`.
- Scenario mode: per-account scripted outcomes (order-based or by exact `external_id`).

Endpoints
- `POST /xendit/disbursements`
- `GET /xendit/healthz`
- `GET /xendit/healthz-callback`
- `POST /xendit/simulate/success`

## Run locally

```bash
cd /path/to/xendit-api-mock
go run .
```

By default it listens on `:8080`.

## Run with Docker

```bash
docker build -t xendit-api-mock .
docker run --env-file .env -p 8080:8080 xendit-api-mock
```

## Run with Docker Compose

```bash
docker compose up --build
```

## Run on Railway

Railway will build using the Dockerfile. Set these variables in Railway:

- `CALLBACK_URL` (required)
- `PORT` (optional, Railway sets this automatically)
- `XENDIT_USER_ID` (optional)

## Expose via ngrok

```bash
ngrok http 8080
```

Copy the public HTTPS URL, for example:
```
https://abcd-1234.ngrok-free.app
```

## Point sandbox to mock

Set the Xendit base URL in sandbox to the ngrok URL:

```
XENDIT__ADDRESS=https://abcd-1234.ngrok-free.app
```

Restart the consumer service in sandbox.

## Callback base URL

Create a local `.env` file (gitignored) to configure the callback target:

```bash
CALLBACK_URL=https://sandbox.example.com/api/v1/it/xendit/disbursement/callback
```

The mock loads `.env` on startup and will POST callbacks to `CALLBACK_URL`.

## Test the retry flow

1. Trigger a topup/disbursement in your normal flow.
2. First disbursement will return `FAILED`.
3. Retry should be created and the next disbursement will return `COMPLETED`.

## Scenario mode (per account)

Use `SCENARIO_FILE` to control responses:

```bash
SCENARIO_FILE=/path/to/xendit-api-mock/scenario.sample.json go run .
```

Rules are matched by `account_number`.

You can also define **batch-specific rules** by `topup_id` + `account_number`.
Batch rules are evaluated before account rules.

Two ways to define rules:
1) **Order-based**: set `external_id` empty and the mock will apply rules in order of requests.
2) **Exact match**: set `external_id` to the specific value to match.

Outcomes:
- `success`
- `fail_then_succeed` with `retry_success_at` (1 = success on first retry)
- `fail_until_timeout` (always FAILED; use to trigger email path)

Example file: `scenario.sample.json`

## Reset mock state

To clear in-memory attempts and ordering:

```bash
curl -X POST http://localhost:8080/xendit/reset
```

## Callback health check

```bash
curl http://localhost:8080/xendit/healthz-callback
```

This endpoint calls the sandbox disbursement callback URL and logs the response.

## Simulate success flow

```bash
curl -X POST http://localhost:8080/xendit/simulate/success
```

Optional JSON body supports the same fields as `/xendit/disbursements`.

## Notes

- The mock keeps state in memory. Restarting the mock resets the `FAILED`-first behavior.
- Order-based rules are per `account_number` in the scenario file.
- You can set a custom port:

```bash
PORT=9090 go run .
ngrok http 9090
```

## Conventional Commits

This repo enforces Conventional Commits on pull requests via GitHub Actions.
