// Package testkit provides testing helpers for portsmith Clean Architecture applications.
//
// # Testing the service layer (unit, no database)
//
// Use generated mocks via portsmith mock, then test with plain testing.T:
//
//	func TestCreateUser(t *testing.T) {
//	    repo := mocks.NewUserRepository(t)
//	    repo.On("FindByEmail", mock.Anything, "a@b.com").Return(nil, users.ErrUserNotFound)
//	    repo.On("Create", mock.Anything, mock.AnythingOfType("*users.User")).Return(nil)
//
//	    svc := users.NewService(repo)
//	    result, err := svc.Create(ctx, users.CreateParams{Email: "a@b.com"})
//
//	    testkit.NoError(t, err)
//	    testkit.Equal(t, "a@b.com", result.Email)
//	}
//
// # Testing the handler layer (HTTP, no database)
//
//	func TestUserHandler_Create(t *testing.T) {
//	    svc := mocks.NewUserService(t)
//	    svc.On("Create", ...).Return(&users.User{ID: 1}, nil)
//
//	    suite := testkit.NewHTTPSuite(t, userHandler.New(svc).Router())
//	    suite.POST("/users", `{"email":"a@b.com","name":"Alice"}`).
//	        ExpectStatus(201).
//	        ExpectJSONPath("$.id", float64(1))
//	}
//
// # Testing the repository layer (integration, SQLite in-memory)
//
//	func TestUserRepository_FindByEmail(t *testing.T) {
//	    db := testkit.NewTestDB(t, &users.User{})
//	    repo := users.NewRepository(db.DB())
//	    // ... test repository methods
//	}
//
// # Table-driven tests
//
//	testkit.Table(t, []testkit.Case{
//	    {Name: "success", Run: func(t *testing.T) { ... }},
//	    {Name: "not found", Run: func(t *testing.T) { ... }},
//	})
package testkit
