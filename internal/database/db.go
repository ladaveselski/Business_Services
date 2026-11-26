package database

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"business_services/internal/app/repository"
	"business_services/internal/auth"
)

// New открывает БД, применяет миграции и гарантирует наличие фиксированного пользователя.
func New(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("открытие БД: %w", err)
	}

	// Миграции
	if err := db.AutoMigrate(
		&repository.User{},
		&repository.Service{},
		&repository.Like{},
	); err != nil {
		return nil, fmt.Errorf("миграции: %w", err)
	}

	// Гарантируем существование фиксированного пользователя (creator).
	// Без этого вставка Service с CreatorID=1 упадёт на FK.
	if err := ensureFixedUser(db); err != nil {
		return nil, fmt.Errorf("создание фиксированного пользователя: %w", err)
	}

	log.Printf("БД готова: %s", dsn)
	return db, nil
}

func ensureFixedUser(db *gorm.DB) error {
	user := auth.CurrentUser()
	// Пароль-заглушка, чтобы не пустое поле.
	user.Password = "stub"

	return db.
		Where(repository.User{ID: user.ID}).
		Attrs(user).
		FirstOrCreate(&user).Error
}
