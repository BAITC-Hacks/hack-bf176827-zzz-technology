package analysis

import "fmt"

// formatKZT — сумма для текста: «2.2 млн», «555 тыс», «900».
func formatKZT(value float64) string {
	switch {
	case value >= 1e6:
		return fmt.Sprintf("%.1f млн", value/1e6)
	case value >= 1e3:
		return fmt.Sprintf("%.0f тыс", value/1e3)
	default:
		return fmt.Sprintf("%.0f", value)
	}
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit-1]) + "…"
}
