// Package dto — внешние JSON-модели API (то, что видит swagger и фронт).
package dto

type ListResponse[T any] struct {
	Items []T `json:"items"`
}

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}
