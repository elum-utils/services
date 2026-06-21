# Control benchmarks

`benchmark_test.go` measures every domain method of `Admin` and `Internal`, plus the `Control` lifecycle (`NewWithDatabase`, `IsReady`, `Close`, `Run`), against a real MySQL database. It does not include REST, adapters, or network calls to external identity providers.

The MySQL credentials follow the existing test configuration from the old `control` service and are set directly in `benchmark_test.go`. The suite uses the dedicated `control_bench` database:

```bash
go test -run '^$' -bench . -benchmem ./control
```

Preparation is outside the measured time. In particular, state-changing methods receive fresh accounts, roles, sessions, invitations, or 2FA state per iteration. Read methods use a prepared, valid state; access checks run with the service cache enabled.
