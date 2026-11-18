package model

import "time"

// RefreshToken adalah entitas pada tabel refresh_tokens yang dipakai untuk
// rotasi & pencabutan refresh token.
type RefreshToken struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	JTI       string     `json:"jti"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// IsRevoked menandakan token sudah dicabut (logout / rotasi).
func (t *RefreshToken) IsRevoked() bool { return t.RevokedAt != nil }

// IsExpired menandakan masa berlaku token sudah habis.
func (t *RefreshToken) IsExpired() bool { return time.Now().UTC().After(t.ExpiresAt) }

// IsUsable menandakan token masih boleh dipakai untuk meminta token baru.
func (t *RefreshToken) IsUsable() bool { return !t.IsRevoked() && !t.IsExpired() }
