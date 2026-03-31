# Wallet Transfer Service

## Overview

This project implements a wallet-to-wallet transfer service using **Go (Gin)** and **PostgreSQL (GORM)**.

The system ensures:
- Idempotent request handling
- Double-entry ledger consistency
- Safe concurrent execution
- Transactional integrity

---

## Tech Stack

- Go (Golang)
- Gin (HTTP framework)
- GORM (ORM)
- PostgreSQL

---

## Features

- `POST /transfers` endpoint
- Idempotency using idempotency keys
- Double-entry ledger (DEBIT & CREDIT)
- Transaction-safe balance updates
- Transfer state management:
  - PENDING
  - PROCESSED
  - FAILED
- Concurrency-safe execution using DB transactions

---

## AI Usage Disclosure

I used AI tools as an assistant during this assignment to support my development process.

### Where AI was used

* Refining the database schema for wallets, transfers, ledger entries, and idempotency records
* Validating edge cases for idempotency and concurrency handling
* Generating ideas for test scenarios and improving documentation clarity

### Where AI was NOT blindly used

* The core business logic (transfer workflow, transaction handling, idempotency flow) was implemented and validated by me
* I manually reviewed, modified, and tested all generated suggestions
* I ensured the system behavior matches the assignment requirements, especially around:

  * exactly-once semantics
  * double-entry ledger consistency
  * safe concurrent execution

### Example prompts used

* "Design a wallet transfer service with idempotency and concurrency safety using Go and PostgreSQL"
* "How to implement row-level locking and transactional safety in GORM?"
* "What test cases should be included for idempotent APIs?"

I understand the system end-to-end and can explain all design decisions, trade-offs, and implementation details.
