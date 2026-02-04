# Xendit API Mock

Simple mock for Xendit disbursement API to test retry behavior.

Behavior
- Default (no scenario file): the very first disbursement request returns `FAILED`. After that, any new `external_id` returns `COMPLETED`, and the same `external_id` always returns `COMPLETED` on subsequent requests.
- Scenario mode: per-account scripted outcomes (order-based or by exact `external_id`).

Endpoints
- `POST /xendit/disbursements`
- `GET /xendit/healthz`
- `GET /xendit/healthz-callback`
- `POST /xendit/simulate/success`
- `POST /xendit/reset`

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
- `RANDOM_STATUS` (optional, set to `true` to randomize COMPLETED/FAILED)

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

How a single request decides SUCCESS/FAILED in scenario mode:
1) Match **batch rules** first (`topup_id` + `account_number`). `topup_id` is compared to the request `description`.
2) If no batch rule applies, match **account rules** by `account_number`.
3) Within a rule list, **exact match** rules (`external_id` set) take priority over order-based rules.
4) If no exact match is found, the next order-based rule is used and its index advances per account/batch key.
5) If no rules match (or the list is exhausted), status defaults to `COMPLETED`.

How outcomes map to status:
- `success` -> `COMPLETED`
- `fail_until_timeout` -> always `FAILED`
- `fail_then_succeed` -> `FAILED` until the attempt count exceeds `retry_success_at`, then `COMPLETED`

Schema:
```json
{
  "retry_timeout_minutes": 60,
  "accounts": [
    {
      "account_number": "123",
      "disbursements": [
        {"external_id": "ext-override", "outcome": "fail_then_succeed", "retry_success_at": 1},
        {"external_id": "", "outcome": "success"}
      ]
    }
  ],
  "batches": [
    {
      "topup_id": "TOPUP-001",
      "account_number": "123",
      "disbursements": [
        {"external_id": "", "outcome": "success"},
        {"external_id": "", "outcome": "fail_then_succeed", "retry_success_at": 1},
        {"external_id": "", "outcome": "fail_until_timeout"}
      ]
    }
  ]
}
```

Notes:
- `retry_timeout_minutes` is parsed and defaulted to 60; time-based timeout is reserved for future behavior.
- `topup_id` matches the disbursement `description` field.
- Exact match rules (`external_id` set) take precedence over order-based rules.
- If a batch rule omits `topup_id`, it matches any description for that account.
- Order-based indices are per account/batch and reset on service restart or `/xendit/reset`.
- If `SCENARIO_FILE` fails to load or parse, the mock falls back to default behavior.

Example file: `scenario.sample.json`

JSON schema: `scenario.schema.json`

### Adding more test case variety
You can introduce more success/failure patterns by adding rules and mixing exact-match and order-based rules:

1) **Exact-match override for a specific external_id**:
```json
{"external_id": "ext-123", "outcome": "fail_then_succeed", "retry_success_at": 2}
```
This will force `ext-123` to fail twice, then succeed.

2) **Order-based sequences** (per account or batch):
```json
[
  {"external_id": "", "outcome": "success"},
  {"external_id": "", "outcome": "fail_then_succeed", "retry_success_at": 1},
  {"external_id": "", "outcome": "fail_until_timeout"}
]
```
This creates a predictable sequence across requests for that account/batch.

3) **Multiple accounts or batches**:
Define different `account_number` entries or multiple `batches` to simulate parallel flows.

4) **Add new outcome types (advanced)**:
If you want new outcome values beyond the three listed above, add support in `internal/scenario/engine.go` and update `scenario.schema.json` to include your new enum value so validation stays in sync.

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
