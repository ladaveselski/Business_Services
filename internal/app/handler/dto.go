package handler

// CreateServiceInput — данные для POST /api/business_services.
// Клиент присылает ТОЛЬКО цену и файлы (image, video).
// Всё остальное (title, description, avg_time_min, status, creator_id) — на бэкенде.
type CreateServiceInput struct {
	Price int `form:"price" binding:"gte=0"`
}

// LikeInput — тело POST /api/business_services/{id}/like.
type LikeInput struct {
	Value int `json:"value" binding:"oneof=0 1"`
}

// ServiceResponse — то, что уходит клиенту.
type ServiceResponse struct {
	ID          int    `json:"id"`
	Status      string `json:"status"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       int    `json:"price"`
	AvgTimeMin  int    `json:"avg_time_min"`
	AvgTimeStr  string `json:"avg_time_str"`
	ImageKey    string `json:"image_key"`
	VideoKey    string `json:"video_key"`
	ImageURL    string `json:"image_url"`
	VideoURL    string `json:"video_url"`
	LikesCount  int    `json:"likes_count"`

	CreatorID int  `json:"creator_id"`
	IsOwner   bool `json:"is_owner"`
	IsLiked   bool `json:"is_liked"`
}
