package task

import "time"

type Task struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"not null" json:"title"`
	Description string    `json:"description"`
	Completed   bool      `gorm:"default:false" json:"completed"`
	Subject     string    `json:"category"`
	DueDate     string    `json:"dueDate"`
	UserID      int64     `json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TaskRequest struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	Subject     string `json:"category"`
	DueDate     string `json:"dueDate"`
}
