package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Manager struct {
	secret []byte
	ttl    time.Duration
}

type claims struct {
	Subject   string
	ExpiresAt int64
	IssuedAt  int64
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

func (m *Manager) Issue(userID string) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	now := time.Now().UTC()
	payload, err := json.Marshal(map[string]any{
		"sub": userID,
		"iat": now.Unix(),
		"exp": now.Add(m.ttl).Unix(),
	})
	if err != nil {
		return "", err
	}
	unsigned := encode(header) + "." + encode(payload)
	return unsigned + "." + m.sign(unsigned), nil
}

func (m *Manager) Parse(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || !hmac.Equal([]byte(parts[2]), []byte(m.sign(parts[0]+"."+parts[1]))) {
		return "", errors.New("token invalido")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("token invalido")
	}
	var parsed map[string]any
	if json.Unmarshal(payload, &parsed) != nil {
		return "", errors.New("token invalido")
	}
	subject, okSubject := parsed["sub"].(string)
	expiresAt, okExpires := parsed["exp"].(float64)
	if !okSubject || !okExpires || subject == "" || time.Now().Unix() >= int64(expiresAt) {
		return "", errors.New("token expirado ou invalido")
	}
	return subject, nil
}

func (m *Manager) sign(value string) string {
	h := hmac.New(sha256.New, m.secret)
	_, _ = h.Write([]byte(value))
	return encode(h.Sum(nil))
}

func encode(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}
