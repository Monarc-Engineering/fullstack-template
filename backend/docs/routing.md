---
topic: routing
last_verified: 2026-10-23
sources:
  - internal/transport/handlers/handler.go
  - internal/transport/handlers/routes.go
  - internal/transport/handlers/hello_handler.go
  - internal/transport/handlers/health_handler.go
  - internal/transport/handlers/auth_handler.go
  - internal/transport/handlers/me_handler.go
  - internal/transport/handlers/pprof_handler_test.go
  - internal/transport/handlers/validation.go
  - internal/transport/handlers/response.go
  - internal/transport/middleware/logger.go
  - internal/transport/middleware/request_id.go
  - internal/server/server.go
  - cmd/api/main.go
---

# Routing

## Handler struct
```go
// internal/transport/handlers/handler.go
type Handler struct {
    healthUC       usecase.HealthUseCase
    verifier       usecase.FirebaseTokenVerifier // nil disables auth (dev only)
    hub            *ws.Hub
    enqueuer       usecase.Enqueuer           // nil when REDIS_URL is not set
    queueUI        http.Handler               // nil disables /admin/queues route
    fcmSender      usecase.NotificationSender // nil when Firebase is not configured
    fcmTokenRepo   usecase.FCMTokenRepository // nil when Firebase is not configured
    emailSender    usecase.EmailSender        // nil when MAILJET_API_KEY is not set
    storageService usecase.StorageService     // nil when R2_ACCOUNT_ID is not set
    geoLocator     usecase.GeoLocator         // nil when geo client is not configured
    streamProducer usecase.StreamProducer     // nil when REDIS_URL is not set
    userRepo       usecase.UserRepository     // nil when not wired
}

func NewHandler(
    healthUC usecase.HealthUseCase,
    verifier usecase.FirebaseTokenVerifier,
    hub *ws.Hub,
    enqueuer usecase.Enqueuer,
    queueUI http.Handler,
    fcmSender usecase.NotificationSender,
    fcmTokenRepo usecase.FCMTokenRepository,
    emailSender usecase.EmailSender,
    storageService usecase.StorageService,
    geoLocator usecase.GeoLocator,
    streamProducer usecase.StreamProducer,
    userRepo usecase.UserRepository,
) *Handler
```
The `Handler` struct holds use case interfaces and infrastructure dependencies — not `*sql.DB` directly. `verifier` is stored on the struct (not passed to `RegisterRoutes`) so the WebSocket handler can read it inline for query-param auth. `fcmTokenRepo` and `fcmSender` are nil when `FIREBASE_PROJECT_ID` is not set; their routes are only registered when non-nil. `userRepo` is always wired (constructed unconditionally in `server.NewServer`).

## Wiring (server.go)
`internal/server/server.go` contains `NewServer(app *bootstrap.App, hub *ws.Hub) (*http.Server, error)` — wiring only, no logic.
It receives the already-validated `*bootstrap.App` (which holds `*sql.DB`, `Cache`, `Enqueuer`, `Firebase`, `FCMSender`, and `Config`) and a `*ws.Hub`, constructs the repository, use case, and handler in order, then returns a configured `*http.Server`. Errors from initialisation steps are returned to the caller.

```go
healthRepo := postgres.NewHealthRepository(app.DB)
healthUC := usecase.NewHealthUseCase(healthRepo)

var fcmTokenRepo *postgres.FCMTokenRepository
if app.Firebase != nil {
    fcmTokenRepo = postgres.NewFCMTokenRepository(app.DB)
}

userRepo := postgres.NewUserRepository(app.DB)

var queueUI http.Handler
if app.Config.RedisURL != "" {
    // parse URL and build asynqmon.New(...)
}

h := handlers.NewHandler(healthUC, app.Firebase, hub, app.Enqueuer, queueUI, app.FCMSender, fcmTokenRepo, app.EmailSender, app.StorageService, app.GeoLocator, app.StreamProducer, userRepo)

// Register DB pool metrics collector (AlreadyRegisteredError is silenced).
prometheus.Register(postgres.NewDBStatsCollector(app.DB))

return &http.Server{
    Addr:         fmt.Sprintf(":%d", app.Config.Port),
    Handler:      h.RegisterRoutes(app.Config.RateLimitRPS, app.Config.RateLimitBurst, app.Config.SentryDSN, app.Config.CORSAllowedOrigins),
    IdleTimeout:  time.Minute,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 30 * time.Second,
}, nil
```

## Route registration
All routes are registered in `RegisterRoutes()` on `*Handler`, which returns `http.Handler`.
`rps` and `burst` come from `bootstrap.Config` (env vars `RATE_LIMIT_RPS` / `RATE_LIMIT_BURST`); pass `rps=0` to disable.
`allowedOrigins` comes from `bootstrap.Config.CORSAllowedOrigins` (env var `CORS_ALLOWED_ORIGINS`); defaults to `["http://localhost:3000"]` when the env var is not set.
`h.verifier` (set via `NewHandler`) controls Firebase auth — the verifier is read from the struct, not passed to `RegisterRoutes`; a `nil` verifier skips Firebase auth (development only — see [auth](auth.md)).

