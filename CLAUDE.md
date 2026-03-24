# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`github.com/sanservices/kit` is a shared Go utility library providing reusable packages for microservice infrastructure: configuration, database connections, Kafka messaging, logging, TLS, and pprof debugging endpoints.

## Build & Test Commands

```bash
make test              # Run all unit tests
make coverage          # Show test coverage summary
make vet               # Run go vet
make staticcheck       # Run static analysis (install first: make install-dev)
make scan-cli          # Run all analysis tools (vet + staticcheck + tests)

# Single test
go test -v -run TestFunctionName ./package/...

# Build check
go build -v ./...
```

## Architecture

The library is organized as independent packages, each providing a specific infrastructure concern:

- **`bundle/`** — Service locator pattern. `bundle.All` holds initialized services (logger, middleware, etc.) for centralized access.
- **`config/`** — Viper-based JSON config reader. Loads `config/{env}.json` based on environment enum (`Local`, `Test`, `Dev`, `Prod`). Supports env var overrides via `viper.AutomaticEnv()`.
- **`database/`** — Factory functions (`CreateMySqlConnection`, `CreateOracleConnection`, `CreateSqliteConnection`, `CreateRedisConnection`) returning `*sqlx.DB` or `*redis.Client`. Redis supports Sentinel.
- **`kafkalistener/`** — Kafka producer/consumer built on Watermill + Sarama. Integrates with Confluent Schema Registry for Avro encoding. Message wire format: `[0x00][4-byte schema ID BE][Avro payload]`. Includes exponential backoff retry middleware.
- **`log/`** — Global `*zap.SugaredLogger` initialized from config. Uses package-level `Logger()` accessor.
- **`pprofdebug/`** — Wraps Go's `net/http/pprof` handlers as Echo framework route handlers via `WrapGroup()`.
- **`tls/`** — TLS config builder supporting client certs, CA verification, and skip-verify. Provides `GetHTTPSClient()`.

## Testing Patterns

- Table-driven tests with `t.Run()` subtests
- `TestMain()` used for setup/teardown (e.g., SQLite file cleanup in `database/`)
- Error validation uses custom error types (e.g., `ErrInvalidDBName`), not string matching

## Key Dependencies

- **Web framework:** Echo v4
- **Config:** Viper
- **Logging:** Zap (SugaredLogger)
- **SQL:** sqlx + MySQL/Oracle/SQLite drivers
- **Kafka:** Sarama (via Watermill-Kafka adapter)
- **Avro:** hamba/avro
