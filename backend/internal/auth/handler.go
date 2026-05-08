package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"backend/internal/user"
	"backend/pkg/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Handler struct {
	db          *gorm.DB
	authService *Service
}

func NewHandler(db *gorm.DB, authService *Service) *Handler {
	return &Handler{db: db, authService: authService}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.SendResponse(w, http.StatusInternalServerError, "Failed to hash password", nil)
		return
	}

	u := user.AppUser{
		Username: req.Username,
		Password: string(hashedPassword),
		Role:     user.RoleUser,
	}

	if err := h.db.Create(&u).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			utils.SendResponse(w, http.StatusBadRequest, "Username already taken", nil)
			return
		}
		log.Println("Create user error:", err)
		utils.SendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	utils.SendResponse(w, http.StatusOK, "OK", u)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	var u user.AppUser
	if err := h.db.Where("username = ?", req.Username).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.SendResponse(w, http.StatusUnauthorized, "Invalid credentials", nil)
			return
		}
		log.Println("Select user error:", err)
		utils.SendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		utils.SendResponse(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	token, err := h.authService.GenerateToken(u.ID)
	if err != nil {
		utils.SendResponse(w, http.StatusInternalServerError, "Failed to generate token", nil)
		return
	}

	utils.SendResponse(w, http.StatusOK, "OK", token)
}
