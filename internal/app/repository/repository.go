package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Ошибки, которые хендлеры будут превращать в HTTP-коды.
var (
	ErrNotFound    = errors.New("запись не найдена")
	ErrNotOwner    = errors.New("нет прав на изменение")
	ErrWrongStatus = errors.New("неверный статус для операции")
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ---------- Чтение ----------

// GetPublished — все опубликованные услуги (для HTML-ленты).
func (r *Repository) GetPublished() ([]Service, error) {
	var res []Service
	err := r.db.
		Preload("Likes").
		Where("status = ?", StatusPublished).
		Order("id ASC").
		Find(&res).Error
	return res, err
}

// GetPublishedByID — конкретная опубликованная услуга.
func (r *Repository) GetPublishedByID(id int) (Service, error) {
	var s Service
	err := r.db.
		Preload("Likes").
		Where("id = ? AND status = ?", id, StatusPublished).
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Service{}, ErrNotFound
	}
	return s, err
}

// GetNextPublished — следующая по кругу: если id последний — вернёт первый.
func (r *Repository) GetNextPublished(id int) (Service, error) {
	var s Service

	// Следующий по id
	err := r.db.
		Preload("Likes").
		Where("status = ? AND id > ?", StatusPublished, id).
		Order("id ASC").
		First(&s).Error
	if err == nil {
		return s, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Service{}, err
	}

	// Следующего нет — вернуть первый
	err = r.db.
		Preload("Likes").
		Where("status = ?", StatusPublished).
		Order("id ASC").
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Service{}, ErrNotFound
	}
	return s, err
}

// GetPublishedFiltered — список с фильтром по цене (0 = без ограничения).
func (r *Repository) GetPublishedFiltered(minPrice, maxPrice int) ([]Service, error) {
	q := r.db.Preload("Likes").Where("status = ?", StatusPublished)
	if minPrice > 0 {
		q = q.Where("price >= ?", minPrice)
	}
	if maxPrice > 0 {
		q = q.Where("price <= ?", maxPrice)
	}
	var res []Service
	err := q.Order("id ASC").Find(&res).Error
	return res, err
}

// GetDraft — черновик текущего пользователя (не более одного).
// Если черновиков несколько — вернёт самый свежий.
func (r *Repository) GetDraft(userID int) (Service, error) {
	var s Service
	err := r.db.
		Preload("Likes").
		Where("creator_id = ? AND status = ?", userID, StatusDraft).
		Order("id DESC").
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Service{}, ErrNotFound
	}
	return s, err
}

// ---------- Запись ----------

// Create — создаёт новую услугу в статусе draft.
func (r *Repository) Create(s *Service) error {
	return r.db.Create(s).Error
}

// Publish — draft → published. Только создатель. Обратно в draft нельзя.
func (r *Repository) Publish(id, userID int) error {
	var s Service
	if err := r.db.First(&s, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if s.CreatorID != userID {
		return ErrNotOwner
	}
	if s.Status != StatusDraft {
		return ErrWrongStatus
	}
	now := time.Now()
	return r.db.Model(&s).Updates(map[string]interface{}{
		"status":       StatusPublished,
		"published_at": &now,
	}).Error
}

// SoftDelete — ставит status=deleted и deleted_at через GORM.
// Только создатель. Строка остаётся в БД.
func (r *Repository) SoftDelete(id, userID int) error {
	var s Service
	if err := r.db.First(&s, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if s.CreatorID != userID {
		return ErrNotOwner
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&s).Update("status", StatusDeleted).Error; err != nil {
			return err
		}
		return tx.Delete(&s).Error // GORM поставит deleted_at
	})
}

// ---------- Лайки ----------

// SetLike — value=1 ставит лайк (идемпотентно), value=0 снимает.
func (r *Repository) SetLike(serviceID, userID, value int) error {
	var s Service
	if err := r.db.
		Where("id = ? AND status = ?", serviceID, StatusPublished).
		First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if value == 1 {
			like := Like{ServiceID: serviceID, UserID: userID}
			return tx.Where(Like{ServiceID: serviceID, UserID: userID}).
				FirstOrCreate(&like).Error
		}
		return tx.
			Where("service_id = ? AND user_id = ?", serviceID, userID).
			Delete(&Like{}).Error
	})
}

// LikedIDs — множество service_id, которые текущий пользователь лайкнул.
// Используется в ленте для признака is_liked.
func (r *Repository) LikedIDs(userID int) (map[int]bool, error) {
	var likes []Like
	if err := r.db.Where("user_id = ?", userID).Find(&likes).Error; err != nil {
		return nil, err
	}
	res := make(map[int]bool, len(likes))
	for _, l := range likes {
		res[l.ServiceID] = true
	}
	return res, nil
}

func (r *Repository) UpdateDraft(s *Service) error {
	return r.db.Model(s).
		Select("price", "image_key", "video_key").
		Updates(map[string]interface{}{
			"price":     s.Price,
			"image_key": s.ImageKey,
			"video_key": s.VideoKey,
		}).Error
}
