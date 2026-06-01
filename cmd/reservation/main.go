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
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	resconsumer "github.com/farid/reservation-service/internal/reservation/consumer"
	"github.com/farid/reservation-service/internal/reservation/model"
	reshttp "github.com/farid/reservation-service/internal/reservation/handler/http"
	resrepo "github.com/farid/reservation-service/internal/reservation/repository/postgres"
	resuc "github.com/farid/reservation-service/internal/reservation/usecase"
	"github.com/farid/reservation-service/internal/reservation/worker"

	"github.com/farid/reservation-service/pkg/configs"
	pgdb "github.com/farid/reservation-service/pkg/db/postgres"
	"github.com/farid/reservation-service/pkg/lock"
	"github.com/farid/reservation-service/pkg/logger"
	pkgOtel "github.com/farid/reservation-service/pkg/otel"
	"github.com/farid/reservation-service/pkg/rabbit"
	"github.com/farid/reservation-service/pkg/rate"
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
	if err := otel.RegisterRuntimeMetrics(); err != nil {
		logger.Error(ctx, "failed to register runtime metrics", map[string]interface{}{logger.ErrorKey: err.Error()})
	}

	// ── Infra ────────────────────────────────────────────────────────────────
	db, err := pgdb.NewPostgresDB(pgdb.PostgresDsn{
		Host: cfg.DbHost, Port: cfg.DbPort, User: cfg.DbUsername, Password: cfg.DbPassword, Db: cfg.DbName,
		MaxOpen: cfg.DbMaxOpen, MaxIdle: cfg.DbMaxIdle,
	})
	if err != nil {
		logger.Fatal(ctx, "postgres init failed", map[string]interface{}{logger.ErrorKey: err.Error()})
	}
	defer func() { _ = db.Close() }()

	cache := pkgRedis.InitConnection(cfg.RedisDB, cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword, cfg.RedisAppConfig)
	if pingErr := cache.Ping(ctx); pingErr != nil {
		logger.Warn(ctx, "redis ping failed (continuing degraded)",
			map[string]interface{}{logger.ErrorKey: pingErr.Error()})
	}

	limiter := rate.New(cache)

	publisher, err := rabbit.NewPublisher(cfg.RabbitURL, cfg.RabbitExchange)
	if err != nil {
		logger.Fatal(ctx, "rabbitmq init failed", map[string]interface{}{logger.ErrorKey: err.Error()})
	}
	defer publisher.Close()

	// ── Domain wiring ────────────────────────────────────────────────────────
	resvRepo := resrepo.NewReservationRepository(db)
	spotRepo := resrepo.NewSpotRepository(db)
	obRepo := resrepo.NewOutboxRepository(db)

	// Billing is reached asynchronously via the outbox/RabbitMQ pipeline;
	// no synchronous gRPC client is wired into the usecase. The async path
	// is documented in docs/architecture/service-communication/.

	uc := resuc.NewReservationUsecase(
		resvRepo, spotRepo,
		lock.New(cache),
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
	go worker.NewReconciler(db).Run(ctx)

	// ── RabbitMQ consumer (payment events from billing-service) ──────────────
	paymentConsumer := resconsumer.NewBillingPaymentConsumer(resvRepo)
	subscriber, err := rabbit.NewSubscriber(cfg.RabbitURL, cfg.RabbitExchange, cfg.RabbitQueue+"-payment",
		[]string{
			model.EvtPaymentSuccess,
			model.EvtPaymentFailed,
		},
	)
	if err != nil {
		logger.Fatal(ctx, "rabbitmq payment subscriber init failed", map[string]interface{}{logger.ErrorKey: err.Error()})
	}
	defer subscriber.Close()
	go func() {
		logger.Info(ctx, "payment consumer: subscribing to billing.payment.*.v1", map[string]interface{}{"queue": cfg.RabbitQueue + "-payment"})
		if err := subscriber.Consume(ctx, paymentConsumer.HandlePaymentConfirmed, paymentConsumer.HandlePaymentFailed); err != nil {
			logger.Error(ctx, "payment consumer: stopped", map[string]interface{}{logger.ErrorKey: err.Error()})
		}
	}()

	// ── HTTP server ──────────────────────────────────────────────────────────
	if cfg.AppEnv == "local" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(otelgin.Middleware(cfg.AppName))
	router.Use(gin.Recovery(), cors.Default())
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	reshttp.RegisterReservationHandler(router.Group("/v1"), uc, cfg.SuperAppJWTPubKey, limiter)

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
