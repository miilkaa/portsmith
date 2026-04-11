// Package server provides a production-ready Gin HTTP server for portsmith applications.
//
// # Creating a server
//
//	srv := server.New(server.Config{
//	    Port: 8080,
//	    Mode: "release",  // or "debug"
//	})
//
// # Registering routes
//
//	v1 := srv.Router().Group("/api/v1")
//	userHandler.Routes(v1)
//	orderHandler.Routes(v1)
//
// # Starting the server
//
//	if err := srv.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Built-in endpoints
//
//	GET /health  →  200 {"status":"ok"}
//
// # Middleware (applied automatically)
//
//   - Recovery: catches panics → 500 with JSON body
//   - RequestID: generates X-Request-ID header if not present
//   - CORS: permissive defaults (restrict AllowOrigins in production)
//   - Errors: converts apperrors.AppError → HTTP status + JSON body
//
// # Binding and validation
//
// Use BindAndValidate in handlers instead of c.ShouldBindJSON:
//
//	var req CreateUserRequest
//	if err := server.BindAndValidate(c, &req); err != nil {
//	    return  // 400 already written
//	}
//
// # Error handling
//
// Attach errors via c.Error() — the middleware handles the response:
//
//	user, err := svc.GetByID(ctx, id)
//	if err != nil {
//	    _ = c.Error(err)  // apperrors.NotFound → 404
//	    return
//	}
package server
