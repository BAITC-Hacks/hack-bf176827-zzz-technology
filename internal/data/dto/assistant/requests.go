// Package assistantdto — запросы и ответы LLM-ассистента.
package assistantdto

type AskRequest struct {
	Question string   `json:"question" validate:"required,max=2000"`
	Gid      string   `json:"gid" validate:"omitempty,numeric,max=19"`               // выбранный узел — контекст вопроса
	Route    []string `json:"route" validate:"omitempty,max=40,dive,numeric,max=19"` // путь просмотра аналитика, только контекст
}
