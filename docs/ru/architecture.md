# Руководство по архитектуре

Этот документ описывает архитектуру, которую portsmith ожидает в сгенерированных и проверяемых Go-пакетах.

portsmith — CLI-only toolkit. Он не предоставляет runtime-пакеты; сгенерированный код использует Gin, GORM, Chi, sqlx и стандартную библиотеку напрямую.

## Правило Зависимостей

Зависимости направлены внутрь:

```text
Handler -> ServicePort <- Service -> RepositoryPort <- Repository
```

Стрелки показывают направление импортов. Если `service.go` импортирует HTTP-пакеты, а `handler.go` импортирует database-пакеты, структура нарушена.

## Технологические Стеки

portsmith поддерживает два стека для scaffold/check:

- `gin-gorm`: Gin handlers и GORM repositories.
- `chi-sqlx`: Chi handlers и sqlx repositories.

Stack можно задать в `portsmith.yaml`:

```yaml
stack: chi-sqlx
```

или через `--stack` в `portsmith new` / `portsmith check`.

## Слои

### Domain Layer

Файлы: `domain.go`, `errors.go`

- Domain structs, enums, constants и domain errors.
- Не импортирует HTTP routers или database clients.
- Stack-specific struct tags допустимы, когда нужны generated repository code.

### Interface Layer

Файл: `ports.go`, генерируется через `portsmith gen`.

- Определяет минимальные интерфейсы `XxxRepository` и `XxxService`.
- Содержит только методы, которые реально использует потребляющий слой.
- Сохраняет compile-time assertions, что concrete types реализуют generated ports.

### Service Layer

Файл: `service.go`

- Реализует business logic.
- Зависит от repository interfaces, а не concrete repositories.
- Не импортирует HTTP packages или SQL/database clients.

### Repository Layer

Файл: `repository.go`

- Реализует repository interfaces.
- Может использовать GORM, sqlx, `database/sql` или другие storage clients.
- Транслирует storage errors в domain-level errors.
- Не содержит HTTP logic.

### Handler Layer

Файлы: `handler.go`, `dto.go`, опционально mapper files.

- Реализует HTTP endpoints.
- Зависит от service interfaces, а не concrete services.
- Может использовать Gin, Chi, `net/http`, DTO и request/response helpers.
- Не импортирует database clients.

## Wiring

Зависимости собираются вручную в application code:

```go
repo := orders.NewRepository(db)
svc := orders.NewService(repo)
h := orders.NewHandler(svc)
```

Конструкторы на границах слоев должны принимать interfaces:

```go
func NewService(repo OrderRepository) *Service
func NewHandler(svc OrderService) *Handler
```

## Основные Правила

| Rule id | Смысл |
|---|---|
| `ports-required` | нужен `ports.go`, если есть handler, service и repository файлы |
| `ports-complete` | generated ports должны соответствовать методам, которые используют handlers/services |
| `handler-no-db` | handlers не импортируют database clients |
| `service-no-http` | services не импортируют HTTP packages или routers |
| `no-concrete-fields` | handlers/services хранят interfaces, а не concrete layer pointers |
| `layer-dependency` | handlers не зависят от repository types; services не зависят от handler types |
| `type-placement` | exported layer structs живут в файлах своего слоя |
| `file-size` | опциональный лимит строк в файле |
| `cross-imports` | опциональный allowlist для `internal/...` cross imports |
| `constructor-injection` | constructors принимают interfaces на границах слоев |
| `test-files` | service/handler test files должны существовать |
| `no-panic` | запрет `panic()` в service/repository files |
| `context-first` | exported service/repository methods принимают `context.Context` первым |
| `method-count` | опциональный лимит exported methods на service/handler |
| `wiring-isolation` | layer constructors вызываются только из wiring files |
| `logger-no-other` | опциональное правило canonical logger import |
| `logger-no-fmt-print` | опциональное правило structured logging |
| `logger-no-init` | опциональное правило logger initialization |
| `call-pattern` | опциональное правило для `receiver.field.method()` calls |

## Конфигурация

```yaml
stack: chi-sqlx
lint:
  max_lines:
    - pattern: "repository*.go"
      limit: 800
  max_methods:
    - pattern: "service.go"
      limit: 25
  wiring:
    allowed_files:
      - "wire.go"
      - "app.go"
  logger:
    allowed: "log/slog"
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
  rules:
    test-files: { severity: warning }
```

Severity values: `error`, `warning`, `off`.

Inline suppression:

```go
//nolint:portsmith:handler-no-db
```
