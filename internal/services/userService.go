package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/ech00wv/SNserver/internal/auth"
	"github.com/ech00wv/SNserver/internal/config"
	"github.com/ech00wv/SNserver/internal/database"
	"github.com/ech00wv/SNserver/internal/models"
)

type UserService struct {
	Queries *database.Queries
}

func (userServ *UserService) CreateUser(ctx context.Context, requestedUser models.UserRequest) (models.User, int, error) {

	hashedPassword, err := auth.HashPassword(requestedUser.Password)
	if err != nil {
		return models.User{}, http.StatusInternalServerError, fmt.Errorf("cannot hash password: %s", err)
	}

	if !validateEmail(requestedUser.Email) {
		return models.User{}, http.StatusBadRequest, fmt.Errorf("email is not valid")
	}

	dbUser, err := userServ.Queries.CreateUser(ctx, database.CreateUserParams{Email: requestedUser.Email, HashedPassword: hashedPassword})
	if err != nil {
		return models.User{}, http.StatusInternalServerError, fmt.Errorf("error creating user: %s", err)
	}
	responseUser := convertDBToUser(dbUser)
	return responseUser, http.StatusCreated, nil
}

func (userServ *UserService) DeleteUsers(ctx context.Context) error {
	err := userServ.Queries.DeleteUsers(ctx)
	return err
}

func (userServ *UserService) LoginUser(ctx context.Context, requestedUser models.UserRequest, apiCfg *config.ApiConfig) (models.User, int, error) {
	if !validateEmail(requestedUser.Email) {
		return models.User{}, http.StatusBadRequest, fmt.Errorf("email is not valid")
	}

	dbUser, err := userServ.Queries.GetUserByEmail(ctx, requestedUser.Email)
	if err != nil {
		return models.User{}, http.StatusInternalServerError, fmt.Errorf("cannot get user: %s", err)
	}

	err = auth.CheckPasswordHash(requestedUser.Password, dbUser.HashedPassword)
	if err != nil {
		return models.User{}, http.StatusUnauthorized, fmt.Errorf("incorrect email or password: %s", err)
	}
	expiresIn := time.Hour

	token, err := auth.MakeJWT(dbUser.ID, apiCfg.JWTSecret, expiresIn)
	if err != nil {
		return models.User{}, http.StatusInternalServerError, fmt.Errorf("error in token creation: %s", err)
	}

	responseUser := convertDBToUser(dbUser)
	responseUser.Token = token
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		return models.User{}, http.StatusInternalServerError, fmt.Errorf("cannot generate refresh token: %s", err)
	}

	responseUser.RefreshToken = refreshToken
	err = userServ.Queries.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{Token: refreshToken, UserID: dbUser.ID, ExpiresAt: time.Now().Add(1440 * time.Hour)})
	if err != nil {
		return models.User{}, http.StatusInternalServerError, fmt.Errorf("cannot create a refresh token entry: %s", err)
	}

	return responseUser, http.StatusOK, nil
}

func (userServ *UserService) UpdateUser(ctx context.Context, header http.Header, apiCfg *config.ApiConfig, email, password string) (database.User, int, error) {
	token, err := auth.GetBearerToken(header)
	if err != nil {
		return database.User{}, http.StatusUnauthorized, fmt.Errorf("wrong authorization header: %s", err)
	}

	userID, err := auth.ValidateJWT(token, apiCfg.JWTSecret)
	if err != nil {
		return database.User{}, http.StatusUnauthorized, fmt.Errorf("unknown JWT: %s", err)
	}

	if !validateEmail(email) {
		return database.User{}, http.StatusBadRequest, fmt.Errorf("wrong email structure: %s", err)
	}

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return database.User{}, http.StatusInternalServerError, fmt.Errorf("cannot hash password: %s", err)
	}

	dbUser, err := userServ.Queries.UpdateUser(ctx, database.UpdateUserParams{ID: userID, Email: email, HashedPassword: hashedPassword})
	if err != nil {
		return database.User{}, http.StatusInternalServerError, fmt.Errorf("cannot update user: %s", err)
	}

	return dbUser, http.StatusOK, nil

}

func validateEmail(email string) bool {
	matched, _ := regexp.Match(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`, []byte(email))
	return matched
}

func convertDBToUser(dbUser database.User) models.User {
	return models.User{
		ID:             dbUser.ID,
		CreatedAt:      dbUser.CreatedAt,
		UpdatedAt:      dbUser.UpdatedAt,
		Email:          dbUser.Email,
		HashedPassword: dbUser.HashedPassword,
	}
}
