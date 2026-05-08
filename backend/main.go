package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var jwtSecret []byte

// Models
type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

type AppUser struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Role      Role      `json:"role"`
}

type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	Subject     string    `json:"category"`
	DueDate     string    `json:"dueDate"`
	UserID      int64     `json:"-"`
}

// DTOs
type UserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TaskRequest struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	Subject     string `json:"category"`
	DueDate     string `json:"dueDate"`
}

type Response struct {
	Success    bool        `json:"success,omitempty"`
	StatusCode int         `json:"statusCode"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
}

// Helpers
func sendJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func sendResponse(w http.ResponseWriter, status int, msg string, data interface{}) {
	sendJSON(w, status, Response{
		StatusCode: status,
		Message:    msg,
		Data:       data,
	})
}

// generateToken creates a JWT token
func generateToken(userID int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": userID,
		"exp":    time.Now().Add(time.Hour * 24).Unix(), // 24 hours
	})
	return token.SignedString(jwtSecret)
}

// Auth Middleware to extract user ID
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			sendResponse(w, http.StatusUnauthorized, "Missing Authorization Header", nil)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			sendResponse(w, http.StatusUnauthorized, "Invalid token", nil)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			sendResponse(w, http.StatusUnauthorized, "Invalid token claims", nil)
			return
		}

		userIDFloat, ok := claims["userId"].(float64)
		if !ok {
			sendResponse(w, http.StatusUnauthorized, "Invalid user ID in token", nil)
			return
		}

		// Set userId in request header for the next handler
		r.Header.Set("X-User-ID", strconv.FormatInt(int64(userIDFloat), 10))
		next(w, r)
	}
}

// Handlers
func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, "Failed to hash password", nil)
		return
	}

	now := time.Now()
	res, err := db.Exec("INSERT INTO users (username, password, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		req.Username, hashedPassword, RoleUser, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			sendResponse(w, http.StatusBadRequest, "Username already taken", nil)
			return
		}
		log.Println("Insert user error:", err)
		sendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	id, _ := res.LastInsertId()
	
	user := AppUser{
		ID:       id,
		Username: req.Username,
		Role:     RoleUser,
	}

	sendResponse(w, http.StatusOK, "OK", user)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	var user AppUser
	var hashedPassword string
	
	err := db.QueryRow("SELECT id, username, password, role FROM users WHERE username = ?", req.Username).
		Scan(&user.ID, &user.Username, &hashedPassword, &user.Role)
	
	if err != nil {
		if err == sql.ErrNoRows {
			sendResponse(w, http.StatusUnauthorized, "Invalid credentials", nil)
			return
		}
		log.Println("Select user error:", err)
		sendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		sendResponse(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	token, err := generateToken(user.ID)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, "Failed to generate token", nil)
		return
	}

	sendResponse(w, http.StatusOK, "OK", token)
}

func getUserIdFromRequest(r *http.Request) int64 {
	idStr := r.Header.Get("X-User-ID")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	return id
}

func createTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	userID := getUserIdFromRequest(r)
	now := time.Now()

	var dueDate interface{}
	if req.DueDate != "" {
		dueDate = req.DueDate
	} else {
		dueDate = nil
	}

	res, err := db.Exec(`INSERT INTO tasks 
		(title, description, completed, subject, due_date, created_at, updated_at, user_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Title, req.Description, req.Completed, req.Subject, dueDate, now, now, userID)
		
	if err != nil {
		log.Println("Insert task error:", err)
		sendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	id, _ := res.LastInsertId()
	req.ID = id
	sendResponse(w, http.StatusOK, "OK", req)
}

func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	userID := getUserIdFromRequest(r)
	now := time.Now()
	
	var dueDate interface{}
	if req.DueDate != "" {
		dueDate = req.DueDate
	} else {
		dueDate = nil
	}

	result, err := db.Exec(`UPDATE tasks 
		SET title = ?, description = ?, completed = ?, subject = ?, due_date = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`,
		req.Title, req.Description, req.Completed, req.Subject, dueDate, now, req.ID, userID)
		
	if err != nil {
		log.Println("Update task error:", err)
		sendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		sendResponse(w, http.StatusNotFound, "Task not found or not yours", nil)
		return
	}

	sendResponse(w, http.StatusOK, "OK", req)
}

func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIdFromRequest(r)

	rows, err := db.Query(`SELECT id, title, description, completed, subject, due_date 
		FROM tasks WHERE user_id = ?`, userID)
	if err != nil {
		log.Println("Select tasks error:", err)
		sendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var dueDate sql.NullString
		var desc sql.NullString
		var subj sql.NullString
		var compRaw sql.RawBytes
		
		if err := rows.Scan(&t.ID, &t.Title, &desc, &compRaw, &subj, &dueDate); err != nil {
			log.Println("Scan task error:", err)
			continue
		}
		
		if len(compRaw) > 0 && (compRaw[0] == 1 || compRaw[0] == '1') {
			t.Completed = true
		} else {
			t.Completed = false
		}
		
		t.Description = desc.String
		t.Subject = subj.String
		t.DueDate = dueDate.String
		tasks = append(tasks, t)
	}

	if tasks == nil {
		tasks = []Task{}
	}

	sendResponse(w, http.StatusOK, "OK", tasks)
}

func getTaskByIdHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		sendResponse(w, http.StatusBadRequest, "Invalid ID format", nil)
		return
	}

	userID := getUserIdFromRequest(r)

	var t Task
	var dueDate sql.NullString
	var desc sql.NullString
	var subj sql.NullString
	var compRaw sql.RawBytes

	err = db.QueryRow(`SELECT id, title, description, completed, subject, due_date 
		FROM tasks WHERE id = ? AND user_id = ?`, id, userID).
		Scan(&t.ID, &t.Title, &desc, &compRaw, &subj, &dueDate)
	
	if err != nil {
		if err == sql.ErrNoRows {
			sendResponse(w, http.StatusNotFound, "Task not found", nil)
			return
		}
		log.Println("Select task error:", err)
		sendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}
	
	if len(compRaw) > 0 && (compRaw[0] == 1 || compRaw[0] == '1') {
		t.Completed = true
	} else {
		t.Completed = false
	}
	t.Description = desc.String
	t.Subject = subj.String
	t.DueDate = dueDate.String

	sendResponse(w, http.StatusOK, "OK", t)
}

func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		sendResponse(w, http.StatusBadRequest, "Invalid ID format", nil)
		return
	}

	userID := getUserIdFromRequest(r)

	result, err := db.Exec("DELETE FROM tasks WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		log.Println("Delete task error:", err)
		sendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		sendResponse(w, http.StatusNotFound, "Task not found or not yours", nil)
		return
	}

	sendResponse(w, http.StatusOK, "OK", nil)
}

func getTasksByStatusHandler(w http.ResponseWriter, r *http.Request) {
	completedStr := r.URL.Query().Get("completed")
	completed := completedStr == "true"
	userID := getUserIdFromRequest(r)

	rows, err := db.Query(`SELECT id, title, description, completed, subject, due_date 
		FROM tasks WHERE user_id = ? AND completed = ?`, userID, completed)
	if err != nil {
		log.Println("Select tasks by status error:", err)
		sendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var dueDate sql.NullString
		var desc sql.NullString
		var subj sql.NullString
		var compRaw sql.RawBytes
		
		if err := rows.Scan(&t.ID, &t.Title, &desc, &compRaw, &subj, &dueDate); err != nil {
			log.Println("Scan task error:", err)
			continue
		}
		
		if len(compRaw) > 0 && (compRaw[0] == 1 || compRaw[0] == '1') {
			t.Completed = true
		} else {
			t.Completed = false
		}
		t.Description = desc.String
		t.Subject = subj.String
		t.DueDate = dueDate.String
		tasks = append(tasks, t)
	}

	if tasks == nil {
		tasks = []Task{}
	}

	sendResponse(w, http.StatusOK, "OK", tasks)
}

// CORS Middleware to allow requests from frontend
func corsMiddleware(next http.Handler) http.Handler {
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

func initDB() {
	var err error

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	// Construct DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPass, dbHost, dbPort, dbName)

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}

	if err = db.Ping(); err != nil {
		log.Println("WARNING: Database connection failed. Make sure MySQL is running and DB '" + dbName + "' exists.")
		log.Println("Error: ", err)
	} else {
		log.Printf("Successfully connected to MySQL database (%s)!\n", dbName)
	}
}

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize JWT Secret from environment
	secretStr := os.Getenv("JWT_SECRET")
	if secretStr == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}
	jwtSecret = []byte(secretStr)

	initDB()
	defer db.Close()

	mux := http.NewServeMux()

	// Auth endpoints
	mux.HandleFunc("POST /api/auth/register", registerHandler)
	mux.HandleFunc("POST /api/auth/login", loginHandler)

	// Task endpoints
	mux.HandleFunc("POST /api/tasks", authMiddleware(createTaskHandler))
	mux.HandleFunc("PUT /api/tasks", authMiddleware(updateTaskHandler))
	mux.HandleFunc("GET /api/tasks", authMiddleware(getTasksHandler))
	mux.HandleFunc("GET /api/tasks/{id}", authMiddleware(getTaskByIdHandler))
	mux.HandleFunc("DELETE /api/tasks/{id}", authMiddleware(deleteTaskHandler))
	mux.HandleFunc("GET /api/tasks/status", authMiddleware(getTasksByStatusHandler))

	// Wrap mux with CORS middleware
	handler := corsMiddleware(mux)

	fmt.Println("Go Backend Server is starting on port 8081...")
	if err := http.ListenAndServe(":8081", handler); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
