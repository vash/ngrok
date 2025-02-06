package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"ngrok/pkg/log"
	"ngrok/pkg/server/db"
	"ngrok/pkg/util"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var secretKey = []byte("supersecretkey")

const (
	apiKeySize = 32
)

type AccessKey struct {
	ID          string    `json:"id"`
	AuthToken   string    `json:"auth_token"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// createAPIKey generates a new API key and inserts it into the database.
func CreateAPIKey(ctx context.Context, desc string) error {
	db, err := db.GetConnection()
	if err != nil {
		return err
	}
	id := uuid.New().String()

	authToken, err := util.SecureRandId(apiKeySize)
	if err != nil {
		return err
	}
	log.Info("authToken length %d", len(authToken))

	stmt, err := db.PrepareContext(ctx, `
		INSERT INTO apikeys (id, auth_token, description, created_at)
		VALUES ($1, $2, $3, $4)
	`)
	if err != nil {
		log.Error("Failed to prepare SQL statement: %v", err)
		return fmt.Errorf("failed to prepare SQL statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, id, authToken, desc, time.Now().UTC())
	if err != nil {
		log.Error("Failed to insert API key: %v", err)
		return fmt.Errorf("could not insert API key: %w", err)
	}
	return nil
}

func ListAPIKeys(ctx context.Context, offset int) ([]AccessKey, error) {
	if offset < 0 {
		return nil, errors.New("invalid offset: must be non-negative")
	}
	db, err := db.GetConnection()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, auth_token, description, created_at FROM apikeys ORDER BY created_at DESC LIMIT 5 OFFSET ?`

	rows, err := db.QueryContext(ctx, query, offset)
	if err != nil {
		log.Error("ListAPIKeys: query error: %v", err)
		return nil, err
	}
	defer rows.Close()

	var keys []AccessKey
	for rows.Next() {
		var key AccessKey
		if err := rows.Scan(&key.ID, &key.AuthToken, &key.Description, &key.CreatedAt); err != nil {
			log.Error("ListAPIKeys: scan error: %v", err)
			return nil, err
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		log.Error("ListAPIKeys: row iteration error: %v", err)
		return nil, err
	}

	return keys, nil
}

// GetAPIKey retrieves an API key by its value.
func GetAPIKey(ctx context.Context, authToken string) (AccessKey, error) {
	if authToken == "" {
		return AccessKey{}, errors.New("GetAPIKey: authToken cannot be empty")
	}
	db, err := db.GetConnection()
	if err != nil {
		return AccessKey{}, err
	}

	query := `SELECT id, auth_token, description, created_at FROM apikeys WHERE auth_token = ?`
	var recoveredAPIKey AccessKey

	err = db.QueryRowContext(ctx, query, authToken).Scan(
		&recoveredAPIKey.ID,
		&recoveredAPIKey.AuthToken,
		&recoveredAPIKey.Description,
		&recoveredAPIKey.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AccessKey{}, fmt.Errorf("GetAPIKey: key not found")
		}
		log.Error("GetAPIKey: query error: %v", err)
		return AccessKey{}, err
	}

	return recoveredAPIKey, nil
}

func DeleteAPIKey(ctx context.Context, keyId string) error {

	db, err := db.GetConnection()
	if err != nil {
		return err
	}
	if keyId == "" {
		return errors.New("DeleteAPIKey: key cannot be empty")
	}

	query := `DELETE FROM apikeys WHERE id = ?`

	result, err := db.ExecContext(ctx, query, keyId)
	if err != nil {
		log.Error("DeleteAPIKey: query error: %v", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("DeleteAPIKey: failed to get affected rows: %v", err)
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("DeleteAPIKey: no API key deleted for keyId: %s", keyId)
	}
	log.Info("Successfully deleted keyId %s", keyId)

	return nil
}

func ValidateAPIKey(ctx context.Context, token string) error {
	found, err := GetAPIKey(ctx, token)
	if err != nil {
		return fmt.Errorf("provided token key is invalid")
	}
	if found.AuthToken == "" {
		return fmt.Errorf("auth token was not provided")
	}
	return nil
}
