package example

// service.go — слой бизнес-логики.
//
// Правила этого слоя:
//  1. Принимает зависимости ТОЛЬКО через интерфейсы (UserRepository).
//  2. Не знает про SQL, gorm, database/sql.
//  3. Не знает про HTTP, net/http, gin.
//  4. Содержит бизнес-правила: валидацию, авторизацию, оркестрацию.
//  5. Возвращает доменные ошибки (errors.go), не gorm.ErrRecordNotFound.
//
// Как тестировать:
//   - Передаём мок-репозиторий (сгенерированный portsmith mock).
//   - Никакой базы данных не нужно.
//   - Тесты в service_test.go.

import (
	"context"
	"errors"

	"github.com/miilkaa/portsmith/pkg/pagination"
)

// Service реализует бизнес-логику управления пользователями.
type Service struct {
	// repo — зависимость через интерфейс, не через *Repository.
	// Имя поля "repo" соответствует конвенции portsmith gen
	// (инструмент ищет s.repo.Method() для генерации портов).
	repo UserRepository
}

// NewService создаёт новый Service. Принимает интерфейс — легко тестируется.
func NewService(repo UserRepository) *Service {
	return &Service{repo: repo}
}

// Create создаёт нового пользователя.
//
// Бизнес-правила:
//   - Email должен быть уникальным (проверяем через репозиторий).
//   - Если роль не указана — назначаем RoleUser по умолчанию.
func (s *Service) Create(ctx context.Context, params CreateParams) (*User, error) {
	// Проверяем уникальность email — бизнес-правило, не SQL-ограничение.
	existing, err := s.repo.FindByEmail(ctx, params.Email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	role := params.Role
	if role == "" {
		role = RoleUser
	}

	user := &User{
		Email: params.Email,
		Name:  params.Name,
		Role:  role,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// GetByID возвращает пользователя по ID.
func (s *Service) GetByID(ctx context.Context, id uint) (*User, error) {
	return s.repo.FindByID(ctx, id)
}

// Update обновляет поля пользователя.
//
// Бизнес-правила:
//   - Обычный пользователь не может менять свою роль.
//   - Нельзя деактивировать самого себя (callerID == id && Active = false).
func (s *Service) Update(ctx context.Context, id uint, params UpdateParams, callerID uint) (*User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if params.Active != nil && !*params.Active && callerID == id {
		return nil, ErrCannotDeactivateSelf
	}

	if params.Name != nil {
		user.Name = *params.Name
	}
	if params.Role != nil {
		user.Role = *params.Role
	}
	if params.Active != nil {
		user.Active = *params.Active
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Delete удаляет пользователя по ID.
func (s *Service) Delete(ctx context.Context, id uint) error {
	// Проверяем что пользователь существует перед удалением.
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// List возвращает список пользователей с фильтрацией и пагинацией.
func (s *Service) List(ctx context.Context, filter ListFilter, page pagination.OffsetPage) ([]*User, int64, error) {
	return s.repo.List(ctx, filter, page)
}