```go
func (h *Handler) RegisterRoutes(rps float64, burst int, sentryDSN string, allowedOrigins []string) http.Handler {
    r := gin.New()

    r.Use(middleware.RequestID())
    r.Use(middleware.SentryMiddleware(sentryDSN))

    // Gin's colorful logger locally; structured slog logger in staging/production.
    if gin.Mode() == gin.DebugMode {
        r.Use(gin.Recovery(), gin.Logger())
    } else {
        r.Use(gin.Recovery(), middleware.Logger())
    }

    r.Use(middleware.PrometheusMiddleware())
    r.Use(middleware.RateLimit(rps, burst))

    r.Use(cors.New(cors.Config{
        AllowOrigins:     allowedOrigins,
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
        AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
        AllowCredentials: true,
    }))

    r.GET("/", h.HelloWorldHandler)
    r.GET("/health", h.HealthHandler)
    r.GET("/ws", h.WsHandler)

    // /metrics restricted to loopback/RFC 1918 in staging/production.
    if gin.Mode() == gin.ReleaseMode {
        r.GET("/metrics", middleware.LocalNetworkOnly(), gin.WrapH(promhttp.Handler()))
    } else {
        r.GET("/metrics", gin.WrapH(promhttp.Handler()))
    }

    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // /debug/pprof — always restricted to loopback / RFC 1918 (never public).
    debug := r.Group("/debug/pprof", middleware.LocalNetworkOnly())
    {
        debug.GET("/", gin.WrapF(pprof.Index))
        debug.GET("/cmdline", gin.WrapF(pprof.Cmdline))
        debug.GET("/profile", gin.WrapF(pprof.Profile))
        debug.GET("/symbol", gin.WrapF(pprof.Symbol))
        debug.POST("/symbol", gin.WrapF(pprof.Symbol))
        debug.GET("/trace", gin.WrapF(pprof.Trace))
        debug.GET("/:profile", gin.WrapF(pprof.Index))
    }

    // Asynqmon job-monitoring UI — debug/local only.
    if gin.Mode() == gin.DebugMode && h.queueUI != nil {
        r.GET("/admin/queues", gin.WrapH(h.queueUI))
        r.Any("/admin/queues/*path", gin.WrapH(h.queueUI))
    }

    // Email template preview — debug/local only (docs/email.md).
    if gin.Mode() == gin.DebugMode && h.emailPreview != nil {
        r.GET("/dev/emails", gin.WrapH(h.emailPreview))
        r.GET("/dev/emails/*path", gin.WrapH(h.emailPreview))
    }

    api := r.Group("/api/v1")
    if h.verifier != nil {
        api.Use(middleware.FirebaseAuth(h.verifier))
    }
    if h.geoLocator != nil {
        api.Use(middleware.GeoFromRequest(h.geoLocator))
    }
    api.GET("/me", h.MeHandler)
    if h.userRepo != nil {
        api.PATCH("/me", h.UpdateMeHandler)
        api.DELETE("/me", h.DeleteMeHandler)
    }

    if h.fcmTokenRepo != nil {
        api.POST("/fcm/register", h.RegisterFCMToken)
        api.DELETE("/fcm/unregister", h.UnregisterFCMToken)
    }

    if h.storageService != nil {
        api.POST("/storage/presign", h.PresignHandler)
        api.DELETE("/storage/:key", h.DeleteObjectHandler)
    }

    return r
}
```

## Handler pattern
All handlers are methods on `*Handler`. Always use `*gin.Context`.
Use `JSON`, `JSONStatus`, and `JSONError` from `internal/transport/handlers/response.go` — do not call `c.JSON` directly in handlers (see [error-handling](error-handling.md) for the envelope shape).

```go
func (h *Handler) myHandler(c *gin.Context) {
    result, err := h.someUC.DoSomething(c.Request.Context(), ...)
    if err != nil {
        JSONError(c, http.StatusInternalServerError, "internal_error", "internal error")
        return
    }
    JSON(c, result)
}
```

## CORS configuration
Pre-configured in `RegisterRoutes()` via `github.com/gin-contrib/cors`.
Allowed origins come from `allowedOrigins []string` (4th parameter), sourced from `bootstrap.Config.CORSAllowedOrigins` — set via the `CORS_ALLOWED_ORIGINS` env var (comma-separated, defaults to `http://localhost:3000`).
Allowed methods: GET, POST, PUT, DELETE, OPTIONS, PATCH.
`AllowCredentials: true` — cookies and auth headers pass through.

