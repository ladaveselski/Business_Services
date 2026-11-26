package handler

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"business_services/internal/app/repository"
	"business_services/internal/auth"
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
// HTML-страницы
// =====================================================================

// GET /business_services_feed
// GET /business_services_feed/:id?next=true
func (h *Handler) Feed(ctx *gin.Context) {
	idStr := ctx.Param("id")
	next := ctx.Query("next") == "true"

	var svc repository.Service
	var err error

	if idStr == "" {
		orders, e := h.Repository.GetPublished()
		if e != nil || len(orders) == 0 {
			ctx.Status(http.StatusNotFound)
			return
		}
		svc = orders[0]
	} else {
		id, e := strconv.Atoi(idStr)
		if e != nil {
			ctx.Status(http.StatusBadRequest)
			return
		}
		if next {
			svc, err = h.Repository.GetNextPublished(id)
		} else {
			svc, err = h.Repository.GetPublishedByID(id)
		}
		if err != nil {
			logrus.WithError(err).Warn("лента: услуга не найдена")
			ctx.Status(http.StatusNotFound)
			return
		}
	}

	ctx.HTML(http.StatusOK, "business_services_feed.html", gin.H{
		"title":     svc.Title,
		"active":    "feed",
		"order":     svc,
		"likes":     svc.LikesCount(),
		"image_url": svc.ImageURL(),
		"video_url": svc.VideoURL(),
		"avg_time":  svc.AvgTimeStr(),
	})
}

// GET /business_services_add
func (h *Handler) AddPage(ctx *gin.Context) {
	draft, err := h.Repository.GetDraft(auth.CurrentUserID())
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		logrus.WithError(err).Warn("черновик: ошибка")
	}

	ctx.HTML(http.StatusOK, "business_services_add.html", gin.H{
		"title":     "Новая заявка",
		"active":    "add",
		"order":     draft,
		"image_url": draft.ImageURL(),
		"video_url": draft.VideoURL(),
	})
}

// GET /business_services_grid?min=&max=
func (h *Handler) Grid(ctx *gin.Context) {
	minStr := ctx.Query("min")
	maxStr := ctx.Query("max")
	min, _ := strconv.Atoi(minStr)
	max, _ := strconv.Atoi(maxStr)

	orders, err := h.Repository.GetPublishedFiltered(min, max)
	if err != nil {
		logrus.WithError(err).Error("плитка: ошибка выборки")
	}

	liked, _ := h.Repository.LikedIDs(auth.CurrentUserID())

	type GridCard struct {
		repository.Service
		Likes   int
		IsLiked bool
	}
	cards := make([]GridCard, 0, len(orders))
	for _, o := range orders {
		cards = append(cards, GridCard{
			Service: o,
			Likes:   o.LikesCount(),
			IsLiked: liked[o.ID],
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
// Media
// =====================================================================

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
		logrus.WithError(err).WithField("key", key).Warn("media: not found in MinIO, fallback")
		h.mediaFromLocal(ctx, key)
		return
	}
	obj, err := h.Storage.Get(bg, key)
	if err != nil {
		logrus.WithError(err).Error("media: get from MinIO, fallback")
		h.mediaFromLocal(ctx, key)
		return
	}
	defer obj.Close()

	ctx.Header("Content-Type", info.ContentType)
	ctx.Header("Content-Length", strconv.FormatInt(info.Size, 10))
	ctx.Header("Cache-Control", "public, max-age=86400")
	if info.ETag != "" {
		ctx.Header("ETag", info.ETag)
	}
	ctx.Status(http.StatusOK)
	if _, err := io.Copy(ctx.Writer, obj); err != nil {
		logrus.WithError(err).Error("media: stream MinIO")
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
		logrus.WithError(err).WithField("path", localPath).Warn("media: not found locally")
		ctx.Status(http.StatusNotFound)
		return
	}
	f, err := os.Open(localPath)
	if err != nil {
		logrus.WithError(err).Error("media: open local")
		ctx.Status(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	ctx.Header("Content-Type", contentTypeByExt(filepath.Ext(name)))
	ctx.Header("Content-Length", strconv.FormatInt(info.Size(), 10))
	ctx.Header("Cache-Control", "public, max-age=3600")
	ctx.Status(http.StatusOK)
	if _, err := io.Copy(ctx.Writer, f); err != nil {
		logrus.WithError(err).Error("media: stream local")
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

// =====================================================================
// API (JSON) — домен услуг
// =====================================================================

// GET /api/business_services?min=&max=
func (h *Handler) APIList(ctx *gin.Context) {
	min, _ := strconv.Atoi(ctx.Query("min"))
	max, _ := strconv.Atoi(ctx.Query("max"))

	orders, err := h.Repository.GetPublishedFiltered(min, max)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	uid := auth.CurrentUserID()
	liked, _ := h.Repository.LikedIDs(uid)

	res := make([]ServiceResponse, 0, len(orders))
	for _, o := range orders {
		res = append(res, toResponse(o, uid, liked[o.ID]))
	}
	ctx.JSON(http.StatusOK, gin.H{"items": res, "count": len(res)})
}

// GET /api/business_services/feed
// GET /api/business_services/feed/{id}?next=true
func (h *Handler) APIFeed(ctx *gin.Context) {
	idStr := ctx.Param("id")
	next := ctx.Query("next") == "true"

	var svc repository.Service
	var err error

	if idStr == "" {
		orders, e := h.Repository.GetPublished()
		if e != nil || len(orders) == 0 {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "лента пуста"})
			return
		}
		svc = orders[0]
	} else {
		id, e := strconv.Atoi(idStr)
		if e != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "некорректный id"})
			return
		}
		if next {
			svc, err = h.Repository.GetNextPublished(id)
		} else {
			svc, err = h.Repository.GetPublishedByID(id)
		}
		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "услуга не найдена"})
			return
		}
	}

	uid := auth.CurrentUserID()
	liked, _ := h.Repository.LikedIDs(uid)
	ctx.JSON(http.StatusOK, toResponse(svc, uid, liked[svc.ID]))
}

