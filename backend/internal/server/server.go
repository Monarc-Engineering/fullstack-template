package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
	"github.com/prometheus/client_golang/prometheus"
	goredis "github.com/redis/go-redis/v9"

	"backend/internal/bootstrap"
	"backend/internal/infrastructure/database/postgres"
	"backend/internal/infrastructure/email"
	"backend/internal/infrastructure/ws"
	"backend/internal/transport/handlers"
	"backend/internal/usecase"
)

// NewServer wires all layers and returns a configured *http.Server.
func NewServer(app *bootstrap.App, hub *ws.Hub) (*http.Server, error) {
	switch app.Config.Env {
	case "staging", "production":
		gin.SetMode(gin.ReleaseMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	// Suppress all [GIN-debug] output (banner + route table); docs URL is logged in main instead.
	gin.DebugPrintFunc = func(_ string, _ ...any) {}
	gin.DebugPrintRouteFunc = func(_, _, _ string, _ int) {}

	healthRepo := postgres.NewHealthRepository(app.DB)
	healthUC := usecase.NewHealthUseCase(healthRepo)

	var fcmTokenRepo *postgres.FCMTokenRepository
	if app.Firebase != nil {
		fcmTokenRepo = postgres.NewFCMTokenRepository(app.DB)
	}

	userRepo := postgres.NewUserRepository(app.DB)

	// Build Asynqmon UI handler when Redis is available.
	var queueUI http.Handler
	if app.Config.RedisURL != "" {
		redisOpt, err := goredis.ParseURL(app.Config.RedisURL)
		if err == nil {
			queueUI = asynqmon.New(asynqmon.Options{
				RootPath: "/admin/queues",
				RedisConnOpt: asynq.RedisClientOpt{
					Addr:     redisOpt.Addr,
					Password: redisOpt.Password,
					DB:       redisOpt.DB,
				},
			})
		}
	}

	h := handlers.NewHandler(healthUC, app.Firebase, hub, app.Enqueuer, queueUI, app.FCMSender, fcmTokenRepo, app.EmailSender, app.StorageService, app.GeoLocator, app.StreamProducer, userRepo).
		WithEmailPreview(email.NewPreview("/dev/emails"))

	// Register DB pool metrics collector.
	// AlreadyRegisteredError is silenced — only the first registration wins
	// (safe for test suites that call NewServer more than once).
	dbCollector := postgres.NewDBStatsCollector(app.DB)
	if err := prometheus.Register(dbCollector); err != nil {
		var are prometheus.AlreadyRegisteredError
		if !errors.As(err, &are) {
			return nil, fmt.Errorf("server: register db metrics collector: %w", err)
		}
	}

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", app.Config.Port),
		Handler:      h.RegisterRoutes(app.Config.RateLimitRPS, app.Config.RateLimitBurst, app.Config.SentryDSN, app.Config.CORSAllowedOrigins),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}, nil
}
