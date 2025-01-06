package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 4
	minDigitCount     = 2
)

func HashPassword(password string) (string, error) {
	err := checkPassword(password)
	if err != nil {
		return "", err
	}
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPass), err
}

func CheckPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func checkPassword(password string) error {
	if len([]rune(password)) < minPasswordLength {
		return fmt.Errorf("password must be at least 4 characters")
	}
	digitCount := 0
	for _, char := range password {
		if unicode.IsDigit(char) {
			digitCount++
		}
	}
	if digitCount < minDigitCount {
		return fmt.Errorf("password must have at least 2 digits")
	}
	return nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "SNserver",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Time.Add(time.Now(), expiresIn)),
		Subject:   userID.String(),
	})
	signedToken, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unknown signing method: %v", token.Method.Alg())
		}
		return []byte(tokenSecret), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, fmt.Errorf("token is not valid")
	}

	subjectId, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, fmt.Errorf("subject is unknown")
	}

	userId, err := uuid.Parse(subjectId)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error in parsing id")
	}
	return userId, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	bearer := headers.Get("Authorization")
	if bearer == "" {
		return "", fmt.Errorf("authorization header does not exists")
	}
	token, ok := strings.CutPrefix(bearer, "Bearer ")
	if !ok {
		return "", fmt.Errorf("wrong authorization structure")
	}
	return token, nil
}

func MakeRefreshToken() (string, error) {
	randomData := make([]byte, 32)
	_, err := rand.Read(randomData)
	if err != nil {
		return "", fmt.Errorf("error in randomization: %s", err)
	}
	token := hex.EncodeToString(randomData)
	return token, nil
}
