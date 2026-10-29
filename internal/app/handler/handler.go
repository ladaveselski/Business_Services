package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"business_services/internal/app/repository"
	"business_services/internal/storage"
)

type Handler struct {
	Repository *repository.Repository
	Storage    *storage.Storage
}

func NewHandler(r *repository.Repository, s *storage.Storage) *Handler {
	return &Handler{Repository: r, Storage: s}
}

// =====================================================================
// GET-ЗАПРОСЫ (3 штуки)
// =====================================================================

// 1. GET /business_services_feed и /business_services_feed/:id
func (h *Handler) Feed(ctx *gin.Context) {
	idStr := ctx.Param("id")
	next := ctx.Query("next") == "true"

	var service repository.Service
	var err error

	if idStr == "" {
		service, err = h.Repository.GetFirstPublished()
	} else {
		id, e := strconv.Atoi(idStr)
		if e != nil {
			ctx.Status(http.StatusBadRequest)
			return
		}
		if next {
			service, err = h.Repository.GetNextPublished(id)
		} else {
			service, err = h.Repository.GetPublishedByID(id)
		}
	}

	if err != nil {
		logrus.WithError(err).Warn("лента: услуга не найдена")
		ctx.String(http.StatusNotFound, "Услуга не найдена или удалена")
		return
	}

	// Считаем лайки через GORM (по заданию лабы 2 только отображение)
	var likes int64
	h.Repository.DB.Table("likes").Where("service_id = ?", service.ID).Count(&likes)

	ctx.HTML(http.StatusOK, "business_services_feed.html", gin.H{
		"title":     service.Title,
		"active":    "feed",
		"service":   service,
		"likes":     likes,
		"image_url": service.ImageURL(h.Storage),
		"video_url": service.VideoURL(h.Storage),
		"deadline":  service.DeadlineStr(),
	})
}

// 2. GET /business_services_add
func (h *Handler) AddPage(ctx *gin.Context) {
	// Ищем черновик для пользователя с ID=1
	draft, err := h.Repository.GetDraftByCreator(1)

	// Если черновика нет, передаем пустую структуру
	if err != nil {
		draft = repository.Service{}
	}

	ctx.HTML(http.StatusOK, "business_services_add.html", gin.H{
		"title":     "Новая заявка",
		"active":    "add",
		"service":   draft,
		"is_draft":  draft.ID != 0,
		"image_url": draft.ImageURL(h.Storage),
		"video_url": draft.VideoURL(h.Storage),
	})
}

// 3. GET /business_services_grid
func (h *Handler) Grid(ctx *gin.Context) {
	minStr := ctx.Query("min")
	maxStr := ctx.Query("max")
	min, _ := strconv.Atoi(minStr)
	max, _ := strconv.Atoi(maxStr)

	services, err := h.Repository.GetPublishedFiltered(min, max)
	if err != nil {
		logrus.WithError(err).Error("плитка: ошибка выборки")
	}

	type GridCard struct {
		repository.Service
		Likes    int
		ImageURL string
	}

	cards := make([]GridCard, 0, len(services))
	for _, s := range services {
		var likes int64
		h.Repository.DB.Table("likes").Where("service_id = ?", s.ID).Count(&likes)

		cards = append(cards, GridCard{
			Service:  s,
			Likes:    int(likes),
			ImageURL: s.ImageURL(h.Storage),
		})
	}

	ctx.HTML(http.StatusOK, "business_services_grid.html", gin.H{
		"title":  "Все заявки",
		"active": "grid",
		"cards":  cards,
		"min":    minStr,
		"max":    maxStr,
	})
}

// =====================================================================
// POST-ЗАПРОСЫ (3 штуки)
// =====================================================================

