package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL              string
	ExchangeRateAPIKey string
	Port               string

	WorkerInterval     time.Duration
	WorkerPoolSize     int
	PollBatchSize      int
	JobsChannelSize    int
	WorkerStaleAfter   time.Duration
	WorkerJobTimeout   time.Duration
	ResetStaleInterval time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	ShutdownTimeout    time.Duration

	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	DBConnMaxIdleTime time.Duration
	DBPingTimeout     time.Duration
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
		DBURL:              getEnv("DATABASE_URL", ""),
		ExchangeRateAPIKey: getEnv("EXCHANGE_RATE_API_KEY", ""),
		Port:               getEnv("PORT", "8080"),
		WorkerInterval:     getEnvDuration("WORKER_INTERVAL", 10*time.Second),
		WorkerPoolSize:     getEnvInt("WORKER_POOL_SIZE", 5),
		PollBatchSize:      getEnvInt("POLL_BATCH_SIZE", 20),
		JobsChannelSize:    getEnvInt("JOBS_CHANNEL_SIZE", 100),
		WorkerStaleAfter:   getEnvDuration("WORKER_STALE_AFTER", 5*time.Minute),
		WorkerJobTimeout:   getEnvDuration("WORKER_JOB_TIMEOUT", 30*time.Second),
		ResetStaleInterval: getEnvDuration("RESET_STALE_INTERVAL", 1*time.Minute),
		ReadTimeout:        getEnvDuration("HTTP_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:       getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
		ShutdownTimeout:    getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		DBMaxOpenConns:     getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:     getEnvInt("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifetime:  getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		DBConnMaxIdleTime:  getEnvDuration("DB_CONN_MAX_IDLE_TIME", 1*time.Minute),
		DBPingTimeout:      getEnvDuration("DB_PING_TIMEOUT", 3*time.Second),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
