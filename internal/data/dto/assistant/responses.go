package assistantdto

type StatusResponse struct {
	Enabled bool   `json:"enabled"`
	Model   string `json:"model"`
}

type AskResponse struct {
	Answer string   `json:"answer"`
	Gids   []string `json:"gids"` // упомянутые gid — для подсветки на графе
	Steps  int      `json:"steps"`
	Cached bool     `json:"cached"`
}

type CardResponse struct {
	Gid   string `json:"gid"`
	Text  string `json:"text"`
	ByLLM bool   `json:"by_llm"` // false — шаблон (нет ключа или LLM недоступен)
}
