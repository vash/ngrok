package config

import (
	"ngrok/pkg/server/db"
	"os"
	"strconv"

	"gorm.io/gorm"
	klog "k8s.io/klog/v2"
)

type Config struct {
	RegistryCacheFile string
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
	ConnectionTimeout int
	Database          *gorm.DB
}

func InitConfig() *Config {

	dbConf := db.Database{
		Type:     getEnvStr("DATABASE_TYPE", "sqlite"), // sqlite/postgres/mysql
		File:     getEnvStr("DATABASE_FILE", "sqlite.db"),
		Host:     getEnvStr("DATABASE_HOST", "localhost"),
		Port:     getEnvInt("DATABASE_PORT", 5432),
		User:     getEnvStr("DATABASE_USER", "postgres"),
		Password: getEnvStr("DATABASE_PASSWORD", "supersecretpassw0rd"),
	}

	dbConn, err := db.GetDB(&dbConf)
	if err != nil {
		klog.Fatalf("Could not connect to database %v", err)
	}
	if dbConn.Error != nil {
		klog.Fatalf("Could not connect to database v2 %v", err)
	}

	err = db.AutoMigrate(dbConn)
	if err != nil {
		klog.Fatalf("Could not migrate database %v", err)
	}
	config := Config{
		RegistryCacheFile: getEnvStr("REGISTRY_CACHE_FILE", ""),
		TLSCert:           getEnvStr("TLS_CERT_PATH", "./certs/tls.crt"),
		TLSKey:            getEnvStr("TLS_KEY_PATH", "./certs/tls.key"),
		LogLevel:          getEnvStr("LOG_LEVEL", "DEBUG"), // DEBUG,INFO,WARNING,ERROR
		HttpAddr:          getEnvStr("HTTP_LISTEN_ADDR", ":80"),
		HttpsAddr:         getEnvStr("HTTPS_LISTEN_ADDR", ":443"),
		TunnelAddr:        getEnvStr("TUNNEL_LISTEN_ADDR", ":4443"),
		AdminAddr:         getEnvStr("ADMIN_ADDR", ":4111"),
		HealthAddr:        getEnvStr("HTTP_ADDR", ":4112"),
		Domain:            getEnvStr("DOMAIN", "ngrok.me"),
		ProxyMaxPoolSize:  getEnvInt("PROXY_MAX_POOL_SIZE", 10),
		ConnectionTimeout: getEnvInt("CONNECTION_TIMEOUT_SECONDS", 10),
		Database:          dbConn,
	}
	klog.Infof("CONFIG IS %+v", config)

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
