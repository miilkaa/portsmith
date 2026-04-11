package example_test

// service_test.go — unit-тесты бизнес-логики.
//
// Ключевая идея: тестируем Service в полной изоляции от БД.
// Вместо реального репозитория передаём mockRepository — простую
// ручную реализацию UserRepository для тестов.
//
// В реальном проекте вместо ручного мока используется сгенерированный:
//
//	portsmith mock ./internal/example
//	# создаёт: internal/example/mocks/mock_user_repository.go
//
// Здесь пишем мок вручную, чтобы пример был самодостаточным
// и не зависел от внешних инструментов.

import (
	"context"
	"errors"
	"testing"

	example "github.com/miilkaa/portsmith/examples/clean_package_example_ru"
	"github.com/miilkaa/portsmith/pkg/pagination"
)

// mockRepository — тестовая реализация UserRepository.
// Хранит данные в памяти. В реальном проекте используй mockery.
type mockRepository struct {
	users  map[uint]*example.User
	nextID uint

	// Перехватчики для симуляции ошибок в конкретных тестах.
	createErr   error
	findByIDErr error
	findByEmail func(email string) (*example.User, error)
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:  make(map[uint]*example.User),
		nextID: 1,
	}
}

func (m *mockRepository) Create(ctx context.Context, user *example.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	user.ID = m.nextID
	m.nextID++
	m.users[user.ID] = user
	return nil
}

func (m *mockRepository) FindByID(ctx context.Context, id uint) (*example.User, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	u, ok := m.users[id]
	if !ok {
		return nil, example.ErrUserNotFound
	}
	return u, nil
}

func (m *mockRepository) FindByEmail(ctx context.Context, email string) (*example.User, error) {
	if m.findByEmail != nil {
		return m.findByEmail(email)
	}
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, example.ErrUserNotFound
}

func (m *mockRepository) Update(ctx context.Context, user *example.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id uint) error {
	delete(m.users, id)
	return nil
}

func (m *mockRepository) List(ctx context.Context, filter example.ListFilter, page pagination.OffsetPage) ([]*example.User, int64, error) {
	var result []*example.User
	for _, u := range m.users {
		result = append(result, u)
	}
	return result, int64(len(result)), nil
}

// --- Тесты ---

func TestService_Create_success(t *testing.T) {
	repo := newMockRepository()
	svc := example.NewService(repo)

	user, err := svc.Create(context.Background(), example.CreateParams{
		Email: "alice@example.com",
		Name:  "Alice",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", user.Email)
	}
	// Роль по умолчанию должна быть RoleUser.
	if user.Role != example.RoleUser {
		t.Errorf("expected role %s, got %s", example.RoleUser, user.Role)
	}
}

func TestService_Create_duplicateEmail(t *testing.T) {
	repo := newMockRepository()
	svc := example.NewService(repo)

	params := example.CreateParams{Email: "alice@example.com", Name: "Alice"}
	if _, err := svc.Create(context.Background(), params); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err := svc.Create(context.Background(), params)
	if !errors.Is(err, example.ErrEmailTaken) {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}
}

func TestService_GetByID_notFound(t *testing.T) {
	repo := newMockRepository()
	svc := example.NewService(repo)

	_, err := svc.GetByID(context.Background(), 999)
	if !errors.Is(err, example.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestService_Update_cannotDeactivateSelf(t *testing.T) {
	repo := newMockRepository()
	svc := example.NewService(repo)

	user, err := svc.Create(context.Background(), example.CreateParams{
		Email: "bob@example.com",
		Name:  "Bob",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	active := false
	_, err = svc.Update(context.Background(), user.ID, example.UpdateParams{
		Active: &active,
	}, user.ID) // callerID == user.ID — деактивируем себя

	if !errors.Is(err, example.ErrCannotDeactivateSelf) {
		t.Errorf("expected ErrCannotDeactivateSelf, got %v", err)
	}
}

func TestService_Update_success(t *testing.T) {
	repo := newMockRepository()
	svc := example.NewService(repo)

	user, _ := svc.Create(context.Background(), example.CreateParams{
		Email: "carol@example.com",
		Name:  "Carol",
	})

	newName := "Carol Updated"
	updated, err := svc.Update(context.Background(), user.ID, example.UpdateParams{
		Name: &newName,
	}, 999) // другой callerID

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected name %q, got %q", newName, updated.Name)
	}
}

func TestService_Delete_notFound(t *testing.T) {
	repo := newMockRepository()
	svc := example.NewService(repo)

	err := svc.Delete(context.Background(), 999)
	if !errors.Is(err, example.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}
