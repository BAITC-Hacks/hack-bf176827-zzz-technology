package dto

import (
	"time"

	"hackaton/internal/repo/db"

	"github.com/google/uuid"
)

type Item struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateItemRequest struct {
	Title string `json:"title" validate:"required,min=1,max=200" example:"Первый элемент"`
}

func ItemFromDB(m db.Item) Item {
	return Item{ID: m.ID, Title: m.Title, CreatedAt: m.CreatedAt}
}

func ItemsFromDB(ms []db.Item) []Item {
	res := make([]Item, 0, len(ms))
	for _, m := range ms {
		res = append(res, ItemFromDB(m))
	}
	return res
}
