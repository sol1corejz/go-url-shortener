package auth

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"time"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

var UserUUID string

const TokenExp = time.Hour * 3
const SecretKey = "supersecretkey"

func GenerateToken() (string, error) {
	tokenString, err := BuildJWTString()
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func BuildJWTString() (string, error) {

	UserUUID = uuid.New().String()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},

		UserID: UserUUID,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GetUserID(tokenString string) string {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			return []byte(SecretKey), nil
		})
	if err != nil {
		return ""
	}

	if !token.Valid {
		logger.Log.Info("Token is not valid")
		return ""
	}

	logger.Log.Info("Token is valid")
	return claims.UserID
}
