package config

import (
	"ngrok/pkg/server/db"
	"os"
	"strconv"

	"gorm.io/gorm"
	klog "k8s.io/klog/v2"
)

type Config struct {
	TLSCert           string
	TLSKey            string
	LogLevel          string
	HttpAddr          string
	HttpsAddr         string
	TunnelAddr        string
	AdminAddr         string
	HealthAddr        string
	Domain            string
	ProxyMaxPoolSize  int
	SecretKey         string
	ConnectionTimeout int
	Database          *gorm.DB
}

func InitConfig() *Config {

	dbConf := db.Database{
		Type:     getEnvStr("DATABASE_TYPE", "sqlite"),
		File:     getEnvStr("DATABASE_FILE", "sqlite.db"),
		Host:     getEnvStr("DATABASE_HOST", "sqlite"),
		Port:     getEnvInt("DATABASE_PORT", 3306),
		User:     getEnvStr("DATABASE_USER", "sqlite"),
		Password: getEnvStr("DATABASE_PASSWORD", "sqlite"),
	}

	dbConn, err := db.GetDB(&dbConf)
	if err != nil {
		klog.Fatalf("Could not connect to database %w", err)
	}
	if dbConn.Error != nil {
		klog.Fatalf("Could not connect to database v2 %w", err)
	}

	err = db.AutoMigrate(dbConn)
	if err != nil {
		klog.Fatalf("Could not migrate database %w", err)
	}
	klog.Info("LGTM")
	config := Config{
		TLSCert:           getEnvStr("TLS_CERT_PATH", "/"),
		TLSKey:            getEnvStr("TLS_CERT_PATH", "/"),
		LogLevel:          getEnvStr("LOG_LEVEL", "DEBUG"), // DEBUG,INFO,WARNING,ERROR
		HttpAddr:          getEnvStr("HTTP_LISTEN_ADDR", ":80"),
		HttpsAddr:         getEnvStr("HTTPS_LISTEN_ADDR", ":443"),
		TunnelAddr:        getEnvStr("TUNNEL_LISTEN_ADDR", ":4443"),
		AdminAddr:         getEnvStr("ADMIN_ADDR", ":4111"),
		HealthAddr:        getEnvStr("HTTP_ADDR", ":4112"),
		Domain:            getEnvStr("HTTP_ADDR", ":80"),
		ProxyMaxPoolSize:  getEnvInt("PROXY_MAX_POOL_SIZE", 10),
		SecretKey:         getEnvStr("SECRET_KEY", "supersecretkey"),
		ConnectionTimeout: getEnvInt("CONNECTION_TIMEOUT_SECONDS", 10),
		Database:          dbConn,
	}

	return &config
}

func getEnvStr(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		i64, err := strconv.ParseInt(value, 10, 0)
		var ret int = int(i64) // Safe based on ParseInt doc (0 bitsize = int)
		if err != nil {
			return fallback
		}
		return ret
	}
	return fallback
}
