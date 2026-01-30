package jwt

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var refresh_key = []byte(os.Getenv("REFRESH_SECRET_KEY"))
var accsess_key = []byte(os.Getenv("ACCESS_SECRET_KEY"))

func GenerateRefreshToken(user_id int) (string, error) {
	payload := jwt.MapClaims{
		"user_id": user_id,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return token.SignedString(refresh_key)
}

func GenerateAccessToken(user_id int) (string, error) {
	payload := jwt.MapClaims{
		"user_id": user_id,
		"exp":     time.Now().Add(time.Minute * 15).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, payload)
	return token.SignedString(accsess_key)
}

func ValidateRefreshToken(token string) (jwt.MapClaims, error) {
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return refresh_key, nil
	})

	if err != nil {
		return nil, err
	}
	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims or token not valid")
}

func ValidateAccessToken(token string) (jwt.MapClaims, error) {

	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return accsess_key, nil
	})

	if err != nil {
		return nil, err
	}
	if !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return parsedToken.Claims.(jwt.MapClaims), nil

}
