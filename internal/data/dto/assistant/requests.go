// Package assistantdto — запросы и ответы LLM-ассистента.
package assistantdto

type AskRequest struct {
	Question string `json:"question" validate:"required,max=2000"`
}
