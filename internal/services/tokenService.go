package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ech00wv/SNserver/internal/auth"
	"github.com/ech00wv/SNserver/internal/config"
	"github.com/ech00wv/SNserver/internal/database"
)

type TokenService struct {
	Queries database.Queries
}

func (tokenServ *TokenService) RefreshAccessToken(ctx context.Context, header http.Header, apiCfg *config.ApiConfig) (string, int, error) {

	refreshToken, err := auth.GetBearerToken(header)
	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf("authorization header has wrong structure: %s", err)
	}

	dbUserId, err := tokenServ.Queries.GetUserFromRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", http.StatusUnauthorized, fmt.Errorf("could not find refresh token or it is expired: %s", err)
	}

	newAccessToken, err := auth.MakeJWT(dbUserId, apiCfg.JWTSecret, time.Hour)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("error in creating jwt: %s", err)
	}

	return newAccessToken, http.StatusOK, nil

}

func (tokenServ *TokenService) RevokeRefreshToken(ctx context.Context, header http.Header) (int, error) {
	refreshToken, err := auth.GetBearerToken(header)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("authorization header has wrong structure: %s", err)
	}
	err = tokenServ.Queries.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot revoke token: %s", err)
	}
	return http.StatusNoContent, nil
}
