package api

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"business_services/internal/app/handler"
	"business_services/internal/app/repository"
	"business_services/internal/storage"
)

func StartServer(db *gorm.DB) {
	log.Println("Starting server")

	// ---------- Storage ----------
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

	// ---------- Repository ----------
	repo := repository.New(db)
	h := handler.NewHandler(repo, st)

	// ---------- Router ----------
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

	// ---------- HTML-страницы ----------
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/business_services_feed")
	})
	r.GET("/business_services_feed", h.Feed)
	r.GET("/business_services_feed/:id", h.Feed)
	r.GET("/business_services_add", h.AddPage)
	r.GET("/business_services_grid", h.Grid)
	r.GET("/business_services_media/*key", h.Media)

	// ---------- API ----------
	api := r.Group("/api")
	{
		// Домен услуг
		bs := api.Group("/business_services")
		{
			bs.GET("", h.APIList)
			bs.GET("/feed", h.APIFeed)
			bs.GET("/feed/:id", h.APIFeed)
			bs.GET("/draft", h.APIDraft)
			bs.POST("", h.APICreate)
			bs.PUT("/:id/publish", h.APIPublish)
			bs.DELETE("/:id", h.APIDelete)
			bs.POST("/:id/like", h.APILike)
		}

		// Домен пользователей (заглушки)
		users := api.Group("/business_services_users")
		{
			users.POST("/register", h.APIRegister)
			users.POST("/login", h.APILogin)
			users.POST("/logout", h.APILogout)
		}
	}

	log.Println("Сервер слушает :8080")
	if err := r.Run(":8080"); err != nil {
		logrus.WithError(err).Fatal("сервер упал")
	}
	log.Println("Server down")
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
