package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
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

// ---------- GET /business_services_feed ----------
func (h *Handler) Feed(ctx *gin.Context) {
	idStr := ctx.Param("id")
	next := ctx.Query("next") == "true"

	var order repository.BusinessServicesOrder
	var err error

	if idStr == "" {
		orders, e := h.Repository.GetPublished()
		if e != nil || len(orders) == 0 {
			ctx.Status(http.StatusNotFound)
			return
		}
		order = orders[0]
	} else {
		id, e := strconv.Atoi(idStr)
		if e != nil {
			ctx.Status(http.StatusBadRequest)
			return
		}
		if next {
			order, err = h.Repository.GetNextPublished(id)
		} else {
			order, err = h.Repository.GetPublishedByID(id)
		}
		if err != nil {
			logrus.WithError(err).Warn("лента: услуга не найдена")
			ctx.Status(http.StatusNotFound)
			return
		}
	}

	ctx.HTML(http.StatusOK, "business_services_feed.html", gin.H{
		"title":     order.Title,
		"active":    "feed",
		"order":     order,
		"likes":     order.LikesCount(),
		"image_url": order.ImageURL(),
		"video_url": order.VideoURL(),
		"deadline":  order.DeadlineStr(),
	})
}

// ---------- GET /business_services_add ----------
func (h *Handler) AddPage(ctx *gin.Context) {
	draft, err := h.Repository.GetDraft()
	if err != nil {
		logrus.WithError(err).Warn("черновик не найден")
	}

	ctx.HTML(http.StatusOK, "business_services_add.html", gin.H{
		"title":     "Новая заявка",
		"active":    "add",
		"order":     draft,
		"image_url": draft.ImageURL(),
		"video_url": draft.VideoURL(),
	})
}

// ---------- GET /business_services_grid ----------
func (h *Handler) Grid(ctx *gin.Context) {
	minStr := ctx.Query("min")
	maxStr := ctx.Query("max")
	min, _ := strconv.Atoi(minStr)
	max, _ := strconv.Atoi(maxStr)

	orders, err := h.Repository.GetPublishedFiltered(min, max)
	if err != nil {
		logrus.WithError(err).Error("плитка: ошибка выборки")
	}

	type GridCard struct {
		repository.BusinessServicesOrder
		Likes int
	}
	cards := make([]GridCard, 0, len(orders))
	for _, o := range orders {
		cards = append(cards, GridCard{BusinessServicesOrder: o, Likes: o.LikesCount()})
	}

	ctx.HTML(http.StatusOK, "business_services_grid.html", gin.H{
		"title":  "Все заявки",
		"active": "grid",
		"cards":  cards,
		"min":    minStr,
		"max":    maxStr,
	})
}

// ---------- GET /business_services_media/*key ----------
func (h *Handler) Media(ctx *gin.Context) {
	key := strings.TrimPrefix(ctx.Param("key"), "/")
	if key == "" {
		ctx.Status(http.StatusNotFound)
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
		logrus.WithError(err).WithField("key", key).Warn("media not found in MinIO, fallback to local")
		h.mediaFromLocal(ctx, key)
		return
	}

	obj, err := h.Storage.Get(bg, key)
	if err != nil {
		logrus.WithError(err).Error("media get from MinIO, fallback to local")
		h.mediaFromLocal(ctx, key)
		return
	}
	defer obj.Close()

	ctx.Header("Content-Type", info.ContentType)
	ctx.Header("Content-Length", strconv.FormatInt(info.Size, 10))
	ctx.Header("Cache-Control", "public, max-age=86400")
	ctx.Header("ETag", info.ETag)

	ctx.Status(http.StatusOK)
	if _, err := io.Copy(ctx.Writer, obj); err != nil {
		logrus.WithError(err).Error("media stream from MinIO")
	}
}

func (h *Handler) mediaFromLocal(ctx *gin.Context, key string) {
	name := filepath.Base(key)
	if name == "" || name == "." || name == "/" {
		ctx.Status(http.StatusNotFound)
		return
	}

	localPath := filepath.Join("resources", "seed", name)

	info, err := os.Stat(localPath)
	if err != nil || info.IsDir() {
		logrus.WithError(err).WithField("path", localPath).Warn("media not found locally")
		ctx.Status(http.StatusNotFound)
		return
	}

	f, err := os.Open(localPath)
	if err != nil {
		logrus.WithError(err).Error("open local media")
		ctx.Status(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	ctx.Header("Content-Type", contentTypeByExt(filepath.Ext(name)))
	ctx.Header("Content-Length", strconv.FormatInt(info.Size(), 10))
	ctx.Header("Cache-Control", "public, max-age=3600")

	ctx.Status(http.StatusOK)
	if _, err := io.Copy(ctx.Writer, f); err != nil {
		logrus.WithError(err).Error("media stream from local")
	}
}

func contentTypeByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	}
	return "application/octet-stream"
}
