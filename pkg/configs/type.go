package configs

import "time"

// Config holds reservation-service configuration.
// All fields are loaded from environment variables with safe defaults.
type Config struct {
	AppName  string `env:"APP_NAME" envDefault:"reservation-service"`
	AppEnv   string `env:"APP_ENV" envDefault:"local"`
	AppPort  string `env:"APP_PORT" envDefault:"8081"`  // REST port (mini app)
	GrpcPort string `env:"GRPC_PORT" envDefault:"9090"` // gRPC port (s2s)

	DbHost     string `env:"DB_HOST" envDefault:"localhost"`
	DbPort     string `env:"DB_PORT" envDefault:"5432"`
	DbUsername string `env:"DB_USERNAME" envDefault:"postgres"`
	DbPassword string `env:"DB_PASSWORD" envDefault:"postgres"`
	DbName     string `env:"DB_NAME" envDefault:"reservation_service"`
	DbMaxOpen  int    `env:"DB_MAX_OPEN" envDefault:"25"`
	DbMaxIdle  int    `env:"DB_MAX_IDLE" envDefault:"10"`

	RedisHost      string `env:"REDIS_HOST" envDefault:"localhost"`
	RedisPort      string `env:"REDIS_PORT" envDefault:"6379"`
	RedisPassword  string `env:"REDIS_PASSWORD" envDefault:""`
	RedisDB        int    `env:"REDIS_DB" envDefault:"2"`
	RedisAppConfig string `env:"REDIS_APP_CONFIG" envDefault:"reservation-service"`

	RabbitURL      string `env:"RABBIT_URL" envDefault:"amqp://guest:guest@localhost:5672/"`
	RabbitExchange string `env:"RABBIT_EXCHANGE" envDefault:"parkirpintar.events"`
	RabbitQueue    string `env:"RABBIT_QUEUE" envDefault:"reservation-service"`

	UserGrpcAddr    string `env:"USER_GRPC_ADDR" envDefault:"localhost:9094"`
	BillingGrpcAddr string `env:"BILLING_GRPC_ADDR" envDefault:"localhost:9091"`
	// BillingMode: "grpc" — call billing-service over gRPC (default for prod-like dev)
	//              "stub" — log only; useful when billing isn't running yet
	BillingMode string `env:"BILLING_MODE" envDefault:"grpc"`

	OTLPEndpoint string `env:"OTLP_ENDPOINT" envDefault:"localhost:4317"`

	// RS256 public key PEM from super-app. Empty = skip signature check (dev only).
	SuperAppJWTPubKey string `env:"SUPER_APP_JWT_PUBLIC_KEY_PEM" envDefault:""`

	// Reservation behaviour knobs
	HoldDuration         time.Duration `env:"HOLD_DURATION" envDefault:"60m"`
	GeofenceRadiusMeters float64       `env:"GEOFENCE_RADIUS_METERS" envDefault:"150"`
	BuildingLat          float64       `env:"CHECK_IN_BUILDING_LAT" envDefault:"-6.2088"`
	BuildingLng          float64       `env:"CHECK_IN_BUILDING_LNG" envDefault:"106.8456"`
}

// ConfigLoader controls the source of config (env file path, etc.).
type ConfigLoader struct {
	Env     string
	EnvFile string
}
