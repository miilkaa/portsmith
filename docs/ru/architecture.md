# Руководство по архитектуре

Этот документ объясняет подход Clean Architecture, применяемый в проектах на основе portsmith.

## Правило зависимостей

**Зависимости всегда направлены внутрь.** Внешние слои знают про внутренние; внутренние слои не знают ничего о внешних.

```
Handler → ServicePort ← Service → RepositoryPort ← Repository
   ↑                                                     ↑
HTTP-слой                                           Слой данных
(знает gin)                                         (знает gorm)

              Service + ports = Доменный слой
              (не знает ни про HTTP, ни про SQL)
```

Стрелки показывают **направление импортов**. Если вы видите импорт `net/http` внутри `service.go` или `gorm` внутри `handler.go` — это нарушение архитектуры.

## Слои и их правила

### Доменный слой (ядро)

Файлы: `domain.go`, `errors.go`

- Чистые Go-типы — структуры, перечисления, константы
- Запрещены импорты `database/sql`, `net/http`, `gorm`, `gin`
- GORM struct-теги допустимы (прагматичный компромисс ради AutoMigrate)
- Доменные ошибки используют конструкторы `apperrors`, но не содержат HTTP-кодов

### Слой интерфейсов

Файл: `ports.go` — **генерируется командой `portsmith gen`**

- Определяет интерфейсы `XxxRepository` и `XxxService`
- Содержит только те методы, которые реально вызываются потребляющим слоем (минимальные интерфейсы)
- Compile-time assertions: `var _ XxxRepository = (*Repository)(nil)`

### Сервисный слой

Файл: `service.go`

- Реализует бизнес-логику
- Зависит от интерфейса `XxxRepository`, никогда от конкретного `*Repository`
- Нет SQL, нет HTTP
- Возвращает доменные ошибки, никогда `gorm.ErrRecordNotFound`

### Слой репозитория

Файл: `repository.go`

- Реализует интерфейс `XxxRepository`
- Единственный слой, которому разрешено использовать `gorm` или `database/sql`
- Транслирует ошибки хранилища в доменные ошибки (напр. `gorm.ErrRecordNotFound` → `ErrUserNotFound`)
- Нет бизнес-логики — только доступ к данным

### Слой хендлера

Файлы: `handler.go`, `dto.go`, `mappers.go`

- Реализует HTTP-эндпоинты
- Зависит от интерфейса `XxxService`, никогда от конкретного `*Service`
- Единственный слой, которому разрешено использовать `gin` или `net/http`
- Цикл: HTTP-запрос → доменные параметры → вызов сервиса → доменный результат → HTTP-ответ
- Нет бизнес-логики, нет SQL

## Сборка зависимостей (Dependency Injection)

Всё собирается вручную в `main.go`:

```go
db, _ := database.Connect(cfg)
database.Register(db, &user.User{})

repo := user.NewRepository(db.DB())  // *Repository удовлетворяет UserRepository
svc  := user.NewService(repo)        // *Service  удовлетворяет UserService
h    := user.NewHandler(svc)

srv := server.New(server.Config{Port: 8080})
h.Routes(srv.Router().Group("/api/v1"))
srv.Run()
```

Конструкторы всегда принимают интерфейсы, а не конкретные типы:

```go
func NewService(repo UserRepository) *Service  // ✓
func NewHandler(svc UserService) *Handler      // ✓

func NewService(repo *Repository) *Service     // ✗ — нарушает тестируемость
```

## Проверка архитектуры

Запустите `portsmith check` для верификации правил (печатается выбранный stack; переопределение: `--stack`):

```bash
portsmith check ./internal/...
```

**Серьёзность:** по умолчанию все правила — `error` (код выхода 1). Переопределение в `portsmith.yaml`: `lint.rules.<id>.severity: error | warning | off`.

**Подавление в коде** (следующая строка или конец строки с кодом):

```go
//nolint:portsmith:handler-no-db
import "gorm.io/gorm"
```

### Основные правила

| Rule id | Смысл |
|---|---|
| `ports-required` | Нужен `ports.go`, если есть `handler.go`, `service.go`, `repository.go` |
| `handler-no-db` | В handler-файлах нельзя импортировать драйверы БД (`database/sql`, `gorm`, `sqlx`) |
| `service-no-http` | В `service.go` нельзя импортировать HTTP и роутеры |
| `no-concrete-fields` | Поля `Handler` / `Service` — не конкретные `*Service`, `*Repository`, `*Handler` |
| `layer-dependency` | `Handler` не зависит от типов слоя репозитория; `Service` — от типов handler-слоя (карта типов пакета + суффиксы) |
| `type-placement` | Экспортируемые struct в `repository*.go` / `service.go` — только «своего» слоя по имени |
| `file-size` | Лимит строк (опционально `lint.max_lines`) |
| `cross-imports` | Импорты `internal/...` только из allowlist (`lint.allowed_cross_imports`) |
| `constructor-injection` | Конструкторы принимают порты-интерфейсы, не `*` на слой |
| `test-files` | Наличие `service_test.go` / `handler_test.go` |
| `no-panic` | Запрет `panic()` в `service*.go` / `repository*.go` |
| `context-first` | У экспортируемых методов `*Service` / `*Repository` первый параметр — `context.Context` |
| `method-count` | Лимит экспортируемых методов (`lint.max_methods`) |
| `wiring-isolation` | Вызовы `New*Repository` / `New*Service` / `New*Handler` только в wiring-файлах (`lint.wiring.allowed_files`) |

### Правила логирования (opt-in)

**По умолчанию выключены.** Укажите `lint.logger.allowed` — канонический import-путь логгера (например `log/slog`). Включаются три правила с `error`, пока не переопределите в `lint.rules`. Действуют на **все** не-тестовые `.go` файлы пакета.

| Rule id | Смысл |
|---|---|
| `logger-no-other` | Запрет известных пакетов логирования (`log`, `log/slog`, `zap`, `logrus`, `zerolog`), кроме указанного в `allowed` |
| `logger-no-fmt-print` | Запрет `fmt.Print*` / `fmt.Fprint*` вместо логгера |
| `logger-no-init` | Запрет `<pkg>.New(...)` для разрешённого пакета (например `slog.New`); `slog.Default().With(...)` разрешён |

Реализация: пакет [`internal/lint`](../../internal/lint), CLI — [`cmd/portsmith/check`](../../cmd/portsmith/check/check.go).

### Секция `lint` в `portsmith.yaml` (опционально)

```yaml
stack: chi-sqlx
lint:
  max_lines:
    - pattern: "repository*.go"
      limit: 800
    - file: "internal/orders/repository.go"
      limit: 1200
  max_methods:
    - pattern: "service.go"
      limit: 25
  allowed_cross_imports:
    "repository*.go":
      - "internal/shared"
  wiring:
    allowed_files:
      - "module.go"
      - "wire.go"
  logger:
    allowed: "log/slog"
  rules:
    test-files:     { severity: warning }
    context-first:  { severity: warning }
    method-count:   { severity: warning }
```
