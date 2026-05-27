package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bongcloud/chess/internal/cache"
	"github.com/bongcloud/chess/internal/config"
	"github.com/bongcloud/chess/internal/database"
	"github.com/bongcloud/chess/internal/middleware"
	"github.com/bongcloud/chess/internal/ws"
	"github.com/bongcloud/chess/pkg/response"
)

// Server owns the Gin engine, HTTP server, and all injected infrastructure
// dependencies. Domain modules register their routes against the engine here.
type Server struct {
	cfg      *config.Config
	log      *zap.Logger
	engine   *gin.Engine
	http     *http.Server
	mongo    *database.MongoDB
	redis    *cache.RedisClient
	upgrader *ws.Upgrader
}

// Dependencies bundles all infrastructure dependencies for clean injection.
type Dependencies struct {
	Config   *config.Config
	Log      *zap.Logger
	Mongo    *database.MongoDB
	Redis    *cache.RedisClient
	Upgrader *ws.Upgrader
}

// New builds and configures a fully wired Server ready to call ListenAndServe.
func New(deps Dependencies) *Server {
	if deps.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	s := &Server{
		cfg:      deps.Config,
		log:      deps.Log,
		engine:   engine,
		mongo:    deps.Mongo,
		redis:    deps.Redis,
		upgrader: deps.Upgrader,
	}

	s.registerMiddleware()
	s.registerRoutes()

	s.http = &http.Server{
		Addr:         fmt.Sprintf(":%d", deps.Config.App.Port),
		Handler:      engine,
		ReadTimeout:  deps.Config.Server.ReadTimeout,
		WriteTimeout: deps.Config.Server.WriteTimeout,
		IdleTimeout:  deps.Config.Server.IdleTimeout,
	}

	return s
}

// registerMiddleware attaches global middleware to the Gin engine.
func (s *Server) registerMiddleware() {
	s.engine.Use(
		requestid.New(),
		middleware.Recovery(s.log),
		middleware.RequestLogger(s.log),
		middleware.CORS([]string{"*"}), // Tighten in production
	)
}

// registerRoutes mounts all application routes.
// Domain routers are registered here as they are added.
func (s *Server) registerRoutes() {
	// System routes 
	s.engine.GET("/health", s.handleHealth)
	s.engine.GET("/ready", s.handleReadiness)

	//  API v1
	// v1 := s.engine.Group("/api/v1")
	// {
	//     game.RegisterRoutes(v1, ...)
	//     user.RegisterRoutes(v1, ...)
	//     matchmaking.RegisterRoutes(v1, ...)
	// }

	//  WebSocket
	// ws := s.engine.Group("/ws")
	// {
	//     game.RegisterWSRoutes(ws, ...)
	// }

	s.log.Info("routes registered")
}

// Start begins accepting connections. Blocks until the server is stopped.
// Returns http.ErrServerClosed on graceful shutdown, other errors otherwise.
func (s *Server) Start() error {
	s.log.Info("server starting",
		zap.String("addr", s.http.Addr),
		zap.String("env", s.cfg.App.Env),
	)
	return s.http.ListenAndServe()
}

// Shutdown performs a graceful drain of in-flight requests within the
// configured timeout, then closes the listener.
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("server shutting down gracefully")
	return s.http.Shutdown(ctx)
}

// System handlers 
// handleHealth is a lightweight liveness probe. Returns 200 if the process
// is alive. Intentionally does not check external dependencies.
func (s *Server) handleHealth(c *gin.Context) {
	response.OK(c, gin.H{
		"status":  "ok",
		"service": s.cfg.App.Name,
		"version": s.cfg.App.Version,
	})
}

// handleReadiness is a deeper readiness probe that verifies all critical
// infrastructure dependencies are reachable. Returns 503 on failure.
func (s *Server) handleReadiness(c *gin.Context) {
	ctx := c.Request.Context()
	checks := map[string]string{}
	healthy := true

	if err := s.mongo.Ping(ctx); err != nil {
		checks["mongo"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		checks["mongo"] = "ok"
	}

	if err := s.redis.Ping(ctx); err != nil {
		checks["redis"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		checks["redis"] = "ok"
	}

	payload := gin.H{
		"service": s.cfg.App.Name,
		"checks":  checks,
	}

	if !healthy {
		response.ServiceUnavailable(c, "one or more dependencies are unhealthy")
		return
	}

	response.OK(c, payload)
}