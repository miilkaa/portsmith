# portsmith

**portsmith** — Go-фреймворк для построения бэкендов с Clean Architecture «из коробки».

Он даёт вашей команде:
- Единую структуру, понятную каждому разработчику и QA-инженеру с первого взгляда
- CLI-инструменты для генерации, скаффолдинга, моков и валидации архитектуры
- Runtime-библиотеки для HTTP, базы данных, конфигурации, пагинации и тестирования
- Архитектурный линтер для CI/CD (`portsmith check`)

---

## Содержание

- [Быстрый старт](#быстрый-старт)
- [Обзор Clean Architecture](#обзор-clean-architecture)
- [Структура проекта](#структура-проекта)
- [Библиотеки](#библиотеки)
- [CLI-инструменты](#cli-инструменты)
- [Линтинг](docs/ru/linting.md)
- [Тестирование](#тестирование)
- [Интеграция с CI/CD](#интеграция-с-cicd)
- [Пример пакета](#пример-пакета)

---

## Быстрый старт

```bash
# Установить CLI
go install github.com/miilkaa/portsmith/cmd/portsmith@latest

# Создать новый пакет
portsmith new internal/orders

# Реализовать домен, сервис и репозиторий, затем сгенерировать порты:
portsmith gen internal/orders

# Сгенерировать моки (требуется mockery)
go install github.com/vektra/mockery/v2@latest
portsmith mock internal/orders

# Проверить архитектуру
portsmith check ./internal/...
```

В `main.go`:

```go
package main

import (
    "log"
    "os"

    "github.com/miilkaa/portsmith/pkg/config"
    "github.com/miilkaa/portsmith/pkg/database"
    "github.com/miilkaa/portsmith/pkg/server"

    "yourmodule/internal/orders"
)

type Config struct {
    Port        int    `env:"PORT"         env-default:"8080"`
    DatabaseURL string `env:"DATABASE_URL" env-required:"true"`
}

func main() {
    var cfg Config
    if err := config.Load(&cfg); err != nil {
        log.Fatal(err)
    }

    db, err := database.Connect(database.Config{DSN: cfg.DatabaseURL})
    if err != nil {
        log.Fatal(err)
    }
    database.Register(db, &orders.Order{})

    repo := orders.NewRepository(db.DB())
    svc  := orders.NewService(repo)
    h    := orders.NewHandler(svc)

    srv := server.New(server.Config{Port: cfg.Port})
    h.Routes(srv.Router().Group("/api/v1"))
    log.Fatal(srv.Run())
}
```

---

## Обзор Clean Architecture

Зависимости направлены **только внутрь**. Внешние слои знают про внутренние, но не наоборот.

```
Handler → ServicePort ← Service → RepositoryPort ← Repository
```

| Файл | Слой | Знает про | Не знает про |
|------|------|-----------|--------------|
| `domain.go` | Ядро | Go-типы | database, HTTP |
| `errors.go` | Ядро | `apperrors` | HTTP-коды |
| `ports.go` | Интерфейсы | domain | SQL, HTTP |
| `service.go` | Бизнес-логика | ports, domain | SQL, HTTP |
| `repository.go` | Данные | gorm, domain | HTTP |
| `handler.go` | HTTP | ports, dto | SQL |
| `dto.go` | HTTP | Go-типы, validator | SQL |
| `mappers.go` | Трансформация | domain, dto | всё остальное |

Подробное объяснение — в [docs/ru/architecture.md](docs/ru/architecture.md).

---

## Структура проекта

```
your-app/
├── internal/
│   ├── orders/
│   │   ├── domain.go
│   │   ├── errors.go
│   │   ├── ports.go          ← генерируется portsmith gen
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── handler.go
│   │   ├── dto.go
│   │   └── mocks/            ← генерируется portsmith mock
│   └── users/
│       └── ...
└── main.go
```

---

## Библиотеки

### `pkg/apperrors`

Типизированные доменные ошибки с автоматическим маппингом на HTTP-статусы.

```go
var ErrNotFound = apperrors.NotFound("order not found")   // → 404
var ErrConflict = apperrors.Conflict("duplicate order")   // → 409

// В сервисе — возвращаем доменные ошибки, не HTTP-коды:
if errors.Is(err, ErrNotFound) { return nil, ErrNotFound }

// Middleware сервера обрабатывает маппинг на HTTP автоматически.
```

[Полная документация](pkg/apperrors/doc.go)

---

### `pkg/database`

Обёртка над GORM с авто-миграциями и генерик-репозиторием `Repository[T]`.

```go
db, _ := database.Connect(database.Config{DSN: os.Getenv("DATABASE_URL")})
database.Register(db, &orders.Order{}, &users.User{})

// Генерик-репозиторий для базовых CRUD-операций:
type Repository struct {
    base database.Repository[Order]
    db   *gorm.DB
}

// Транзакции:
database.WithTx(ctx, db, func(tx *database.DB) error {
    return repo.Create(ctx, order)
})
```

[Полная документация](pkg/database/doc.go)

---

### `pkg/pagination`

Пагинация на основе offset с парсингом HTTP query-параметров.

```go
// В хендлере:
page := pagination.OffsetFromQuery(c.Request)  // ?page=2&limit=20

// В репозитории:
query.Offset(page.Offset()).Limit(page.Limit())

// В ответе:
totalPages := pagination.TotalPages(total, page.Limit())
```

[Полная документация](pkg/pagination/doc.go)

---

### `pkg/config`

Загрузка конфигурации из переменных окружения.

```go
type Config struct {
    Port int    `env:"PORT" env-default:"8080"`
    DSN  string `env:"DATABASE_URL" env-required:"true"`
}
var cfg Config
config.Load(&cfg)
```

[Полная документация](pkg/config/doc.go)

---

### `pkg/server`

Gin-сервер с батарейками в комплекте.

```go
srv := server.New(server.Config{Port: 8080})
// Встроено: GET /health, recovery, requestID, CORS, error → HTTP маппинг
userHandler.Routes(srv.Router().Group("/api/v1"))
srv.Run()
```

[Полная документация](pkg/server/doc.go)

---

### `pkg/testkit`

Тестовые хелперы для всех слоёв архитектуры.

```go
// Тест хендлера (без базы данных):
suite := testkit.NewHTTPSuite(t, router)
suite.POST("/orders", `{"item":"book"}`).ExpectStatus(201).ExpectJSONPath("$.id", float64(1))

// Тест репозитория (SQLite in-memory):
db := testkit.NewTestDB(t, &orders.Order{})

// Табличные тесты:
testkit.Table(t, []testkit.Case{
    {Name: "success", Run: func(t *testing.T) { ... }},
})
```

[Полная документация](pkg/testkit/doc.go)

---

## CLI-инструменты

### `portsmith gen`

Генерирует `ports.go` — минимальные интерфейсы из реального использования.

```bash
portsmith gen internal/orders          # один пакет
portsmith gen --all                    # все пакеты в internal/
portsmith gen --dry-run internal/orders # предпросмотр без записи
```

### `portsmith new`

Создаёт новый пакет со всеми файлами Clean Architecture.

```bash
portsmith new internal/products
# создаёт: domain.go, errors.go, ports.go, service.go, repository.go, handler.go, dto.go
```

### `portsmith mock`

Генерирует моки для всех интерфейсов в `ports.go` (обёртка над [mockery](https://github.com/vektra/mockery)).

```bash
portsmith mock internal/orders
# создаёт: internal/orders/mocks/mock_orders_repository.go
#          internal/orders/mocks/mock_orders_service.go
```

### `portsmith check`

Проверяет правила архитектуры. Код выхода `1` только при нарушениях уровня **error**; **warning** печатаются, но не роняют команду.

```bash
portsmith check ./internal/...
```

Базовые правила (всегда): наличие `ports.go` при тройке handler/service/repository; запрет драйверов БД в handler-файлах; запрет HTTP/роутеров в `service.go`; поля через порты, не `*Service`/`*Repository`; направление зависимостей между слоями; экспортируемые типы только в «своих» файлах слоя; конструкторы на интерфейсах; запрет `panic` в service/repository; опционально `context.Context` первым параметром у экспортируемых методов сервиса/репозитория.

Дополнительно через `portsmith.yaml` → `lint`: лимиты строк и методов, allowlist для `internal/...`, изоляция wiring для `New*Repository` / `New*Service` / `New*Handler`.

Настройка серьёзности и подавление в коде — см. [docs/ru/architecture.md](docs/ru/architecture.md#проверка-архитектуры).

---

## Тестирование

portsmith использует **Red → Green → Refactor** (TDD):

1. Сначала пишем контрактные тесты (определяем публичный API)
2. Реализуем до прохождения тестов
3. Рефакторим — тесты страхуют от регрессий

Три уровня тестирования:

| Слой | Что использовать | База данных |
|------|-----------------|-------------|
| Service | моки от `mockery` | Не нужна |
| Handler | `testkit.NewHTTPSuite` | Не нужна |
| Repository | `testkit.NewTestDB` (SQLite) | SQLite in-memory |

Паттерны и примеры — в [docs/ru/tdd-guide.md](docs/ru/tdd-guide.md).

---

## Интеграция с CI/CD

```yaml
# GitHub Actions
- name: Architecture check
  run: go run ./cmd/portsmith check ./internal/...

# GitLab CI
architecture:
  script:
    - go run ./cmd/portsmith check ./internal/...
```

---

## Пример пакета

Полная эталонная реализация доступна в двух локалях:

- [`examples/clean_package_example_en`](examples/clean_package_example_en) — с комментариями на **английском**
- [`examples/clean_package_example_ru`](examples/clean_package_example_ru) — с комментариями на **русском**

Оба примера демонстрируют все слои с подробными комментариями к каждому архитектурному решению.

---

## Требования

- Go 1.21+
- PostgreSQL (или SQLite для разработки/тестов)
- [mockery](https://github.com/vektra/mockery) для генерации моков

---

## Лицензия

[MIT](LICENSE)
