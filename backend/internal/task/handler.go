package task

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"backend/pkg/utils"
	"gorm.io/gorm"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func getUserIdFromRequest(r *http.Request) int64 {
	idStr := r.Header.Get("X-User-ID")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	return id
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	userID := getUserIdFromRequest(r)
	task := Task{
		Title:       req.Title,
		Description: req.Description,
		Completed:   req.Completed,
		Subject:     req.Subject,
		DueDate:     req.DueDate,
		UserID:      userID,
	}

	if err := h.repo.Create(&task).Error; err != nil {
		log.Println("Insert task error:", err)
		utils.SendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	req.ID = task.ID
	utils.SendResponse(w, http.StatusOK, "OK", req)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	userID := getUserIdFromRequest(r)
	
	task, err := h.repo.FindByIDAndUserID(req.ID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.SendResponse(w, http.StatusNotFound, "Task not found or not yours", nil)
			return
		}
		utils.SendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	task.Title = req.Title
	task.Description = req.Description
	task.Completed = req.Completed
	task.Subject = req.Subject
	task.DueDate = req.DueDate

	if err := h.repo.Update(task); err != nil {
		log.Println("Update task error:", err)
		utils.SendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	utils.SendResponse(w, http.StatusOK, "OK", req)
}

func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	userID := getUserIdFromRequest(r)

	tasks, err := h.repo.FindByUserID(userID)
	if err != nil {
		log.Println("Select tasks error:", err)
		utils.SendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	utils.SendResponse(w, http.StatusOK, "OK", tasks)
}

func (h *Handler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.SendResponse(w, http.StatusBadRequest, "Invalid ID format", nil)
		return
	}

	userID := getUserIdFromRequest(r)

	task, err := h.repo.FindByIDAndUserID(id, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.SendResponse(w, http.StatusNotFound, "Task not found", nil)
			return
		}
		log.Println("Select task error:", err)
		utils.SendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	utils.SendResponse(w, http.StatusOK, "OK", task)
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.SendResponse(w, http.StatusBadRequest, "Invalid ID format", nil)
		return
	}

	userID := getUserIdFromRequest(r)

	rowsAffected, err := h.repo.Delete(id, userID)
	if err != nil {
		log.Println("Delete task error:", err)
		utils.SendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	if rowsAffected == 0 {
		utils.SendResponse(w, http.StatusNotFound, "Task not found or not yours", nil)
		return
	}

	utils.SendResponse(w, http.StatusOK, "OK", nil)
}

func (h *Handler) GetTasksByStatus(w http.ResponseWriter, r *http.Request) {
	completedStr := r.URL.Query().Get("completed")
	completed := completedStr == "true"
	userID := getUserIdFromRequest(r)

	tasks, err := h.repo.FindByStatus(userID, completed)
	if err != nil {
		log.Println("Select tasks by status error:", err)
		utils.SendResponse(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	utils.SendResponse(w, http.StatusOK, "OK", tasks)
}
