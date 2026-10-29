package repository

import (
	"strings"
	"time"

	"business_services/internal/storage"
)

// --- Модели GORM (соответствуют таблицам в PostgreSQL) ---

type Service struct {
	ID          int        `gorm:"primaryKey;column:id"`
	Title       string     `gorm:"column:title"`
	Description string     `gorm:"column:description"`
	Status      string     `gorm:"column:status"` // draft, published, deleted
	ImageKey    string     `gorm:"column:image_key"`
	VideoKey    string     `gorm:"column:video_key"`
	Price       int        `gorm:"column:price"`
	Deadline    *time.Time `gorm:"column:deadline"`
	CreatorID   int        `gorm:"column:creator_id"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
}

func (Service) TableName() string { return "services" }

type User struct {
	ID        int       `gorm:"primaryKey;column:id"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (User) TableName() string { return "users" }

type Like struct {
	ID        int       `gorm:"primaryKey;column:id"`
	UserID    int       `gorm:"column:user_id"`
	ServiceID int       `gorm:"column:service_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Like) TableName() string { return "likes" }

// --- Методы-хелперы для шаблонов ---

// DeadlineStr возвращает отформатированную дату или прочерк.
func (s Service) DeadlineStr() string {
	if s.Deadline == nil || s.Deadline.IsZero() {
		return "—"
	}
	return s.Deadline.Format("02.01.2006 15:04")
}

// ImageURL формирует URL для картинки.
// Если ключ пустой или выглядит как неверный внешний URL — возвращает дефолтную картинку.
func (s Service) ImageURL(st *storage.Storage) string {
	if s.ImageKey == "" || strings.HasPrefix(s.ImageKey, "http") {
		return "/static/default.jpg"
	}
	return "/business_services_media/" + s.ImageKey
}

// VideoURL формирует URL для видео.
func (s Service) VideoURL(st *storage.Storage) string {
	if s.VideoKey == "" || strings.HasPrefix(s.VideoKey, "http") {
		return "/static/default.mp4"
	}
	return "/business_services_media/" + s.VideoKey
}
