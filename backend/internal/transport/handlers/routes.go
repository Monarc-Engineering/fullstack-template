package handlers

import (
	"net/http"
	"net/http/pprof"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "backend/docs/swagger"
	"backend/internal/transport/middleware"
)

// RegisterRoutes creates the Gin engine, applies middleware, and registers all routes.
// rps and burst configure IP-based rate limiting; pass rps<=0 to disable.
// sentryDSN enables Sentry error tracking; pass empty string to disable.
// allowedOrigins is the list of CORS allowed origins; defaults to localhost:3000 when empty.
// Firebase auth is read from h.verifier; nil disables auth (dev only).
func (h *Handler) RegisterRoutes(rps float64, burst int, sentryDSN string, allowedOrigins []string) http.Handler {
	r := gin.New()

	r.Use(middleware.RequestID())
	r.Use(middleware.SentryMiddleware(sentryDSN))

	// Use Gin's colorful logger locally; structured slog logger in staging/production.
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

	// In staging/production, restrict /metrics to loopback and RFC 1918 addresses
	// so Prometheus can scrape from the internal network but external clients cannot.
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

	// Email template preview — debug/local only (backend/docs/email.md).
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
