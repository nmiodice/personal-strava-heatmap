package backend

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sethvargo/go-envconfig"
)

type HttpServerConfig struct {
	Port int `env:"PORT,default=8080"`
}

type HttpClientConfig struct {
	Timeout time.Duration `env:"HTTP_CLIENT_TIMEOUT_SECONDS,default=5s"`
}

type StravaAppConfig struct {
	ClientID     string `env:"STRAVA_CLIENT_ID,required"`
	ClientSecret string `env:"STRAVA_CLIENT_SECRET,required"`
}

type DatabaseConfig struct {
	User    string `env:"DB_USER,required"`
	Pass    string `env:"DB_PASS,required"`
	Name    string `env:"DB_NAME,required"`
	Port    int    `env:"DB_PORT,required"`
	Host    string `env:"DB_HOST,required"`
	SSLMode string `env:"DB_SSLMODE,required"`
}

type StorageConfig struct {
	ContainerName       string `env:"STORAGE_CONTAINER_NAME,required"`
	AccountName         string `env:"STORAGE_ACCOUNT_NAME,required"`
	AccountKey          string `env:"STORAGE_ACCOUNT_KEY,required"`
	ConcurrencyLimit    int    `env:"STORAGE_CONCURRENCY_LIMIT,default=32"`
	UploadContainerName string `env:"UPLOAD_STORAGE_CONTAINER_NAME,required"`
}

type QueueConfig struct {
	QueueName   string `env:"STORAGE_QUEUE_NAME,required"`
	AccountName string `env:"STORAGE_ACCOUNT_NAME,required"`
	AccountKey  string `env:"STORAGE_ACCOUNT_KEY,required"`
	BatchSize   int    `env:"QUEUE_BATCH_SIZE,default=250"`
}

func (dbc DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s@%s:%s@%s:%d/%s",
		dbc.User,
		dbc.Host,
		dbc.Pass,
		dbc.Host,
		dbc.Port,
		dbc.Name,
	)
}

type MapConfig struct {
	MinTileZoom int `env:"MIN_TILE_ZOOM,default=2"`
	MaxTileZoom int `env:"MAX_TILE_ZOOM,default=20"`
}

type Config struct {
	HttpServer     HttpServerConfig
	HttpClient     HttpClientConfig
	Database       DatabaseConfig
	Storage        StorageConfig
	Queue          QueueConfig
	Strava         StravaAppConfig
	Map            MapConfig
	TemplatePath   string `env:"TEMPLATE_PATH,default=./templates"`
	StaticFileRoot string `env:"STATIC_FILE_ROOT,default=./static"`
}

func GetConfig(ctx context.Context) *Config {
	var config Config
	if err := envconfig.Process(ctx, &config); err != nil {
		log.Fatal(err)
	}
	return &config
}
