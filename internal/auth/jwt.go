package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/statusy/statusy/config"
	"github.com/statusy/statusy/internal/models"
	"gorm.io/gorm"
)

type Claims struct {
	UserID   uint             `json:"uid"`
	Username string           `json:"sub"`
	Role     models.UserRole  `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// IssueTokenPair creates a new access + refresh token pair for a user.
func IssueTokenPair(user *models.User, cfg *config.Config) (*TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(time.Duration(cfg.Auth.JWTExpiryHours) * time.Hour)
	refreshExp := now.Add(time.Duration(cfg.Auth.RefreshExpiryHours) * time.Hour)

	accessClaims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
			Issuer:    "statusy",
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(cfg.Auth.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	refreshClaims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			Issuer:    "statusy-refresh",
		},
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString([]byte(cfg.Auth.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("signing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExp.Unix(),
	}, nil
}

// ParseToken validates a JWT and returns its claims.
func ParseToken(tokenStr string, cfg *config.Config) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cfg.Auth.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// BlacklistToken adds a JWT ID to the blacklist table.
func BlacklistToken(db *gorm.DB, jti string, expiresAt time.Time) error {
	return db.Create(&models.JWTBlacklist{
		JTI:       jti,
		ExpiresAt: expiresAt,
	}).Error
}

// IsBlacklisted checks if a JTI has been revoked.
func IsBlacklisted(db *gorm.DB, jti string) (bool, error) {
	var count int64
	err := db.Model(&models.JWTBlacklist{}).
		Where("jti = ? AND expires_at > ?", jti, time.Now()).
		Count(&count).Error
	return count > 0, err
}

// PurgeExpiredTokens removes expired blacklist entries (call periodically).
func PurgeExpiredTokens(db *gorm.DB) error {
	return db.Where("expires_at <= ?", time.Now()).Delete(&models.JWTBlacklist{}).Error
}
