package controller

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/abibby/validate/handler"
	"github.com/abibby/wishlist/auth"
	"github.com/abibby/wishlist/db"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type Purpose string

const (
	PurposeAuthorize = Purpose("authorize")
	PurposeRefresh   = Purpose("refresh")
	PurposeInvite    = Purpose("invite")
)

func WithPurpose(purpose Purpose) auth.TokenOptions {
	return auth.WithClaim("purpose", string(purpose))
}

func WithUser(u *db.User) auth.TokenOptions {
	return func(claims jwt.MapClaims) jwt.MapClaims {
		claims = auth.WithSubject(u.ID)(claims)
		claims = auth.WithClaim("username", u.Username)(claims)
		return claims
	}
}

func createUser(ctx context.Context, username string, passwordHash []byte, name string) (*db.User, error) {
	u := &db.User{}
	err := db.Tx(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(
			"INSERT INTO users (username,name,password) VALUES (?, ?, ?);",
			username,
			name,
			passwordHash,
		)
		if err != nil {
			return err
		}
		return tx.Get(u, "SELECT * FROM users ORDER BY id DESC LIMIT 1")
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

type CreateUserRequest struct {
	Username string `json:"username" validate:"required"`
	Name     string `json:"name"     validate:"required"`
	Password []byte `json:"password" validate:"required"`
	Request  *http.Request
}
type CreateUserResponse *db.User

var CreateUser = handler.Handler(func(r *CreateUserRequest) (any, error) {
	hash, err := bcrypt.GenerateFromPassword(r.Password, bcrypt.MinCost)
	if err != nil {
		return nil, err
	}

	u, err := createUser(r.Request.Context(), r.Username, hash, r.Name)
	return CreateUserResponse(u), err
})

type CreateUserPasswordlessRequest struct {
	Username string `json:"username" validate:"required"`
	Name     string `json:"name"     validate:"required"`
	Request  *http.Request
}
type CreateUserPasswordlessResponse struct {
	User    *db.User `json:"user"`
	Token   string   `json:"token"`
	Refresh string   `json:"refresh"`
}

var CreateUserPasswordless = handler.Handler(func(r *CreateUserPasswordlessRequest) (any, error) {
	u, err := createUser(r.Request.Context(), r.Username, []byte{}, r.Name)
	if err != nil {
		return nil, err
	}

	token, refresh, err := generateTokens(u)
	if err != nil {
		return nil, err
	}

	return &CreateUserPasswordlessResponse{
		User:    u,
		Token:   token,
		Refresh: refresh,
	}, nil
})

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password []byte `json:"password" validate:"required"`
	Request  *http.Request
}
type LoginResponse struct {
	Token   string `json:"token"`
	Refresh string `json:"refresh"`
}

var Login = handler.Handler(func(r *LoginRequest) (any, error) {
	u := &db.User{}

	err := db.Tx(r.Request.Context(), func(tx *sqlx.Tx) error {
		return tx.Get(u, "select * from users where username=?", r.Username)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return handler.ErrorResponse(fmt.Errorf("unauthorized"), http.StatusUnauthorized), nil
	} else if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword(u.Password, r.Password)
	if err != nil {
		return nil, err
	}

	token, refresh, err := generateTokens(u)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{
		Token:   token,
		Refresh: refresh,
	}, nil
})

type RefreshRequest struct {
	Request *http.Request
}

var Refresh = handler.Handler(func(r *RefreshRequest) (any, error) {
	claims, _ := auth.Claims(r.Request.Context())
	iPasswordless, _ := claims["passwordless"]
	passwordless, _ := iPasswordless.(bool)

	u := &db.User{}
	uid := userID(r.Request.Context())
	err := db.Tx(r.Request.Context(), func(tx *sqlx.Tx) error {
		return tx.Get(u, "select * from users where id=?", uid)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return handler.ErrorResponse(fmt.Errorf("unauthorized"), http.StatusUnauthorized), nil
	} else if err != nil {
		return nil, err
	}

	if passwordless && len(u.Password) > 0 {
		return handler.ErrorResponse(fmt.Errorf("unauthorized"), http.StatusUnauthorized), nil
	}

	token, refresh, err := generateTokens(u)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:   token,
		Refresh: refresh,
	}, nil
})

func generateTokens(u *db.User) (string, string, error) {
	passwordless := len(u.Password) == 0

	token, err := auth.GenerateToken(
		WithUser(u),
		auth.WithLifetime(24*time.Hour),
		WithPurpose(PurposeAuthorize),
	)
	if err != nil {
		return "", "", err
	}

	refresh, err := auth.GenerateToken(
		WithUser(u),
		claimsIf(!passwordless, auth.WithLifetime(30*24*time.Hour)),
		auth.WithClaim("passwordless", passwordless),
		WithPurpose(PurposeRefresh),
	)
	if err != nil {
		return "", "", err
	}
	return token, refresh, nil
}
func claimsIf(condition bool, modifyClaims ...auth.TokenOptions) auth.TokenOptions {
	return func(claims jwt.MapClaims) jwt.MapClaims {
		if condition {
			for _, m := range modifyClaims {
				claims = m(claims)
			}
		}
		return claims
	}
}
