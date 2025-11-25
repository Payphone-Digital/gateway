package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/surdiana/gateway/internal/dto"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type JWTService struct {
	secretKey string
}

func NewJWTService(secretKey string) *JWTService {
	return &JWTService{
		secretKey: secretKey,
	}
}

// GenerateToken creates a new JWT token for the user (short-lived)
func (s *JWTService) GenerateToken(user *dto.UserResponse, tokenVersion int) (string, error) {
	// Create claims with short expiry (15 minutes)
	claims := jwt.MapClaims{
		"user_id":       user.ID,
		"email":         user.Email,
		"first_name":    user.FirstName,
		"last_name":     user.LastName,
		"token_version": tokenVersion,
		"exp":           time.Now().Add(15 * time.Minute).Unix(), // 15 minutes
		"iat":           time.Now().Unix(),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GenerateRefreshToken creates a secure refresh token
func (s *JWTService) GenerateRefreshToken() (string, error) {
	// Generate 32-byte random token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashRefreshToken securely hashes a refresh token
func (s *JWTService) HashRefreshToken(refreshToken string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(refreshToken), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash refresh token: %w", err)
	}
	return string(hash), nil
}

// VerifyRefreshToken verifies a refresh token against its hash
func (s *JWTService) VerifyRefreshToken(refreshToken, hashedToken string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedToken), []byte(refreshToken))
	return err == nil
}

// GetSecretKey returns the secret key (for repository use)
func (s *JWTService) GetSecretKey() string {
	return s.secretKey
}

// ValidateToken validates the JWT token and returns the claims
func (s *JWTService) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	// Validate token and extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateTokenWithVersion validates JWT token and returns claims with token version check
func (s *JWTService) ValidateTokenWithVersion(tokenString string, expectedVersion int) (*jwt.MapClaims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Check token version
	if tokenVersion, ok := (*claims)["token_version"]; ok {
		if versionFloat, ok := tokenVersion.(float64); ok {
			tokenVersionInt := int(versionFloat)
			if tokenVersionInt != expectedVersion {
				return nil, errors.New("token version mismatch")
			}
		} else {
			return nil, errors.New("invalid token version format")
		}
	} else {
		// For backward compatibility with tokens without version
		return nil, errors.New("token version missing")
	}

	return claims, nil
}

// RefreshToken generates a new token with extended expiration (deprecated - use LoginUser with refresh token)
func (s *JWTService) RefreshToken(oldTokenString string) (string, error) {
	// This method is deprecated, should use refresh token flow instead
	// Keeping for backward compatibility but should not be used
	return "", errors.New("refresh token flow should be used instead")
}