// 4. POST /business_services_add (Кнопка "Далее" — создание/обновление черновика)
func (h *Handler) CreateDraft(ctx *gin.Context) {
	title := ctx.PostForm("title")

	// По заданию: новые фото и видео не сохраняются, но заполняем неверные url
	invalidImage := "invalid/image_" + uuid.NewString() + ".jpg"
	invalidVideo := "invalid/video_" + uuid.NewString() + ".mp4"

	// Проверяем, есть ли уже черновик у пользователя 1
	existingDraft, err := h.Repository.GetDraftByCreator(1)

	if err == nil && existingDraft.ID != 0 {
		// Черновик уже есть — обновляем название и редиректим на ту же страницу
		h.Repository.UpdateDraftTitle(existingDraft.ID, title)
		ctx.Redirect(http.StatusSeeOther, "/business_services_add")
		return
	}

	// Создаем новый черновик
	err = h.Repository.CreateDraft(&repository.Service{
		Title:     title,
		Status:    "draft",
		ImageKey:  invalidImage,
		VideoKey:  invalidVideo,
		CreatorID: 1,
	})

	if err != nil {
		logrus.WithError(err).Error("ошибка создания черновика")
		ctx.String(http.StatusInternalServerError, "Ошибка создания")
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/business_services_add")
}

// 5. POST /business_services_publish (Публикация черновика)
func (h *Handler) Publish(ctx *gin.Context) {
	idStr := ctx.PostForm("id")
	id, _ := strconv.Atoi(idStr)

	description := ctx.PostForm("description")
	priceStr := ctx.PostForm("price")
	price, _ := strconv.Atoi(priceStr)
	deadlineStr := ctx.PostForm("deadline") // формат HTML: 2026-10-05T14:30

	err := h.Repository.PublishDraft(id, description, price, deadlineStr)
	if err != nil {
		logrus.WithError(err).Error("ошибка публикации")
		ctx.String(http.StatusInternalServerError, "Ошибка публикации")
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/business_services_feed")
}

// 6. POST /business_services_delete (Логическое удаление через SQL UPDATE, без ORM)
func (h *Handler) Delete(ctx *gin.Context) {
	idStr := ctx.PostForm("id")
	id, _ := strconv.Atoi(idStr)

	err := h.Repository.LogicalDeleteSQL(id)
	if err != nil {
		logrus.WithError(err).Error("ошибка удаления")
		ctx.String(http.StatusInternalServerError, "Ошибка удаления")
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/business_services_grid")
}

// =====================================================================
// МЕДИА
// =====================================================================

func (h *Handler) Media(ctx *gin.Context) {
	key := strings.TrimPrefix(ctx.Param("key"), "/")
	if key == "" || strings.HasPrefix(key, "invalid/") {
		ctx.File("./resources/static/default.jpg")
		return
	}

	if h.Storage.IsAvailable() {
		h.mediaFromMinIO(ctx, key)
		return
	}
	h.mediaFromLocal(ctx, key)
}

func (h *Handler) mediaFromMinIO(ctx *gin.Context, key string) {
	bg := ctx.Request.Context()

	info, err := h.Storage.Stat(bg, key)
	if err != nil {
		ctx.File("./resources/static/default.jpg")
		return
	}

	obj, err := h.Storage.Get(bg, key)
	if err != nil {
		ctx.File("./resources/static/default.jpg")
		return
	}
	defer obj.Close()

	ctx.Header("Content-Type", info.ContentType)
	ctx.Header("Content-Length", strconv.FormatInt(info.Size, 10))
	ctx.Header("Cache-Control", "public, max-age=86400")
	ctx.Header("ETag", info.ETag)

	ctx.Status(http.StatusOK)
	io.Copy(ctx.Writer, obj)
}

func (h *Handler) mediaFromLocal(ctx *gin.Context, key string) {
	name := filepath.Base(key)
	localPath := filepath.Join("resources", "seed", name)

	info, err := os.Stat(localPath)
	if err != nil || info.IsDir() {
		ctx.File("./resources/static/default.jpg")
		return
	}

	f, err := os.Open(localPath)
	if err != nil {
		ctx.File("./resources/static/default.jpg")
		return
	}
	defer f.Close()

	ctx.Header("Content-Type", contentTypeByExt(filepath.Ext(name)))
	ctx.Header("Content-Length", strconv.FormatInt(info.Size(), 10))
	ctx.Status(http.StatusOK)
	io.Copy(ctx.Writer, f)
}

func contentTypeByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".mp4":
		return "video/mp4"
	}
	return "application/octet-stream"
}
