package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	jwtSecret []byte
}

func NewService(secret string) *Service {
	return &Service{jwtSecret: []byte(secret)}
}

func (s *Service) GenerateToken(userID int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": userID,
		"exp":    time.Now().Add(time.Hour * 24).Unix(), // 24 hours
	})
	return token.SignedString(s.jwtSecret)
}

func (s *Service) ValidateToken(tokenString string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, jwt.ErrSignatureInvalid
	}

	userIDFloat, ok := claims["userId"].(float64)
	if !ok {
		return 0, jwt.ErrSignatureInvalid
	}

	return int64(userIDFloat), nil
}
