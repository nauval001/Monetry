package helper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	ErrTokenInvalid = errors.New("token tidak valid")
	ErrTokenExpired = errors.New("token sudah kedaluwarsa")
)

// Claims adalah payload JWT Monetry.
type Claims struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTConfig adalah konfigurasi pembuatan token.
type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

// JWTManager menangani pembuatan & parsing JWT.
type JWTManager struct {
	cfg JWTConfig
}

// NewJWTManager membuat instance JWTManager.
func NewJWTManager(cfg JWTConfig) *JWTManager {
	return &JWTManager{cfg: cfg}
}

// AccessExpiry mengembalikan masa berlaku access token.
func (m *JWTManager) AccessExpiry() time.Duration { return m.cfg.AccessExpiry }

// GenerateAccessToken membuat access token berumur pendek.
func (m *JWTManager) GenerateAccessToken(userID int64, username, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Username:  username,
		Role:      role,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.cfg.AccessExpiry)),
			ID:        NewJTI(),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(m.cfg.Secret))
}

// GenerateRefreshToken membuat refresh token berumur panjang beserta JTI
// yang akan disimpan di database untuk keperluan rotasi & pencabutan token.
func (m *JWTManager) GenerateRefreshToken(userID int64, username, role string) (token string, jti string, expiresAt time.Time, err error) {
	now := time.Now()
	expiresAt = now.Add(m.cfg.RefreshExpiry)
	jti = NewJTI()

	claims := Claims{
		UserID:    userID,
		Username:  username,
		Role:      role,
		TokenType: TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti,
		},
	}

	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(m.cfg.Secret))
	return token, jti, expiresAt, err
}

// Parse memvalidasi signature & masa berlaku token lalu mengembalikan claims.
func (m *JWTManager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(m.cfg.Secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// ParseAccess memastikan token yang dikirim adalah access token.
func (m *JWTManager) ParseAccess(tokenString string) (*Claims, error) {
	claims, err := m.Parse(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeAccess {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// ParseRefresh memastikan token yang dikirim adalah refresh token.
func (m *JWTManager) ParseRefresh(tokenString string) (*Claims, error) {
	claims, err := m.Parse(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// HashToken menghasilkan SHA-256 dari token. Yang disimpan di database
// hanya hash-nya, bukan token asli.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// NewJTI membuat identifier unik untuk refresh token.
func NewJTI() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf)
}
