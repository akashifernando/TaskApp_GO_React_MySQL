package main

import (
	"fmt"
	"log"
	"net/http"

	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/middleware"
	"backend/internal/task"
	"backend/internal/user"
)

func main() {
	// 1. Load Configuration
	cfg := config.Load()

	// 2. Initialize Database
	db := database.Init(cfg)

	// 3. Auto-Migration
	log.Println("Running Auto-Migration...")
	err := db.AutoMigrate(&user.AppUser{}, &task.Task{})
	if err != nil {
		log.Fatal("Auto-Migration failed: ", err)
	}

	// 4. Initialize Services & Repositories
	authService := auth.NewService(cfg.JWTSecret)
	taskRepo := task.NewRepository(db)

	// 5. Initialize Handlers
	authHandler := auth.NewHandler(db, authService)
	taskHandler := task.NewHandler(taskRepo)

	// 6. Setup Router
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	// Task routes (Protected)
	mux.HandleFunc("POST /api/tasks", middleware.Auth(authService, taskHandler.CreateTask))
	mux.HandleFunc("PUT /api/tasks", middleware.Auth(authService, taskHandler.UpdateTask))
	mux.HandleFunc("GET /api/tasks", middleware.Auth(authService, taskHandler.GetTasks))
	mux.HandleFunc("GET /api/tasks/{id}", middleware.Auth(authService, taskHandler.GetTaskByID))
	mux.HandleFunc("DELETE /api/tasks/{id}", middleware.Auth(authService, taskHandler.DeleteTask))
	mux.HandleFunc("GET /api/tasks/status", middleware.Auth(authService, taskHandler.GetTasksByStatus))

	// 7. Start Server
	handler := middleware.CORS(mux)

	fmt.Printf("Go Backend Server is starting on port %s...\n", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
