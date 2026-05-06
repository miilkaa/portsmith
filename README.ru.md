# portsmith

**portsmith** — CLI toolkit для создания и поддержки Go-пакетов в стиле Clean Architecture.

Он предоставляет:

- настройку проекта через `portsmith init`
- scaffolding пакетов через `portsmith new`
- генерацию минимальных портов через `portsmith gen`
- генерацию моков через `portsmith mock`
- проверку архитектуры через `portsmith check`

portsmith больше не поставляет runtime-библиотеки. Сгенерированный код использует обычные библиотеки Go-экосистемы напрямую: Gin, GORM, Chi, sqlx и стандартную библиотеку.

## Быстрый Старт

```bash
# Установить CLI
go install github.com/miilkaa/portsmith/cmd/portsmith@latest

# Создать portsmith.yaml в существующем проекте
portsmith init

# Создать новый пакет
portsmith new --stack gin-gorm internal/orders

# Реализовать domain, service и repository, затем сгенерировать ports.go
portsmith gen internal/orders

# Сгенерировать моки для интерфейсов из ports.go
go install github.com/vektra/mockery/v2@latest
portsmith mock internal/orders

# Проверить архитектуру
portsmith check ./internal/...
```

## Clean Architecture

Зависимости направлены только внутрь:

```text
Handler -> ServicePort <- Service -> RepositoryPort <- Repository
```

| Файл | Слой | Может знать про | Не должен знать про |
|---|---|---|---|
| `domain.go` | Core | Go-типы | HTTP, SQL clients |
| `errors.go` | Core | стандартные errors | HTTP status mapping |
| `ports.go` | Interfaces | domain-типы | конкретный HTTP/SQL implementation |
| `service.go` | Business logic | repository port, domain | HTTP, SQL |
| `repository.go` | Data access | GORM или sqlx | HTTP |
| `handler.go` | HTTP | service port, DTO | SQL |
| `dto.go` | HTTP DTO | request/response-типы | SQL |

## Структура, Которую Генерирует `portsmith new`

```text
your-app/
├── internal/
│   └── orders/
│       ├── domain.go
│       ├── errors.go
│       ├── ports.go
│       ├── service.go
│       ├── repository.go
│       ├── handler.go
│       └── dto.go
└── main.go
```

## Технологические Стеки

`portsmith new` и `portsmith check` поддерживают два стека:

| Stack | HTTP | Database | Generated Code |
|---|---|---|---|
| `gin-gorm` | Gin | GORM | `uint` IDs, прямое использование Gin/GORM |
| `chi-sqlx` | Chi v5 | sqlx + PostgreSQL | `uuid.UUID` IDs, прямое использование Chi/sqlx |

Приоритет определения stack:

1. `--stack gin-gorm` или `--stack chi-sqlx`
2. `stack:` в `portsmith.yaml`
3. эвристика по `go.mod`: Chi module -> `chi-sqlx`, Gin module -> `gin-gorm`
4. default: `gin-gorm`

## CLI Команды

```bash
portsmith init  [--force]
portsmith new   [--stack gin-gorm|chi-sqlx] <pkg-dir>
portsmith gen   [--dry-run] [--all] [--scan-callers] [<pkg-dir>...]
portsmith mock  [<pkg-dir>...]
portsmith check [--stack gin-gorm|chi-sqlx] [<pkg-dir>...]
portsmith version
portsmith help
```

## Проверка Архитектуры

`portsmith check` проверяет правила Clean Architecture на уровне пакетов:

- наличие и полноту `ports.go`
- отсутствие database imports в handlers
- отсутствие HTTP imports в services
- constructor injection через интерфейсы
- `context.Context` первым параметром в service/repository методах
- лимиты размера файлов и количества методов
- опциональные правила logger и call-pattern

Конфигурация находится в `portsmith.yaml`:

```yaml
stack: chi-sqlx
lint:
  rules:
    test-files: { severity: warning }
  logger:
    allowed: "log/slog"
```

Полный справочник правил: [`docs/ru/architecture.md`](docs/ru/architecture.md).
