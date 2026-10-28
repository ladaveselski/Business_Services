package api

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"business_services/internal/app/handler"
	"business_services/internal/app/repository"
	"business_services/internal/storage"
)

func StartServer() {
	log.Println("Starting server")

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
		log.Println("Режим: MinIO")
	} else {
		log.Println("Режим: локальные файлы (resources/seed/)")
	}

	repo, err := repository.NewRepository()
	if err != nil {
		logrus.WithError(err).Fatal("репозиторий")
	}

	h := handler.NewHandler(repo, st)

	r := gin.Default()

	r.LoadHTMLFiles(
		"templates/partials/head.html",
		"templates/partials/bottom_nav.html",
		"templates/partials/like_icon.html",
		"templates/business_services_feed.html",
		"templates/business_services_add.html",
		"templates/business_services_grid.html",
	)

	r.Static("/static", "./resources")

	// Корень редиректит на ленту
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/business_services_feed")
	})

	// Все маршруты — плоские, с подчёркиваниями
	r.GET("/business_services_feed", h.Feed)
	r.GET("/business_services_feed/:id", h.Feed)
	r.GET("/business_services_add", h.AddPage)
	r.GET("/business_services_grid", h.Grid)
	r.GET("/business_services_media/*key", h.Media)

	r.Run(":8080")
	log.Println("Server down")
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
