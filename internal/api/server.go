package api

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"business_services/internal/app/handler"
	"business_services/internal/app/repository"
	"business_services/internal/database"
	"business_services/internal/storage"
)

func StartServer() {
	log.Println("Starting server")

	// 1. Подключение к PostgreSQL через GORM
	db, err := database.Connect()
	if err != nil {
		logrus.WithError(err).Fatal("не удалось подключиться к PostgreSQL")
	}
	log.Println("PostgreSQL подключён")

	// 2. Подключение к MinIO (с fallback на локальные файлы)
	st, err := storage.NewStorage(storage.Config{
		Endpoint:  env("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey: env("MINIO_ACCESS_KEY", "admin"),
		SecretKey: env("MINIO_SECRET_KEY", "admin12345"),
		Bucket:    env("MINIO_BUCKET", "metoda-media"),
		UseSSL:    env("MINIO_USE_SSL", "false") == "true",
	})
	if err != nil {
		logrus.WithError(err).Error("ошибка инициализации Storage")
	}
	if st.IsAvailable() {
		log.Println("Режим MinIO: включён")
	} else {
		log.Println("Режим MinIO: выключен (локальные файлы resources/seed/)")
	}

	// 3. Репозиторий теперь принимает *gorm.DB
	repo, err := repository.NewRepository(db)
	if err != nil {
		logrus.WithError(err).Fatal("репозиторий")
	}

	// 4. Хендлер получает репозиторий и хранилище
	h := handler.NewHandler(repo, st)

	// 5. Настройка Gin
	r := gin.Default()

	// Загрузка шаблонов
	r.LoadHTMLFiles(
		"templates/partials/head.html",
		"templates/partials/bottom_nav.html",
		"templates/partials/like_icon.html",
		"templates/business_services_feed.html",
		"templates/business_services_add.html",
		"templates/business_services_grid.html",
	)

	// Статика (CSS, дефолтные фото/видео)
	r.Static("/static", "./resources")

	// 6. Маршруты
	// Корень редиректит на ленту
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/business_services_feed")
	})

	// --- Три GET-запроса ---
	r.GET("/business_services_feed", h.Feed)
	r.GET("/business_services_feed/:id", h.Feed)
	r.GET("/business_services_add", h.AddPage)
	r.GET("/business_services_grid", h.Grid)

	// --- Три POST-запроса ---
	r.POST("/business_services_add", h.CreateDraft) // создание черновика (кнопка "Далее")
	r.POST("/business_services_publish", h.Publish) // публикация (смена статуса на published)
	r.POST("/business_services_delete", h.Delete)   // удаление через SQL UPDATE (без ORM)

	// --- Медиа из MinIO ---
	r.GET("/business_services_media/*key", h.Media)

	r.Run(":8081")
	log.Println("Server down")
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
