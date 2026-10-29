package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

func NewRepository(db *gorm.DB) (*Repository, error) {
	return &Repository{DB: db}, nil
}

// 1. GetFirstPublished — первая опубликованная (для ленты)
func (r *Repository) GetFirstPublished() (Service, error) {
	var service Service
	err := r.DB.Where("status = ?", "published").Order("id ASC").First(&service).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Service{}, fmt.Errorf("нет опубликованных услуг")
		}
		return Service{}, err
	}
	return service, nil
}

// 2. GetPublishedByID — по ID
func (r *Repository) GetPublishedByID(id int) (Service, error) {
	var service Service
	err := r.DB.Where("id = ? AND status = ?", id, "published").First(&service).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Service{}, fmt.Errorf("услуга %d не найдена или удалена", id)
		}
		return Service{}, err
	}
	return service, nil
}

// 3. GetNextPublished — следующая после ID
func (r *Repository) GetNextPublished(currentID int) (Service, error) {
	var service Service
	err := r.DB.Where("id > ? AND status = ?", currentID, "published").Order("id ASC").First(&service).Error
	if err == nil {
		return service, nil
	}
	return r.GetFirstPublished()
}

// 4. GetDraftByCreator — черновик пользователя
func (r *Repository) GetDraftByCreator(userID int) (Service, error) {
	var service Service
	err := r.DB.Where("creator_id = ? AND status = ?", userID, "draft").First(&service).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Service{}, fmt.Errorf("черновик не найден")
		}
		return Service{}, err
	}
	return service, nil
}

// 5. GetPublishedFiltered — список с фильтром по цене
func (r *Repository) GetPublishedFiltered(minPrice, maxPrice int) ([]Service, error) {
	var services []Service
	query := r.DB.Where("status = ?", "published")

	if minPrice > 0 {
		query = query.Where("price >= ?", minPrice)
	}
	if maxPrice > 0 {
		query = query.Where("price <= ?", maxPrice)
	}

	err := query.Order("id ASC").Find(&services).Error
	return services, err
}

// 6. CreateDraft — создание черновика
func (r *Repository) CreateDraft(s *Service) error {
	s.Status = "draft"
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	return r.DB.Create(s).Error
}

// 7. UpdateDraftTitle — обновление названия черновика
func (r *Repository) UpdateDraftTitle(id int, title string) error {
	return r.DB.Model(&Service{}).Where("id = ?", id).Updates(map[string]interface{}{
		"title":      title,
		"updated_at": time.Now(),
	}).Error
}

// 8. PublishDraft — публикация черновика
func (r *Repository) PublishDraft(id int, description string, price int, deadlineStr string) error {
	var deadline *time.Time
	if deadlineStr != "" {
		parsed, err := time.Parse("2006-01-02T15:04", deadlineStr)
		if err == nil {
			deadline = &parsed
		}
	}

	updates := map[string]interface{}{
		"status":      "published",
		"description": description,
		"price":       price,
		"updated_at":  time.Now(),
	}
	if deadline != nil {
		updates["deadline"] = deadline
	}

	return r.DB.Model(&Service{}).Where("id = ? AND status = ?", id, "draft").Updates(updates).Error
}

// 9. LogicalDeleteSQL — удаление через чистый SQL
func (r *Repository) LogicalDeleteSQL(id int) error {
	sqlDB, err := r.DB.DB()
	if err != nil {
		return fmt.Errorf("не удалось получить *sql.DB: %w", err)
	}

	query := `UPDATE services SET status = 'deleted', updated_at = NOW() WHERE id = $1 AND status != 'deleted' RETURNING id`
	var returnedID int
	err = sqlDB.QueryRowContext(context.Background(), query, id).Scan(&returnedID)
	if err != nil {
		return fmt.Errorf("ошибка логического удаления: %w", err)
	}
	return nil
}
