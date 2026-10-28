package repository

import (
	"fmt"
	"sort"
	"time"
)

type Repository struct {
	orders []BusinessServicesOrder
}

func NewRepository() (*Repository, error) {
	r := &Repository{}
	r.seed()
	return r, nil
}

func (r *Repository) seed() {
	r.orders = []BusinessServicesOrder{
		{
			ID:          1,
			Status:      StatusPublished,
			Title:       "Сантехника",
			Description: "Замена смесителя в ванной комнате. Быстрая диагностика и ремонт.",
			Price:       4500,
			Deadline:    time.Date(2026, 10, 5, 14, 30, 0, 0, time.UTC),
			ImageKey:    "seed/plumber.jpg",
			VideoKey:    "seed/plumber.mp4",
			Likes:       []int{101, 102, 103, 104, 105, 106, 107, 108, 109, 110},
		},
		{
			ID:          2,
			Status:      StatusPublished,
			Title:       "Электрика",
			Description: "Профессиональная диагностика для профилактики проблем.",
			Price:       5900,
			Deadline:    time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC),
			ImageKey:    "seed/electrician.jpg",
			VideoKey:    "seed/electrician.mp4",
			Likes:       []int{201, 202, 203, 204, 205},
		},
		{
			ID:          3,
			Status:      StatusPublished,
			Title:       "Ремонт",
			Description: "Замена смесителя в ванной комнате. Быстрая диагностика и ремонт.",
			Price:       5900,
			Deadline:    time.Date(2026, 10, 7, 9, 15, 0, 0, time.UTC),
			ImageKey:    "seed/repair.jpg",
			VideoKey:    "seed/repair.mp4",
			Likes:       []int{301, 302, 303},
		},
		{
			ID:          4,
			Status:      StatusPublished,
			Title:       "Проверка мебели",
			Description: "Контроль сохранности имущества.",
			Price:       3200,
			Deadline:    time.Date(2026, 10, 8, 12, 45, 0, 0, time.UTC),
			ImageKey:    "seed/furniture.jpg",
			VideoKey:    "seed/furniture.mp4",
			Likes:       []int{401},
		},
		{
			ID:          99,
			Status:      StatusDraft,
			Title:       "Новая услуга",
			Description: "Описание проблемы...",
			Price:       0,
			Deadline:    time.Time{},
			ImageKey:    "seed/plumber.jpg", // ← фото по умолчанию
			VideoKey:    "seed/plumber.mp4", // ← видео по умолчанию
			Likes:       []int{},
		},
		{
			ID:          100,
			Status:      StatusDeleted,
			Title:       "Удалённая услуга",
			Description: "Не отображается",
			Price:       0,
			Deadline:    time.Time{},
			ImageKey:    "",
			VideoKey:    "",
			Likes:       []int{},
		},
	}
}

func (r *Repository) GetPublished() ([]BusinessServicesOrder, error) {
	var res []BusinessServicesOrder
	for _, o := range r.orders {
		if o.Status == StatusPublished {
			res = append(res, o)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res, nil
}

func (r *Repository) GetPublishedByID(id int) (BusinessServicesOrder, error) {
	for _, o := range r.orders {
		if o.ID == id && o.Status == StatusPublished {
			return o, nil
		}
	}
	return BusinessServicesOrder{}, fmt.Errorf("услуга %d не найдена", id)
}

func (r *Repository) GetNextPublished(id int) (BusinessServicesOrder, error) {
	orders, _ := r.GetPublished()
	for i, o := range orders {
		if o.ID == id {
			if i+1 < len(orders) {
				return orders[i+1], nil
			}
			return orders[0], nil
		}
	}
	if len(orders) > 0 {
		return orders[0], nil
	}
	return BusinessServicesOrder{}, fmt.Errorf("нет опубликованных услуг")
}

func (r *Repository) GetDraft() (BusinessServicesOrder, error) {
	for _, o := range r.orders {
		if o.Status == StatusDraft {
			return o, nil
		}
	}
	return BusinessServicesOrder{}, fmt.Errorf("черновик не найден")
}

func (r *Repository) GetPublishedFiltered(minPrice, maxPrice int) ([]BusinessServicesOrder, error) {
	all, _ := r.GetPublished()
	var res []BusinessServicesOrder
	for _, o := range all {
		if (minPrice == 0 || o.Price >= minPrice) && (maxPrice == 0 || o.Price <= maxPrice) {
			res = append(res, o)
		}
	}
	return res, nil
}