// GET /api/business_services/draft
func (h *Handler) APIDraft(ctx *gin.Context) {
	uid := auth.CurrentUserID()
	svc, err := h.Repository.GetDraft(uid)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "черновик не найден"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, toResponse(svc, uid, false))
}

// POST /api/business_services (multipart)
// Принимает ТОЛЬКО price, image, video.
// Логика: обновляет черновик пользователя, если он есть; иначе создаёт новый.
func (h *Handler) APICreate(ctx *gin.Context) {
	var in CreateServiceInput
	if err := ctx.ShouldBind(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid := auth.CurrentUserID()

	// Ищем существующий черновик
	draft, err := h.Repository.GetDraft(uid)
	isNew := false
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			isNew = true
			draft = repository.Service{
				Status:      repository.StatusDraft,
				Title:       "Новая услуга",
				Description: "",
				AvgTimeMin:  0,
				CreatorID:   uid,
			}
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Файлы — опциональны. Если пришли — загружаем и перезаписываем ключ.
	if fh := mustFile(ctx, "image"); fh != nil {
		key, err := h.saveUpload(ctx.Request.Context(), fh, "images")
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "image: " + err.Error()})
			return
		}
		draft.ImageKey = key
	}
	if fh := mustFile(ctx, "video"); fh != nil {
		key, err := h.saveUpload(ctx.Request.Context(), fh, "videos")
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "video: " + err.Error()})
			return
		}
		draft.VideoKey = key
	}

	draft.Price = in.Price

	if isNew {
		if err := h.Repository.Create(&draft); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusCreated, toResponse(draft, uid, false))
		return
	}

	// Обновляем существующий черновик
	if err := h.Repository.UpdateDraft(&draft); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, toResponse(draft, uid, false))
}

// PUT /api/business_services/{id}/publish
func (h *Handler) APIPublish(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "некорректный id"})
		return
	}
	if err := h.Repository.Publish(id, auth.CurrentUserID()); err != nil {
		ctx.JSON(statusFromRepoErr(err), gin.H{"error": err.Error()})
		return
	}
	svc, _ := h.Repository.GetPublishedByID(id)
	ctx.JSON(http.StatusOK, toResponse(svc, auth.CurrentUserID(), false))
}

// DELETE /api/business_services/{id}
func (h *Handler) APIDelete(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "некорректный id"})
		return
	}
	if err := h.Repository.SoftDelete(id, auth.CurrentUserID()); err != nil {
		ctx.JSON(statusFromRepoErr(err), gin.H{"error": err.Error()})
		return
	}
	ctx.Status(http.StatusNoContent)
}

// POST /api/business_services/{id}/like
func (h *Handler) APILike(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "некорректный id"})
		return
	}
	var in LikeInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Repository.SetLike(id, auth.CurrentUserID(), in.Value); err != nil {
		ctx.JSON(statusFromRepoErr(err), gin.H{"error": err.Error()})
		return
	}
	svc, _ := h.Repository.GetPublishedByID(id)
	ctx.JSON(http.StatusOK, toResponse(svc, auth.CurrentUserID(), in.Value == 1))
}

// =====================================================================
// API (заглушки домена пользователя)
// =====================================================================

func (h *Handler) APIRegister(ctx *gin.Context) {
	ctx.JSON(http.StatusNotImplemented, gin.H{"error": "заглушка: регистрация в 4-й лабе"})
}

func (h *Handler) APILogin(ctx *gin.Context) {
	ctx.JSON(http.StatusNotImplemented, gin.H{"error": "заглушка: аутентификация в 4-й лабе"})
}

func (h *Handler) APILogout(ctx *gin.Context) {
	ctx.JSON(http.StatusNotImplemented, gin.H{"error": "заглушка: деавторизация в 4-й лабе"})
}

// =====================================================================
// Хелперы
// =====================================================================

func toResponse(s repository.Service, uid int, isLiked bool) ServiceResponse {
	return ServiceResponse{
		ID:          s.ID,
		Status:      string(s.Status),
		Title:       s.Title,
		Description: s.Description,
		Price:       s.Price,
		AvgTimeMin:  s.AvgTimeMin,
		AvgTimeStr:  s.AvgTimeStr(),
		ImageKey:    s.ImageKey,
		VideoKey:    s.VideoKey,
		ImageURL:    s.ImageURL(),
		VideoURL:    s.VideoURL(),
		LikesCount:  s.LikesCount(),
		CreatorID:   s.CreatorID,
		IsOwner:     s.CreatorID == uid,
		IsLiked:     isLiked,
	}
}

func mustFile(ctx *gin.Context, name string) *multipart.FileHeader {
	fh, _ := ctx.FormFile(name)
	return fh
}

func statusFromRepoErr(err error) int {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, repository.ErrNotOwner):
		return http.StatusForbidden
	case errors.Is(err, repository.ErrWrongStatus):
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

func (h *Handler) saveLocal(key string, r io.Reader) error {
	dir := filepath.Join("resources", "seed")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(dir, filepath.Base(key))
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}
