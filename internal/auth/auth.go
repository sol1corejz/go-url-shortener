// Пакет auth предоставляет функционал для работы с JWT (JSON Web Tokens).
// Он включает в себя создание и проверку токенов для аутентификации и авторизации пользователей.
package auth

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
)

// Claims структура, содержащая информацию о пользователе,
// которая будет закодирована в JWT токене.
type Claims struct {
	// Зарегистрированные стандартные поля JWT.
	jwt.RegisteredClaims
	// UserID - уникальный идентификатор пользователя.
	UserID string
}

// UserUUID хранит UUID пользователя, который будет использоваться в JWT токенах.
var UserUUID string

// TokenExp задает срок действия токена. В данном случае - 3 часа.
const TokenExp = time.Hour * 3

// SecretKey используется для подписи токенов. В реальной разработке следует использовать более безопасные ключи.
const SecretKey = "supersecretkey"

// GenerateToken генерирует новый JWT токен для пользователя.
// Возвращает строку с токеном и ошибку, если она возникла.
func GenerateToken() (string, error) {
	// Строим строку токена
	tokenString, err := BuildJWTString()
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// BuildJWTString создает строку JWT токена с уникальным идентификатором пользователя и сроком действия.
// Возвращает строку с токеном и ошибку, если она возникла.
func BuildJWTString() (string, error) {
	// Генерируем новый UUID для пользователя
	UserUUID = uuid.New().String()

	// Создаем новый токен с указанными претензиями (claims)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		UserID: UserUUID,
	})

	// Подписываем токен с использованием секретного ключа
	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// GetUserID извлекает UserID из переданного JWT токена.
// Возвращает строку с UserID, если токен валидный, или пустую строку в случае ошибки.
func GetUserID(tokenString string) string {
	claims := &Claims{}
	// Парсим токен и извлекаем claims
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			return []byte(SecretKey), nil
		})
	if err != nil {
		return ""
	}

	// Проверяем валидность токена
	if !token.Valid {
		logger.Log.Info("Token is not valid")
		return ""
	}

	// Возвращаем UserID из claims
	logger.Log.Info("Token is valid")
	return claims.UserID
}

// CheckIsAuthorized проверяет наличие и валидность JWT токена в куках запроса.
// Возвращает UserID, если пользователь авторизован, или ошибку, если токен отсутствует или недействителен.
func CheckIsAuthorized(r *http.Request) (string, error) {
	// Получаем токен из куки "token"
	cookie, err := r.Cookie("token")
	if err != nil {
		return "", err
	}

	// Извлекаем UserID из токена
	userID := GetUserID(cookie.Value)
	if userID == "" {
		return "", err
	}
	return userID, nil
}
