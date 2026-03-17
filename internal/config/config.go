package config

import (
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Service  *ServiceConfig
	Database *DBConfig
	Policy   *PolicyConfig
	SPRM     *SPRMConfig
	NATS     *NATSConfig
}

type ServiceConfig struct {
	Address  string `envconfig:"SVC_ADDRESS" default:":8080"`
	LogLevel string `envconfig:"SVC_LOG_LEVEL" default:"info"`
}

// DBConfig holds database configuration
type DBConfig struct {
	Type     string `envconfig:"DB_TYPE" default:"pgsql"`
	Hostname string `envconfig:"DB_HOST" default:"localhost"`
	Port     string `envconfig:"DB_PORT" default:"5432"`
	Name     string `envconfig:"DB_NAME" default:"placement-manager"`
	User     string `envconfig:"DB_USER"`
	Password string `envconfig:"DB_PASSWORD"`
}

// PolicyConfig holds policy manager configuration
type PolicyConfig struct {
	URL     string        `envconfig:"POLICY_MANAGER_EVALUATION_URL" default:"http://localhost:8081"`
	Timeout time.Duration `envconfig:"POLICY_MANAGER_EVALUATION_TIMEOUT" default:"10s"`
}

// SPRMConfig holds service provider resource manager configuration
type SPRMConfig struct {
	URL     string        `envconfig:"SP_RESOURCE_MANAGER_URL" default:"http://localhost:8082"`
	Timeout time.Duration `envconfig:"SP_RESOURCE_MANAGER_TIMEOUT" default:"10s"`
}

// NATSConfig holds NATS messaging configuration
type NATSConfig struct {
	URL          string `envconfig:"NATS_URL" default:"nats://localhost:4222"`
	Subject      string `envconfig:"NATS_STATUS_SUBJECT" default:"dcm.*"`
	StreamName   string `envconfig:"NATS_STREAM_NAME" default:"dcm-status"`
	ConsumerName string `envconfig:"NATS_CONSUMER_NAME" default:"placement-manager"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}

	// Validate and set defaults for Database.Type
	if cfg.Database.Type != "pgsql" && cfg.Database.Type != "sqlite" {
		log.Printf("WARNING: invalid DB_TYPE %q, defaulting to sqlite", cfg.Database.Type)
		cfg.Database.Type = "sqlite"
	}

	return cfg, nil
}
