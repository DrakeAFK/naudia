# Contributing

Naudia is designed around deterministic vault operations, reviewable proposals, and rollback safety.

Before changing behavior that writes files, add or update tests that cover:

- stale hash rejection
- clean rollback
- rollback after unrelated manual edits
- rollback conflict after same-section edits
- move, rename, or delete rollback safety when applicable

Run:

```sh
go test ./...
go build ./...
```

Do not add network services, telemetry, or cloud AI dependencies without an explicit opt-in design.

