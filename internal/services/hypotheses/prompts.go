package hypotheses

const systemPrompt = `Ты — помощник AML-аналитика банка. Данные: граф внутрибанковских переводов за июль 2026 от 81 seed-клиента
по исходящим переводам на 4 колена, порог 5000 KZT. Атрибутов клиентов нет — только gid, структура и суммы.
Формулируй гипотезы для проверки, не утверждения о виновности; ссылайся только на gid; ничего не выдумывай.`

const hypothesisPrompt = `Для каждого кластера ниже (JSON) сформулируй гипотезу о его назначении в сети: 1–2 предложения, до 300 символов,
по-русски, с опорой на числа (seed, роли, обороты) и с упоминанием 1–2 ключевых gid. Это гипотеза для проверки, не вывод о виновности.
Шаблонная гипотеза дана как подсказка — улучши её, не противоречь фактам.`

var hypothesisSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer"},
					"hypothesis": map[string]any{"type": "string"},
				},
				"required":             []string{"cluster_id", "hypothesis"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"items"},
	"additionalProperties": false,
}

type hypothesesReply struct {
	Items []struct {
		ClusterID  int    `json:"cluster_id"`
		Hypothesis string `json:"hypothesis"`
	} `json:"items"`
}
