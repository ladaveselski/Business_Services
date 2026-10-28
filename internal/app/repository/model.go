package repository

import "time"

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusDeleted   Status = "deleted"
)

// BusinessServicesOrder — одна услуга (заявка).
type BusinessServicesOrder struct {
	ID          int    `json:"id"`
	Status      Status `json:"status"`
	Title       string `json:"title"`
	Description string `json:"description"`

	Price    int       `json:"price"`
	Deadline time.Time `json:"deadline"`

	ImageKey string `json:"image_key"`
	VideoKey string `json:"video_key"`

	Likes []int `json:"likes"`
}

func (o BusinessServicesOrder) LikesCount() int {
	return len(o.Likes)
}

func (o BusinessServicesOrder) ImageURL() string {
	if o.ImageKey == "" {
		return "/static/img/placeholder.png"
	}
	return "/business_services_media/" + o.ImageKey
}

func (o BusinessServicesOrder) VideoURL() string {
	if o.VideoKey == "" {
		return ""
	}
	return "/business_services_media/" + o.VideoKey
}

func (o BusinessServicesOrder) DeadlineStr() string {
	if o.Deadline.IsZero() {
		return "—"
	}
	return o.Deadline.Format("02.01.2006 15:04")
}
