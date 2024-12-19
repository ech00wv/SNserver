package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/ech00wv/SNserver/internal/auth"
	"github.com/ech00wv/SNserver/internal/database"
	"github.com/ech00wv/SNserver/internal/models"
)

type UserService struct {
	Queries *database.Queries
}

func (userServ *UserService) CreateUser(ctx context.Context, requestedUser models.UserRequest) (database.User, int, error) {

	hashedPassword, err := auth.HashPassword(requestedUser.Password)
	if err != nil {
		return database.User{}, http.StatusInternalServerError, fmt.Errorf("cannot hash password")
	}

	if !validateEmail(requestedUser.Email) {
		return database.User{}, http.StatusBadRequest, fmt.Errorf("email is not valid")
	}

	dbUser, err := userServ.Queries.CreateUser(ctx, database.CreateUserParams{Email: requestedUser.Email, HashedPassword: hashedPassword})
	if err != nil {
		return database.User{}, http.StatusInternalServerError, fmt.Errorf("error creating user")
	}

	return dbUser, http.StatusCreated, nil
}

func (userServ *UserService) DeleteUsers(ctx context.Context) error {
	err := userServ.Queries.DeleteUsers(ctx)
	return err
}

func (userServ *UserService) LoginUser(ctx context.Context, requestedUser models.UserRequest) (database.User, int, error) {
	if !validateEmail(requestedUser.Email) {
		return database.User{}, http.StatusBadRequest, fmt.Errorf("email is not valid")
	}

	dbUser, err := userServ.Queries.GetUserByEmail(ctx, requestedUser.Email)
	if err != nil {
		return database.User{}, http.StatusInternalServerError, fmt.Errorf("cannot get user")
	}

	err = auth.CheckPasswordHash(requestedUser.Password, dbUser.HashedPassword)
	if err != nil {
		return database.User{}, http.StatusUnauthorized, fmt.Errorf("incorrect email or password")
	}

	return dbUser, http.StatusOK, nil
}

func validateEmail(email string) bool {
	matched, _ := regexp.Match(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`, []byte(email))
	return matched
}
