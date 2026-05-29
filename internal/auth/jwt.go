package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Subject    string   `json:"sub"`
	Kind       string   `json:"kind"`
	TenantID   string   `json:"tenantId,omitempty"`
	Role       string   `json:"role,omitempty"`
	Scopes     []string `json:"scopes,omitempty"`
	NodeAccess []string `json:"nodeAccess,omitempty"`
	jwt.RegisteredClaims
}

type IssueOptions struct {
	TenantID   string
	Role       string
	Scopes     []string
	NodeAccess []string
}

func Issue(secret, subject, kind string, ttl time.Duration) (string, error) {
	return IssueWithOptions(secret, subject, kind, ttl, IssueOptions{})
}

func IssueWithOptions(secret, subject, kind string, ttl time.Duration, opts IssueOptions) (string, error) {
	now := time.Now()
	claims := Claims{
		Subject:    subject,
		Kind:       kind,
		TenantID:   opts.TenantID,
		Role:       opts.Role,
		Scopes:     opts.Scopes,
		NodeAccess: opts.NodeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func Parse(secret, tokenString string) (Claims, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return claims, err
	}
	if !token.Valid {
		return claims, errors.New("invalid token")
	}
	return claims, nil
}