## Existing routes
| Method | Path | Auth | Handler | File |
|---|---|---|---|---|
| GET | `/` | none | `HelloWorldHandler` — returns `{"message": "Hello World"}` | `hello_handler.go` |
| GET | `/health` | none | `HealthHandler` — returns `HealthStats`; 503 when DB is down | `health_handler.go` |
| GET | `/ws` | `?token=` query param | `WsHandler` — upgrades to WebSocket; 401 when token missing/invalid | `ws_handler.go` |
| GET | `/metrics` | `LocalNetworkOnly()` in release mode | Prometheus scrape endpoint; unrestricted in debug mode | `routes.go` |
| GET | `/swagger/*any` | none | Swagger UI | `routes.go` |
| GET | `/debug/pprof/` | `LocalNetworkOnly()` (always) | pprof index — `net/http/pprof.Index` | `routes.go` |
| GET | `/debug/pprof/cmdline` | `LocalNetworkOnly()` (always) | pprof cmdline | `routes.go` |
| GET | `/debug/pprof/profile` | `LocalNetworkOnly()` (always) | CPU profile | `routes.go` |
| GET | `/debug/pprof/symbol` | `LocalNetworkOnly()` (always) | pprof symbol lookup | `routes.go` |
| POST | `/debug/pprof/symbol` | `LocalNetworkOnly()` (always) | pprof symbol lookup | `routes.go` |
| GET | `/debug/pprof/trace` | `LocalNetworkOnly()` (always) | execution trace | `routes.go` |
| GET | `/debug/pprof/:profile` | `LocalNetworkOnly()` (always) | named profile (heap, goroutine, etc.) | `routes.go` |
| GET | `/admin/queues` | none (debug mode only) | Asynqmon job-monitoring UI | `routes.go` |
| GET | `/dev/emails`, `/dev/emails/*path` | none (debug mode only, `WithEmailPreview`) | Email template preview (`docs/email.md`) | `routes.go` |
| GET | `/api/v1/me` | FirebaseAuth header | `MeHandler` — returns verified `FirebaseToken` claims | `auth_handler.go` |
| PATCH | `/api/v1/me` | FirebaseAuth header | `UpdateMeHandler` — upserts user profile; returns `domain.User` | `me_handler.go` |
| DELETE | `/api/v1/me` | FirebaseAuth header | `DeleteMeHandler` — deletes user profile record; 204 on success | `me_handler.go` |
| POST | `/api/v1/fcm/register` | FirebaseAuth header | `RegisterFCMToken` — stores device FCM token | `fcm_handler.go` |
| DELETE | `/api/v1/fcm/unregister` | FirebaseAuth header | `UnregisterFCMToken` — removes device FCM token | `fcm_handler.go` |
| POST | `/api/v1/storage/presign` | FirebaseAuth header | `PresignHandler` — returns a presigned R2 upload URL | `storage_handler.go` |
| DELETE | `/api/v1/storage/:key` | FirebaseAuth header | `DeleteObjectHandler` — deletes an R2 object | `storage_handler.go` |

FCM routes are only registered when `h.fcmTokenRepo != nil` (i.e., `FIREBASE_PROJECT_ID` is set).
Storage routes are only registered when `h.storageService != nil` (i.e., `R2_ACCOUNT_ID` is set).
`PATCH /api/v1/me` and `DELETE /api/v1/me` are registered when `h.userRepo != nil` (always wired).
`/debug/pprof/*` routes are always registered (in both debug and release modes) but are unconditionally gated by `LocalNetworkOnly()` — they are never reachable from external clients. They are intentionally excluded from Swagger annotations because `gin.WrapF` handlers carry no swaggo metadata.

## Graceful shutdown
Wired in `cmd/api/main.go` via `signal.NotifyContext` for SIGINT/SIGTERM.
5-second shutdown timeout. Server notifies `done chan bool` when complete.
`main()` calls `bootstrap.Run(ctx)` first; on failure it writes to stderr and calls `os.Exit(1)`.
`server.NewServer` returns `(*http.Server, error)` — the caller in `cmd/api/main.go` checks the error and exits on failure.
Do not add shutdown logic to `internal/` — it belongs in `cmd/`.

## Adding a new route — checklist
1. Define domain type in `internal/domain/` if needed.
2. Add use case interface + implementation in `internal/usecase/`.
3. Add repository interface in the use case package; implement in `internal/repository/postgres/`.
4. Add use case field to `Handler` struct in `handler.go`; update `NewHandler` signature.
5. Wire the new repository → use case → handler in `server.NewServer()`.
6. Register the route in `RegisterRoutes()`: `r.METHOD("/path", h.handlerName)`.
7. Add handler as `func (h *Handler) handlerName(c *gin.Context)` in its own file.
8. Always pass `c.Request.Context()` to use case calls.
9. For request body: use the `bindJSON(c, &input) bool` helper (returns `false` and writes 400 on error). For query params: use `bindQuery(c, &input) bool`. Both helpers are defined in `internal/transport/handlers/validation.go` and return stable error messages ("invalid request body" / "invalid query parameters") rather than exposing raw validation errors.
10. For path params: `c.Param("id")`.
