package auth

import (
	"context"
	"errors"
	"fmt"
	"ngrok/pkg/log"
	"ngrok/pkg/server/db"
	"ngrok/pkg/util"

	"gorm.io/gorm"
)

const (
	apiKeySize = 32
)

func CreateAuthToken(ctx context.Context, dbConn *gorm.DB, desc string) error {
	authToken, err := util.SecureRandId(apiKeySize)
	if err != nil {
		return err
	}

	accessKey := db.AuthToken{
		AuthToken:   authToken,
		Description: desc,
	}
	if err := dbConn.WithContext(ctx).Create(&accessKey).Error; err != nil {
		log.Error("CreateAuthToken: Failed to insert token: %v", err)
		return fmt.Errorf("CreateAuthToken: could not insert token: %w", err)
	}
	return nil
}

func ListAuthTokens(ctx context.Context, dbConn *gorm.DB, offset int) ([]db.AuthToken, error) {
	if offset < 0 {
		return nil, errors.New("invalid offset: must be non-negative")
	}

	var accessKeys []db.AuthToken
	result := dbConn.WithContext(ctx).Find(&accessKeys).Limit(5).Offset(offset).Order("created_at DESC")
	if result.Error != nil {
		log.Error("ListAuthTokens: Failed to list tokens: %v", result.Error)
		return nil, fmt.Errorf("ListAuthTokens: could not list tokens: %w", result.Error)
	}
	return accessKeys, nil
}

func GetAuthToken(ctx context.Context, dbConn *gorm.DB, authToken string) (db.AuthToken, error) {
	if authToken == "" {
		return db.AuthToken{}, errors.New("GetAuthToken: authToken cannot be empty")
	}

	var accessKey db.AuthToken
	result := dbConn.WithContext(ctx).Where("auth_token = ?", authToken).First(&accessKey).Order("created_at DESC")
	if result.Error != nil {
		log.Error("GetAuthToken: Failed to insert token: %v", result.Error)
		return db.AuthToken{}, fmt.Errorf("GetAuthToken: could not insert token: %w", result.Error)
	}

	return accessKey, nil
}

func DeleteAuthToken(ctx context.Context, dbConn *gorm.DB, id string) error {

	if id == "" {
		return errors.New("DeleteAuthToken: key cannot be empty")
	}

	result := dbConn.WithContext(ctx).Delete(&db.AuthToken{}, id)

	if result.Error != nil {
		log.Error("DeleteAuthToken: Failed to delete token: %v", result.Error)
		return fmt.Errorf("DeleteAuthToken: could not delete token: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("DeleteAuthToken: no token deleted for id: %s", id)
	}
	log.Info("DeleteAuthToken: Successfully deleted token id %s", id)

	return nil
}

func ValidateAuthToken(ctx context.Context, db *gorm.DB, token string) error {
	found, err := GetAuthToken(ctx, db, token)
	if err != nil {
		return fmt.Errorf("ValidateAuthToken: provided token key is invalid")
	}
	if found.AuthToken == "" {
		return fmt.Errorf("ValidateAuthToken: token was not provided")
	}
	return nil
}
