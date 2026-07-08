# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`kit` (module `github.com/sanservices/kit`) is a shared Go library of boilerplate/infrastructure packages consumed by multiple internal services at sanservices — not a standalone application. It has no `main` package. Each top-level directory is an independently importable package covering one cross-cutting concern (config, logging, database connections, Kafka messaging, TLS, pprof debugging). Changes here ripple into every downstream service that imports the package being touched, so keep exported APIs backwards compatible where possible and check the package's own `README.md` (each package has one) for its intended usage pattern before changing its public interface.

## Common commands

```sh
make test                # go test ./...
make coverage             # go test ./... -json -cover
make vet                  # go vet ./...
make staticcheck          # staticcheck -checks=all ./... (requires: make install-dev)
make scan-cli              # runs test, coverage, vet, staticcheck together
make scan                  # same four checks, output saved to ./report/
```

Run a single test:
```sh
go test ./kafkalistener/... -run TestName -v
go test ./config/...
```

There is no build/lint step beyond `go vet`/`staticcheck` — this is a library, not a deployable binary.

## Architecture

### Package map and dependency shape

- **config**: reads a JSON file (`config/config/{env}.json`) via Viper into `config.General`, exposed through the package-level `config.CFG` var and `config.Read(env)`. `config.Env` is one of `local|test|dev|prod`. `config.General`/`config.Info` only defines the generic `info` section (endpoint, port, log path/level) — services embed/extend this shape in their own config structs.
- **log**: wraps `zap` (`*zap.SugaredLogger`). `log.Initialize(cfg *config.Info)` must be called once before `log.Logger()` returns a usable logger; the sugared logger routes by level (debug vs error+) into stdout via a `zapcore.Tee`.
- **bundle**: the composition root. `bundle.All.Initialize(cfg *config.General)` wires up `config` + `log` into `bundle.Services` (currently just `Log`). Downstream services are expected to call this once at startup, then extend `bundle.All`/`bundle.Services` with their own service-specific fields as needed.
- **database**: connection factories only (no ORM/query layer) — `CreateMySqlConnection`, `CreateOracleConnection`, `CreateSqliteConnection` return a `*sqlx.DB`; `CreateRedisConnection` returns a `*redis.Client` (supports Redis Sentinel via `SentielConfig`). Each takes a `DatabaseConfig`/`RedisConfig` struct with yaml tags.
- **tls**: `tls.GetTLSConf(TLS)` builds a `*tls.Config` from CA/cert/key PEM file paths; `tls.GetHTTPSClient` wraps it into an `*http.Client`. Required by `kafkalistener`, which only supports TLS-enabled brokers.
- **kafkalistener**: the largest package — a Kafka pub/sub layer built on `IBM/sarama` + `ThreeDotsLabs/watermill` + `watermill-kafka`, with Avro schema-registry integration (`hamba/avro`).
  - `kafkalistener.New(ctx, *KafkaConfig, debug)` builds a `*MessageBroker`: configures TLS, sarama (`setSaramaConfig` — session/heartbeat timeouts, fetch size caps, idempotent producer, `ChannelBufferSize`), a watermill publisher (skipped when `ConsumeOnly`), and a schema-registry client.
  - `KafkaConfig.GroupInstanceID` enables Kafka static group membership (KIP-345) so a pod restarting within `SessionTimeout` skips a rebalance — must be stable/unique per replica (e.g. StatefulSet pod name); `SessionTimeout` defaults to 45s and drives the heartbeat interval (`SessionTimeout/3`).
  - `Topic` carries the Avro schema; `MessageBroker.SetSchema` fetches/registers it against the schema registry before use. `Publish` prefixes the Avro-encoded payload with the Confluent wire-format magic byte + 4-byte schema ID. `DecodePayload` strips that same 5-byte prefix before unmarshaling.
  - `router.go` wires `RouteHandler{Topic, HandlerFunc}` entries into a watermill `message.Router`; `Listen` is blocking, `Stop` closes it gracefully.
  - `retry.go` is a deliberately forked/modified copy of watermill's retry middleware — it adds `AckAfterMaxRetries` (ack-and-drop vs. nack-forever after exhausting retries), which upstream watermill doesn't support. Don't replace it with the upstream middleware without preserving that behavior.
- **kafkaqueue**: a separate, decoupled buffering layer (not wired into `kafkalistener`) providing an at-least-once local queue in front of downstream processing. Three interchangeable implementations sharing an implicit `Push/Pop/Len/Close` shape: `MemoryQueue` (bounded channel+slice buffer), `RedisQueue` (Redis list via `RPUSH`/`BLPOP`, keyed per-hostname), and `HybridQueue` (wraps both — prefers Redis, auto-fails over to memory on Redis errors, periodically health-checks and drains the memory buffer back to Redis once Redis recovers).
- **pprofdebug**: registers standard `net/http/pprof` endpoints (`heap`, `goroutine`, `profile`, `trace`, etc.) onto an `echo.Group` via `pprofdebug.WrapGroup(prefix, group)`.

### Typical consumption pattern (from README)

```go
cfg := config.Read(config.Env(configEnv))
b := new(bundle.All)
b.Initialize(cfg)
```

Kafka usage layers `kafkalistener.New` → register `RouteHandler`s → `SetRetry` → `Listen` (blocking); see `kafkalistener/README.md` for the full example including `DecodePayload` inside a handler.

## Notes for making changes

- Config structs across packages consistently use `yaml` struct tags even where the actual file is JSON (see `config/config/*.json` vs. Viper unmarshaling) — follow existing tag conventions per struct rather than introducing `json` tags unless the package already uses them (e.g. `kafkaqueue.RedisConfig` uses both).
- Packages log through different mechanisms inconsistently (`log/log.go` uses `zap`, `kafkaqueue` uses `log/slog`, `tls`/`database`/`kafkalistener.New` use the standard `log` package) — match whatever the file you're editing already uses rather than unifying them as a drive-by change.
- Sarama tuning knobs in `kafkalistener.setSaramaConfig` (channel buffer size, fetch min/default/max, session/heartbeat timeouts) exist to bound memory usage and consumer-group stability; there are exported setters (`SetChannelBufferSize`, `SetConsumerFetchDefault`, etc.) for callers to override post-construction — prefer adding a setter over changing the hardcoded default if a service needs different behavior.
