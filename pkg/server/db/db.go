package db

import (
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

type AuthToken struct {
	ID          string `gorm:"primaryKey;size:36"`
	AuthToken   string `gorm:"unique;not null;size:64"`
	Description string `gorm:"not null"`
	gorm.Model
}

type Database struct {
	Type     string `json:"type"`
	File     string `json:"file,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&AuthToken{})
}

func (a *AuthToken) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == "" {
		a.ID = uuid.NewString() // Generates a UUIDv4 string
	}
	return
}

func GetDB(db *Database) (*gorm.DB, error) {
	switch db.Type {
	case "sqlite":
		return gorm.Open(sqlite.Open(db.File), &gorm.Config{})
	case "postgres": //TODO: implement env vars to take over
		dsn := "host=localhost user=gorm dbname=gorm password=gorm sslmode=disable"
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	case "mysql": //TODO: implement env vars to take over
		dsn := "gorm:gorm@tcp(127.0.0.1:3306)/gorm?charset=utf8mb4&parseTime=True&loc=Local"
		return gorm.Open(mysql.Open(dsn), &gorm.Config{})
	default:
		return nil, gorm.ErrInvalidDB
	}
}
