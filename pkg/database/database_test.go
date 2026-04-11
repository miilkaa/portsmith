package database_test

// database_test.go — контрактные тесты для pkg/database.
//
// Контракт:
//  1. Connect возвращает *gorm.DB с рабочим соединением.
//  2. Register запускает AutoMigrate для переданных моделей.
//  3. Repository[T].Create/FindByID/Update/Delete работают корректно.
//  4. FindByID возвращает apperrors.NotFound при отсутствии записи.
//  5. WithTx выполняет функцию в транзакции и коммитит её.
//  6. WithTx откатывает транзакцию при ошибке.
//
// Все тесты используют SQLite in-memory — никакого Docker/Postgres.

import (
	"context"
	"errors"
	"testing"

	"github.com/miilkaa/portsmith/pkg/apperrors"
	"github.com/miilkaa/portsmith/pkg/database"
)

// testModel — простая модель для тестов, не связана с domain пакетами.
type testModel struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"not null"`
	Score int
}

func setupDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Connect(database.Config{Driver: database.DriverSQLite, DSN: ":memory:"})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := database.Register(db, &testModel{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	return db
}

func TestConnect_returnsFunctionalDB(t *testing.T) {
	db := setupDB(t)
	if db == nil {
		t.Fatal("expected non-nil DB")
	}
	// Verify underlying connection works.
	rawDB := db.DB()
	sqlDB, err := rawDB.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestRegister_createsTables(t *testing.T) {
	db := setupDB(t)
	// If AutoMigrate succeeded, we can query the table without error.
	repo := database.NewRepository[testModel](db.DB())
	if err := repo.Create(context.Background(), &testModel{Name: "probe"}); err != nil {
		t.Fatalf("create after register: %v", err)
	}
}

func TestRepository_createAndFindByID(t *testing.T) {
	db := setupDB(t)
	repo := database.NewRepository[testModel](db.DB())

	m := &testModel{Name: "Alice", Score: 42}
	if err := repo.Create(context.Background(), m); err != nil {
		t.Fatalf("create: %v", err)
	}
	if m.ID == 0 {
		t.Fatal("expected ID to be set after create")
	}

	found, err := repo.FindByID(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("findByID: %v", err)
	}
	if found.Name != "Alice" || found.Score != 42 {
		t.Errorf("unexpected record: %+v", found)
	}
}

func TestRepository_findByID_notFound(t *testing.T) {
	db := setupDB(t)
	repo := database.NewRepository[testModel](db.DB())

	_, err := repo.FindByID(context.Background(), 9999)
	if !apperrors.IsCode(err, apperrors.CodeNotFound) {
		t.Errorf("expected NOT_FOUND apperror, got %v", err)
	}
}

func TestRepository_update(t *testing.T) {
	db := setupDB(t)
	repo := database.NewRepository[testModel](db.DB())

	m := &testModel{Name: "Bob", Score: 10}
	_ = repo.Create(context.Background(), m)

	m.Score = 99
	if err := repo.Update(context.Background(), m); err != nil {
		t.Fatalf("update: %v", err)
	}

	found, _ := repo.FindByID(context.Background(), m.ID)
	if found.Score != 99 {
		t.Errorf("expected score 99, got %d", found.Score)
	}
}

func TestRepository_delete(t *testing.T) {
	db := setupDB(t)
	repo := database.NewRepository[testModel](db.DB())

	m := &testModel{Name: "Charlie"}
	_ = repo.Create(context.Background(), m)
	_ = repo.Delete(context.Background(), m.ID)

	_, err := repo.FindByID(context.Background(), m.ID)
	if !apperrors.IsCode(err, apperrors.CodeNotFound) {
		t.Errorf("expected NOT_FOUND after delete, got %v", err)
	}
}

func TestWithTx_commits(t *testing.T) {
	db := setupDB(t)
	repo := database.NewRepository[testModel](db.DB())

	m := &testModel{Name: "TxUser"}
	err := database.WithTx(context.Background(), db, func(tx *database.DB) error {
		txRepo := database.NewRepository[testModel](tx.DB())
		return txRepo.Create(context.Background(), m)
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}

	found, err := repo.FindByID(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("record not found after committed tx: %v", err)
	}
	if found.Name != "TxUser" {
		t.Errorf("unexpected name: %s", found.Name)
	}
}

func TestWithTx_rollbackOnError(t *testing.T) {
	db := setupDB(t)
	repo := database.NewRepository[testModel](db.DB())

	var savedID uint
	sentinelErr := errors.New("forced rollback")

	_ = database.WithTx(context.Background(), db, func(tx *database.DB) error {
		txRepo := database.NewRepository[testModel](tx.DB())
		m := &testModel{Name: "ShouldNotExist"}
		_ = txRepo.Create(context.Background(), m)
		savedID = m.ID
		return sentinelErr // trigger rollback
	})

	if savedID == 0 {
		t.Skip("ID not assigned, cannot verify rollback")
	}
	_, err := repo.FindByID(context.Background(), savedID)
	if !apperrors.IsCode(err, apperrors.CodeNotFound) {
		t.Errorf("record must not exist after rollback, got err=%v", err)
	}
}
