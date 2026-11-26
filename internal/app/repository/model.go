package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ---------- Статусы ----------
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusDeleted   Status = "deleted"
)

// ---------- User ----------
type User struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// ---------- Service ----------
type Service struct {
	ID          int    `gorm:"primaryKey" json:"id"`
	Status      Status `gorm:"type:varchar(20);not null;default:'draft';index" json:"status"`
	Title       string `gorm:"not null" json:"title"`
	Description string `json:"description"`

	Price      int `json:"price"`
	AvgTimeMin int `gorm:"column:avg_time_min;default:0" json:"avg_time_min"`

	ImageKey string `json:"image_key"`
	VideoKey string `json:"video_key"`

	// Связи
	CreatorID int  `gorm:"not null;index" json:"creator_id"`
	Creator   User `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`

	Likes []Like `gorm:"foreignKey:ServiceID" json:"-"`

	// Даты
	CreatedAt   time.Time      `json:"created_at"`
	PublishedAt *time.Time     `json:"published_at,omitempty"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// ---------- Like ----------
type Like struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	ServiceID int       `gorm:"not null;uniqueIndex:idx_service_user" json:"service_id"`
	UserID    int       `gorm:"not null;uniqueIndex:idx_service_user" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ---------- Хелперы модели ----------

func (s Service) LikesCount() int {
	return len(s.Likes)
}

func (s Service) ImageURL() string {
	if s.ImageKey == "" {
		return "/static/img/placeholder.png"
	}
	return "/business_services_media/" + s.ImageKey
}

func (s Service) VideoURL() string {
	if s.VideoKey == "" {
		return ""
	}
	return "/business_services_media/" + s.VideoKey
}

// AvgTimeStr — «2 ч 30 мин» / «45 мин» / «—».
func (s Service) AvgTimeStr() string {
	if s.AvgTimeMin <= 0 {
		return "—"
	}
	h := s.AvgTimeMin / 60
	m := s.AvgTimeMin % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%d ч %d мин", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%d ч", h)
	}
	return fmt.Sprintf("%d мин", m)
}
