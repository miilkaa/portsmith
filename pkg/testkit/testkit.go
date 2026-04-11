// Package testkit provides testing helpers for Clean Architecture Go applications.
//
// Three testing levels are supported:
//
// # Service tests (unit, no database)
//
// Use generated mocks (portsmith mock) with the service constructor.
// testkit provides assertion helpers on top of testing.T.
//
// # Handler tests (HTTP, no database)
//
// Use HTTPSuite to make HTTP requests against a gin.Engine:
//
//	suite := testkit.NewHTTPSuite(t, router)
//	suite.POST("/users", `{"email":"a@b.com"}`).
//	    ExpectStatus(201).
//	    ExpectJSONPath("$.id", float64(1))
//
// # Repository tests (integration, SQLite in-memory)
//
// Use NewTestDB to get a *database.DB backed by SQLite:
//
//	db := testkit.NewTestDB(t, &user.User{})
//	repo := user.NewRepository(db.DB())
//	// test repository methods...
package testkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/miilkaa/portsmith/pkg/database"
)

// --- HTTPSuite ---

// HTTPSuite wraps a gin.Engine for ergonomic HTTP testing.
type HTTPSuite struct {
	t      *testing.T
	router *gin.Engine
}

// NewHTTPSuite creates a new HTTPSuite for the given router.
func NewHTTPSuite(t *testing.T, router *gin.Engine) *HTTPSuite {
	t.Helper()
	return &HTTPSuite{t: t, router: router}
}

// requestBuilder builds an HTTP request before execution.
type requestBuilder struct {
	suite   *HTTPSuite
	method  string
	path    string
	body    string
	headers map[string]string
}

func (s *HTTPSuite) newRequest(method, path, body string) *requestBuilder {
	return &requestBuilder{
		suite:   s,
		method:  method,
		path:    path,
		body:    body,
		headers: make(map[string]string),
	}
}

// GET starts a GET request builder.
func (s *HTTPSuite) GET(path string) *requestBuilder {
	return s.newRequest(http.MethodGet, path, "")
}

// POST starts a POST request builder with a JSON body.
func (s *HTTPSuite) POST(path, body string) *requestBuilder {
	return s.newRequest(http.MethodPost, path, body)
}

// PATCH starts a PATCH request builder with a JSON body.
func (s *HTTPSuite) PATCH(path, body string) *requestBuilder {
	return s.newRequest(http.MethodPatch, path, body)
}

// DELETE starts a DELETE request builder.
func (s *HTTPSuite) DELETE(path string) *requestBuilder {
	return s.newRequest(http.MethodDelete, path, "")
}

// WithHeader adds a header to the request.
func (rb *requestBuilder) WithHeader(key, value string) *requestBuilder {
	rb.headers[key] = value
	return rb
}

// execute performs the HTTP request and returns a Response.
func (rb *requestBuilder) execute() *Response {
	rb.suite.t.Helper()

	var bodyReader *bytes.Reader
	if rb.body != "" {
		bodyReader = bytes.NewReader([]byte(rb.body))
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(rb.method, rb.path, bodyReader)
	if rb.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range rb.headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	rb.suite.router.ServeHTTP(w, req)

	return &Response{t: rb.suite.t, recorder: w}
}

// ExpectStatus asserts the HTTP status code.
func (rb *requestBuilder) ExpectStatus(code int) *Response {
	rb.suite.t.Helper()
	resp := rb.execute()
	resp.ExpectStatus(code)
	return resp
}

// Response holds the HTTP test response and provides assertion methods.
type Response struct {
	t        *testing.T
	recorder *httptest.ResponseRecorder
}

// ExpectStatus asserts the HTTP status code matches the expected value.
func (r *Response) ExpectStatus(code int) *Response {
	r.t.Helper()
	if r.recorder.Code != code {
		r.t.Fatalf("expected HTTP %d, got %d\nbody: %s", code, r.recorder.Code, r.recorder.Body.String())
	}
	return r
}

// ExpectJSONPath asserts that the JSON path in the response body equals the expected value.
// Uses simple dot-notation paths ($.field or $.nested.field or $.field[0]).
func (r *Response) ExpectJSONPath(path string, want any) *Response {
	r.t.Helper()

	var body any
	if err := json.Unmarshal(r.recorder.Body.Bytes(), &body); err != nil {
		r.t.Fatalf("response body is not valid JSON: %v\nbody: %s", err, r.recorder.Body.String())
	}

	got, err := jsonPath(body, path)
	if err != nil {
		r.t.Fatalf("jsonpath %q: %v\nbody: %s", path, err, r.recorder.Body.String())
	}

	// Normalize for comparison: JSON numbers are float64.
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		r.t.Errorf("jsonpath %q: expected %v (%T), got %v (%T)", path, want, want, got, got)
	}
	return r
}

// Body returns the raw response body bytes.
func (r *Response) Body() []byte {
	return r.recorder.Body.Bytes()
}

// Code returns the HTTP status code.
func (r *Response) Code() int {
	return r.recorder.Code
}

// jsonPath evaluates a simple JSONPath expression like $.field or $.a.b.
func jsonPath(data any, path string) (any, error) {
	// Strip leading $. or $
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")
	if path == "" {
		return data, nil
	}

	parts := strings.Split(path, ".")
	current := data
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected object at %q, got %T", part, current)
		}
		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("key %q not found", part)
		}
		current = val
	}
	return current, nil
}

// --- NewTestDB ---

// NewTestDB creates a SQLite in-memory *database.DB and auto-migrates the provided models.
// The database is isolated per test — each call creates a fresh DB.
//
//	db := testkit.NewTestDB(t, &user.User{})
//	repo := user.NewRepository(db.DB())
func NewTestDB(t *testing.T, models ...any) *database.DB {
	t.Helper()

	db, err := database.Connect(database.Config{
		Driver: database.DriverSQLite,
		DSN:    ":memory:",
		Silent: true,
	})
	if err != nil {
		t.Fatalf("testkit.NewTestDB: connect: %v", err)
	}

	if len(models) > 0 {
		if err := database.Register(db, models...); err != nil {
			t.Fatalf("testkit.NewTestDB: migrate: %v", err)
		}
	}

	return db
}

// --- Table ---

// Case represents a single table-driven test case.
type Case struct {
	Name string
	Run  func(t *testing.T)
}

// Table runs a slice of Cases as subtests via t.Run.
//
//	testkit.Table(t, []testkit.Case{
//	    {Name: "success", Run: func(t *testing.T) { ... }},
//	    {Name: "not found", Run: func(t *testing.T) { ... }},
//	})
func Table(t *testing.T, cases []Case) {
	t.Helper()
	for _, tc := range cases {
		tc := tc // capture
		t.Run(tc.Name, tc.Run)
	}
}

// --- Assertion helpers ---

// NoError fails the test if err is not nil.
func NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// Equal fails the test if got != want.
func Equal[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if got != want {
		t.Errorf("expected %v, got %v", want, got)
	}
}

// True fails the test if condition is false.
func True(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Errorf("expected true: %s", msg)
	}
}
