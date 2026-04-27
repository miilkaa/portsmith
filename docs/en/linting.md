# Linting (`portsmith check`)

The `portsmith check` command runs architecture rules on your `internal/` packages. Settings live in **`portsmith.yaml`** at the repository root under the `lint:` key.

## Rule: `call-pattern` (opt-in)

Enforces naming of **three-level method calls** `receiver.field.method()` in layer files:

- **Handler files:** names matching `handler*.go` except `handler_test.go`
- **`service.go` only** for the service layer

This is **not** a file glob: each pattern has **exactly three dot-separated segments**. The segment `*` matches any single Go identifier.

Examples:

| Pattern | Meaning |
|--------|---------|
| `h.svc.*` | Receiver must be `h`, field must be `svc`, any method name |
| `*.svc.*` | Any receiver, field `svc`, any method name |
| `*.service.*` | Any receiver, field `service`, any method name |

### Configuration

```yaml
lint:
  call_patterns:
    handler:
      allowed:
        - "*.svc.*"
      not_allowed:
        - "*.service.*"
    service:
      allowed:
        - "*.repo.*"
      not_allowed:
        - "*.repository.*"
```

- **`not_allowed`:** if a call matches any listed pattern, `portsmith check` reports a **`call-pattern`** violation.
- **`allowed`:** does not enforce a whitelist (avoids false positives on unrelated three-level calls such as `req.Header.Get`). It is used only as a **hint** in the violation message (`use "..." instead`).

The rule is **off** until at least one layer has a non-empty **`not_allowed`** list. The **`allowed`** list only affects violation hints and does not enable the rule by itself.

### Severity

Default severity is **error** when `call_patterns` is configured. Override in `lint.rules`:

```yaml
lint:
  rules:
    call-pattern:
      severity: warning  # or off
```

### Scope limits

- Only **direct** calls `recv.field.Method()` are checked. Local aliases such as `x := h.svc; x.Method()` are not analyzed.
- Invalid patterns (not exactly three segments) are ignored for matching.
