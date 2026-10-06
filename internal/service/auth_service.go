package service

import (
	"context"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/feedback/internal/config"
	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/repository"
)

type AuthService struct {
	users  *repository.UserRepo
	logins *repository.LoginAttemptRepo
	cfg    config.JWTConfig
	limit  config.LoginLimitConfig

	mu     sync.Mutex
	memBuckets map[string][]time.Time // identifier → failure timestamps
}

func NewAuthService(users *repository.UserRepo, logins *repository.LoginAttemptRepo,
	jwtCfg config.JWTConfig, limit config.LoginLimitConfig) *AuthService {
	return &AuthService{
		users:      users,
		logins:     logins,
		cfg:        jwtCfg,
		limit:      limit,
		memBuckets: make(map[string][]time.Time),
	}
}

type LoginResult struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      *model.User `json:"user"`
}

type Claims struct {
	UserID   uint64 `json:"uid"`
	Username string `json:"usr"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func (s *AuthService) Login(ctx context.Context, username, password, ip string) (*LoginResult, error) {
	identifier := username + "@" + ip
	if s.isLockedOut(identifier) {
		return nil, apperr.RateLimited("too many failed attempts, please try later")
	}

	u, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return nil, apperr.Internal("query user", err)
	}
	if u == nil || u.Status != model.UserStatusEnabled {
		s.recordFailure(ctx, identifier)
		return nil, apperr.Unauthorized("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		s.recordFailure(ctx, identifier)
		return nil, apperr.Unauthorized("invalid credentials")
	}

	_ = s.logins.Record(ctx, identifier, true)
	s.clearFailures(identifier)
	_ = s.users.UpdateLastLogin(ctx, u.ID, time.Now())

	exp := time.Now().Add(s.cfg.Expire)
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, apperr.Internal("sign token", err)
	}
	return &LoginResult{Token: signed, ExpiresAt: exp, User: u}, nil
}

func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperr.Unauthorized("invalid signing method")
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil || !tok.Valid {
		return nil, apperr.Unauthorized("invalid token")
	}
	return claims, nil
}

func (s *AuthService) recordFailure(ctx context.Context, identifier string) {
	_ = s.logins.Record(ctx, identifier, false)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-s.limit.Window)
	bucket := s.memBuckets[identifier]
	out := bucket[:0]
	for _, t := range bucket {
		if t.After(cutoff) {
			out = append(out, t)
		}
	}
	out = append(out, now)
	s.memBuckets[identifier] = out
}

func (s *AuthService) clearFailures(identifier string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.memBuckets, identifier)
}

func (s *AuthService) isLockedOut(identifier string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-s.limit.Window)
	bucket := s.memBuckets[identifier]
	n := 0
	for _, t := range bucket {
		if t.After(cutoff) {
			n++
		}
	}
	return n >= s.limit.MaxAttempts
}

// HashPassword wraps bcrypt for callers (e.g. seed, admin create).
func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}
