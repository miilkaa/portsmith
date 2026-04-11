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

Запустите `portsmith check` для верификации всех правил:

```bash
portsmith check ./internal/...
```

Полный список правил — в [portsmith check](../cmd/portsmith/check/check.go).
