package auth

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"time"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

const TOKEN_EXP = time.Hour * 3
const SECRET_KEY = "supersecretkey"

func GenerateToken() (string, error) {
	tokenString, err := BuildJWTString()
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func BuildJWTString() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},

		UserID: uuid.New().String(),
	})

	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GetUserId(tokenString string) string {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			return []byte(SECRET_KEY), nil
		})
	if err != nil {
		return ""
	}

	if !token.Valid {
		fmt.Println("Token is not valid")
		return ""
	}

	fmt.Println("Token is valid")
	return claims.UserID
}
