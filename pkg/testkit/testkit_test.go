package testkit_test

// testkit_test.go — контрактные тесты для pkg/testkit.
//
// Контракт:
//  1. NewHTTPSuite создаёт suite вокруг gin.Engine.
//  2. suite.GET/POST/PATCH/DELETE выполняют запросы и возвращают Response.
//  3. Response.ExpectStatus проверяет статус и сообщает t.Fatalf при несовпадении.
//  4. Response.ExpectJSONPath проверяет значение по JSONPath-выражению.
//  5. NewTestDB возвращает *database.DB с SQLite in-memory и заданными моделями.
//  6. Table запускает table-driven тесты через функциональный API.
//  7. NoError/Equal/True — обёртки для читаемых ассершенов.

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/miilkaa/portsmith/pkg/testkit"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- HTTPSuite tests ---

func TestHTTPSuite_GET_expectStatus(t *testing.T) {
	router := gin.New()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	suite := testkit.NewHTTPSuite(t, router)
	suite.GET("/ping").ExpectStatus(http.StatusOK)
}

func TestHTTPSuite_POST_expectJSONPath(t *testing.T) {
	router := gin.New()
	router.POST("/echo", func(c *gin.Context) {
		var body map[string]any
		_ = c.ShouldBindJSON(&body)
		c.JSON(http.StatusCreated, body)
	})

	suite := testkit.NewHTTPSuite(t, router)
	suite.POST("/echo", `{"name":"Alice"}`).
		ExpectStatus(http.StatusCreated).
		ExpectJSONPath("$.name", "Alice")
}

func TestHTTPSuite_withHeader(t *testing.T) {
	router := gin.New()
	router.GET("/auth", func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.Status(http.StatusUnauthorized)
			return
		}
		c.Status(http.StatusOK)
	})

	suite := testkit.NewHTTPSuite(t, router)
	suite.GET("/auth").
		WithHeader("Authorization", "Bearer token123").
		ExpectStatus(http.StatusOK)
}

// --- NewTestDB test ---

type dbModel struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"not null"`
}

func TestNewTestDB_createsInMemoryDB(t *testing.T) {
	db := testkit.NewTestDB(t, &dbModel{})
	if db == nil {
		t.Fatal("expected non-nil DB")
	}

	// Write and read a record to verify the DB is functional.
	result := db.DB().Create(&dbModel{Name: "test"})
	if result.Error != nil {
		t.Fatalf("create failed: %v", result.Error)
	}

	var found dbModel
	if err := db.DB().First(&found, "name = ?", "test").Error; err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if found.Name != "test" {
		t.Errorf("expected name 'test', got %s", found.Name)
	}
}

// --- Table test ---

func TestTable_runsAllCases(t *testing.T) {
	ran := make(map[string]bool)

	testkit.Table(t, []testkit.Case{
		{Name: "case-a", Run: func(t *testing.T) { ran["case-a"] = true }},
		{Name: "case-b", Run: func(t *testing.T) { ran["case-b"] = true }},
	})

	for _, name := range []string{"case-a", "case-b"} {
		if !ran[name] {
			t.Errorf("expected %s to run", name)
		}
	}
}

// --- Assertion helpers ---

func TestNoError_passes(t *testing.T) {
	testkit.NoError(t, nil)
}

func TestEqual_passes(t *testing.T) {
	testkit.Equal(t, "hello", "hello")
}

func TestTrue_passes(t *testing.T) {
	testkit.True(t, true, "should be true")
}
