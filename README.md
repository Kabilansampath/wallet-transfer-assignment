# Wallet Transfer Service (Minimal Assignment Version)

This version keeps only the assignment-required API:

- `POST /transfers`

No optional APIs are included.

## What this implements

- Idempotent request handling using `idempotency_records`
- Double-entry ledger (`DEBIT` + `CREDIT`)
- Stored wallet balance updates inside one DB transaction
- Transfer states: `PENDING`, `PROCESSED`, `FAILED`
- Concurrency safety using PostgreSQL row locks (`FOR UPDATE` through GORM locking clause)
- Clean layering: handler / service / repository / domain

## Run

```bash
export DATABASE_URL='host=localhost user=postgres password=postgres dbname=wallet_transfer port=5432 sslmode=disable'
psql "$DATABASE_URL" -f migrations/001_init.sql
go mod tidy
go run ./cmd/server
```

## Seed wallets for manual testing

```sql
INSERT INTO wallets(id, balance) VALUES ('wallet_1', 1000), ('wallet_2', 500);
```

## Request

```bash
curl -X POST http://localhost:8080/transfers \
  -H 'Content-Type: application/json' \
  -d '{
    "idempotencyKey": "abc123",
    "fromWalletId": "wallet_1",
    "toWalletId": "wallet_2",
    "amount": 100
  }'
```

## Behavior

- Same `idempotencyKey` + same payload -> returns original result
- Same `idempotencyKey` + different payload -> `409 Conflict`
- Insufficient funds -> transfer becomes `FAILED`, response replayed on retry
- Missing wallet -> transfer becomes `FAILED`, response replayed on retry
- Success -> transfer becomes `PROCESSED` with exactly 2 ledger entries
