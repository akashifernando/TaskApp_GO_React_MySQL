package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"backend/internal/auth"
	"backend/pkg/utils"
)

func Auth(authService *auth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			utils.SendResponse(w, http.StatusUnauthorized, "Missing Authorization Header", nil)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		userID, err := authService.ValidateToken(tokenString)
		if err != nil {
			utils.SendResponse(w, http.StatusUnauthorized, "Invalid token", nil)
			return
		}

		r.Header.Set("X-User-ID", strconv.FormatInt(userID, 10))
		next(w, r)
	}
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}
