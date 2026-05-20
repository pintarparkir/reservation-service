// reservation-service entry point.
//
// Wires:
//   - configs → logger → otel
//   - postgres + redis + rabbitmq publisher
//   - repositories (reservation, spot, outbox)
//   - usecase (with Redis lock + billing client stub)
//   - REST server (mini-app interface)
//   - background workers (no-show expirer, outbox publisher)
//
// The gRPC server registration is conditional on `buf generate` having run to
// produce api/proto/reservation/v1/*.pb.go. Until then, the service exposes
// only its REST surface; it can still be exercised end-to-end via curl.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	reshttp "github.com/farid/reservation-service/internal/reservation/handler/http"
	resrepo "github.com/farid/reservation-service/internal/reservation/repository/postgres"
	resuc "github.com/farid/reservation-service/internal/reservation/usecase"
	"github.com/farid/reservation-service/internal/reservation/worker"

	"github.com/farid/reservation-service/pkg/configs"
	pgdb "github.com/farid/reservation-service/pkg/db/postgres"
	"github.com/farid/reservation-service/pkg/grpcclient"
	"github.com/farid/reservation-service/pkg/lock"
	"github.com/farid/reservation-service/pkg/logger"
	pkgOtel "github.com/farid/reservation-service/pkg/otel"
	"github.com/farid/reservation-service/pkg/rabbit"
	pkgRedis "github.com/farid/reservation-service/pkg/redis"
)

func main() {
	cfg := configs.NewConfig(configs.ConfigLoader{Env: os.Getenv("PROJECT_ENV")})
	if err := logger.NewLogger(cfg.AppName, cfg.AppEnv); err != nil {
		panic(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	otel := pkgOtel.NewOpenTelemetry(cfg.OTLPEndpoint, "reservation", cfg.AppEnv)
	defer func() {
		if err := otel.EndAPM(); err != nil {
			fmt.Fprintln(os.Stderr, "otel shutdown:", err)
		}
	}()

	// ── Infra ────────────────────────────────────────────────────────────────
	db, err := pgdb.NewPostgresDB(pgdb.PostgresDsn{
		Host: cfg.DbHost, Port: cfg.DbPort, User: cfg.DbUsername, Password: cfg.DbPassword, Db: cfg.DbName,
		MaxOpen: cfg.DbMaxOpen, MaxIdle: cfg.DbMaxIdle,
	})
	if err != nil {
		logger.Fatal(ctx, "postgres init failed", map[string]interface{}{logger.ErrorKey: err.Error()})
	}
	defer db.Close()

	cache := pkgRedis.InitConnection(cfg.RedisDB, cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword, cfg.RedisAppConfig)
	if err := cache.Ping(ctx); err != nil {
		logger.Warn(ctx, "redis ping failed (continuing degraded)",
			map[string]interface{}{logger.ErrorKey: err.Error()})
	}

	publisher, err := rabbit.NewPublisher(cfg.RabbitURL, cfg.RabbitExchange)
	if err != nil {
		logger.Fatal(ctx, "rabbitmq init failed", map[string]interface{}{logger.ErrorKey: err.Error()})
	}
	defer publisher.Close()

	// ── Domain wiring ────────────────────────────────────────────────────────
	resvRepo := resrepo.NewReservationRepository(db)
	spotRepo := resrepo.NewSpotRepository(db)
	obRepo := resrepo.NewOutboxRepository(db)

	// Billing client — pick gRPC or stub based on cfg.BillingMode. Default
	// is gRPC; the stub keeps the dev loop working when billing-service
	// isn't running.
	var billing grpcclient.BillingClient
	if cfg.BillingMode == "grpc" {
		bConn, err := grpcclient.Dial(cfg.BillingGrpcAddr)
		if err != nil {
			logger.Warn(ctx, "billing-service grpc dial failed; falling back to stub",
				map[string]interface{}{"addr": cfg.BillingGrpcAddr, logger.ErrorKey: err.Error()})
			billing = grpcclient.NewBillingStub()
		} else {
			defer bConn.Close()
			billing = grpcclient.NewBillingGrpc(bConn)
			logger.Info(ctx, "billing client: grpc", map[string]interface{}{"addr": cfg.BillingGrpcAddr})
		}
	} else {
		billing = grpcclient.NewBillingStub()
		logger.Info(ctx, "billing client: stub (BILLING_MODE != grpc)", nil)
	}

	uc := resuc.NewReservationUsecase(
		resvRepo, spotRepo,
		lock.New(cache),
		billing,
		resuc.Config{
			HoldDuration:         cfg.HoldDuration,
			GeofenceRadiusMeters: cfg.GeofenceRadiusMeters,
			BuildingLat:          cfg.BuildingLat,
			BuildingLng:          cfg.BuildingLng,
		},
	)

	// ── Background workers ───────────────────────────────────────────────────
	go worker.NewNoShowExpirer(resvRepo).Run(ctx)
	go worker.NewOutboxPublisher(obRepo, publisher).Run(ctx)

	// ── HTTP server ──────────────────────────────────────────────────────────
	if cfg.AppEnv == "local" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery(), cors.Default())
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	reshttp.RegisterReservationHandler(router.Group("/v1"), uc, cfg.SuperAppJWTPubKey)

	httpSrv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info(ctx, fmt.Sprintf("reservation HTTP listening on :%s", cfg.AppPort), nil)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(ctx, "http listen failed", map[string]interface{}{logger.ErrorKey: err.Error()})
		}
	}()

	// ── Graceful shutdown ────────────────────────────────────────────────────
	<-ctx.Done()
	logger.Info(context.Background(), "shutdown signal received", nil)

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutCtx); err != nil {
		logger.Error(context.Background(), "http shutdown error", map[string]interface{}{logger.ErrorKey: err.Error()})
	}
	if err := logger.Sync(); err != nil {
		fmt.Fprintln(os.Stderr, "logger sync:", err)
	}
}
