// Package dto — общие JSON-модели API. Доменные DTO лежат в подпакетах dto/graph и dto/assistant;
// gid в ответах всегда строкой (значения ≈ 1e17 не помещаются в JS Number).
package dto

type ListResponse[T any] struct {
	Items []T `json:"items"`
}

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}
