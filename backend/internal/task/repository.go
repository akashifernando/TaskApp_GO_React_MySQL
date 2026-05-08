package task

import "gorm.io/gorm"

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(task *Task) error {
	return r.db.Create(task).Error
}

func (r *Repository) Update(task *Task) error {
	return r.db.Save(task).Error
}

func (r *Repository) FindByUserID(userID int64) ([]Task, error) {
	var tasks []Task
	err := r.db.Where("user_id = ?", userID).Find(&tasks).Error
	return tasks, err
}

func (r *Repository) FindByIDAndUserID(id, userID int64) (*Task, error) {
	var task Task
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&task).Error
	return &task, err
}

func (r *Repository) Delete(id, userID int64) (int64, error) {
	result := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&Task{})
	return result.RowsAffected, result.Error
}

func (r *Repository) FindByStatus(userID int64, completed bool) ([]Task, error) {
	var tasks []Task
	err := r.db.Where("user_id = ? AND completed = ?", userID, completed).Find(&tasks).Error
	return tasks, err
}
