# Руководство по TDD

Это руководство показывает, как применять Red → Green → Refactor в проектах на portsmith.

## Трёхуровневая стратегия тестирования

| Слой | Тип теста | Инструменты | База данных |
|------|-----------|-------------|-------------|
| Service | Unit | моки от `mockery` + `testing.T` | Не нужна |
| Handler | HTTP | `testkit.NewHTTPSuite` | Не нужна |
| Repository | Интеграционный | `testkit.NewTestDB` (SQLite) | SQLite in-memory |

## Red → Green → Refactor

```
1. Red     — пишем контрактные тесты, которые определяют публичный API. Они должны падать.
2. Green   — пишем минимальную реализацию, чтобы тесты прошли.
3. Refactor — чистим код. Тесты должны оставаться зелёными.
4. Unit    — добавляем граничные случаи, пути ошибок, краевые условия.
```

## Тестирование сервисного слоя

```go
func TestOrderService_Create_success(t *testing.T) {
    // 1. Создаём мок-репозиторий (сгенерированный portsmith mock)
    repo := mocks.NewOrderRepository(t)
    repo.On("Create", mock.Anything, mock.AnythingOfType("*orders.Order")).Return(nil)

    // 2. Собираем сервис с моком
    svc := orders.NewService(repo)

    // 3. Вызываем и проверяем результат
    order, err := svc.Create(ctx, orders.CreateParams{Item: "book"})
    testkit.NoError(t, err)
    testkit.Equal(t, "book", order.Item)

    repo.AssertExpectations(t)
}
```

Ключевой момент: ни базы данных, ни HTTP. Тест быстрый и полностью изолированный.

## Тестирование слоя хендлера

```go
func TestOrderHandler_Create(t *testing.T) {
    svc := mocks.NewOrderService(t)
    svc.On("Create", mock.Anything, mock.MatchedBy(func(p orders.CreateParams) bool {
        return p.Item == "book"
    })).Return(&orders.Order{ID: 1, Item: "book"}, nil)

    h := orders.NewHandler(svc)
    router := gin.New()
    h.Routes(router.Group("/api/v1"))

    suite := testkit.NewHTTPSuite(t, router)
    suite.POST("/api/v1/orders", `{"item":"book"}`).
        ExpectStatus(201).
        ExpectJSONPath("$.id", float64(1)).
        ExpectJSONPath("$.item", "book")
}
```

Ключевой момент: нет базы данных. Тест проверяет HTTP-парсинг и сериализацию ответа.

## Тестирование слоя репозитория

```go
func TestOrderRepository_FindByID_notFound(t *testing.T) {
    db := testkit.NewTestDB(t, &orders.Order{})
    repo := orders.NewRepository(db.DB())

    _, err := repo.FindByID(ctx, 9999)
    if !apperrors.IsCode(err, apperrors.CodeNotFound) {
        t.Errorf("expected NOT_FOUND, got %v", err)
    }
}
```

Ключевой момент: SQLite in-memory — без Docker, быстрый старт. Тест проверяет реальное SQL-поведение.

## Табличные тесты

```go
func TestOrderService_Create(t *testing.T) {
    testkit.Table(t, []testkit.Case{
        {
            Name: "успех",
            Run: func(t *testing.T) {
                // ...
            },
        },
        {
            Name: "пустой item возвращает ошибку",
            Run: func(t *testing.T) {
                // ...
            },
        },
    })
}
```

## Запуск тестов

```bash
# Все тесты
go test ./...

# С покрытием
go test ./... -cover

# Конкретный пакет
go test ./internal/orders/...

# Подробный вывод
go test ./... -v
```
