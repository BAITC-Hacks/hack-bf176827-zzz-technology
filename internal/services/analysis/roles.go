package analysis

import (
	"fmt"
	"math"
	"sort"

	"hackaton/internal/data/models"
)

// Пороги ролей. Обоснование — docs/methodology.md, раздел 3.
const (
	consolidatorMinPayers  = 3   // плательщиков ≥
	consolidatorFullPayers = 6.0 // плательщиков, при которых скор = 1
	seedPayersBonusFrom    = 2   // бонус +0.15, если платят ≥ 2 seed
	seedPayersBonus        = 0.15

	distributorMinPayees  = 5    // получателей ≥ (и ≥ 2·плательщиков)
	distributorFullPayees = 30.0 // получателей, при которых скор = 1
	twoSidedHubPenalty    = 0.6  // веер при ≥ 3 плательщиках ближе к координатору

	transitPassThroughLow  = 0.7
	transitPassThroughHigh = 1.3
	transitFastForwardBon  = 0.2

	terminalMinPayers  = 2
	terminalMinKZT     = 100_000.0
	terminalFullKZT    = 500_000.0
	terminalFullPayers = 3.0

	coordinatorMinDegree     = 2
	coordinatorMinSeedUp     = 3 // seed выше по цепочке ≥ (или кластеров на входе ≥ 2)
	coordinatorMinClustersIn = 2
	coordinatorBetwPercentil = 0.97

	truncatedRetention = 0.7 // «удержание» для обрезанных узлов: исходящие неизвестны
	evidenceMaxRunes   = 200
)

// порядок при равенстве скоров
var rolePrecedence = []models.Role{
	models.RoleCoordinator, models.RoleConsolidator, models.RoleDistributor,
	models.RoleTransit, models.RoleTerminal, models.RolePeripheral,
}

type betweennessThresholds struct{ high, top float64 } // P97 и P99

func assignRoles(result *models.AnalysisResult) {
	thresholds := computeBetweennessThresholds(result)
	for i := range result.Nodes {
		node := &result.Nodes[i]
		scores := roleScores(node.Features, thresholds)
		best, bestScore := models.RolePeripheral, -1.0
		for _, role := range rolePrecedence {
			if scores[role] > bestScore {
				best, bestScore = role, scores[role]
			}
		}
		node.Role = best
		node.RoleScore = math.Round(bestScore*100) / 100
		node.Evidence = truncateRunes(buildEvidence(node), evidenceMaxRunes)
	}
}

// roleScores — скор 0..1 для каждой роли; отсутствие ключа = правило не сработало.
func roleScores(f models.Features, th betweennessThresholds) map[models.Role]float64 {
	scores := map[models.Role]float64{}

	// consolidator: много плательщиков, деньги в основном остаются
	if f.InDegree >= consolidatorMinPayers {
		retention := truncatedRetention
		if !f.Truncated {
			retention = 1 - clamp01(f.PassThrough)
		}
		score := clamp01(float64(f.InDegree)/consolidatorFullPayers) * retention
		if f.SeedPayers >= seedPayersBonusFrom {
			score += seedPayersBonus
		}
		scores[models.RoleConsolidator] = clamp01(score)
	}

	// distributor: веер на много получателей
	if f.OutDegree >= distributorMinPayees && f.OutDegree >= 2*f.InDegree {
		score := clamp01(float64(f.OutDegree) / distributorFullPayees)
		if f.InDegree >= consolidatorMinPayers {
			score *= twoSidedHubPenalty
		}
		scores[models.RoleDistributor] = score
	}

	// transit: пришло ≈ ушло; seed исключены — их входящие занижены выгрузкой
	if !f.IsSeed && f.InDegree >= 1 && f.OutDegree >= 1 &&
		f.PassThrough >= transitPassThroughLow && f.PassThrough <= transitPassThroughHigh {
		score := 0.5 + 0.5*(1-math.Abs(f.PassThrough-1)/(transitPassThroughHigh-1))
		score += transitFastForwardBon * f.FastForwardShare
		scores[models.RoleTransit] = clamp01(score)
	}

	// coordinator: и вход, и выход, высокое посредничество, связывает кластеры или несколько seed
	if f.InDegree >= coordinatorMinDegree && f.OutDegree >= coordinatorMinDegree && f.Betweenness >= th.high &&
		(f.ClustersIn >= coordinatorMinClustersIn || f.SeedUpstream >= coordinatorMinSeedUp) {
		mix := 0.5*clamp01(f.Betweenness/th.top) + 0.25*clamp01(float64(f.ClustersIn)/3) + 0.25*clamp01(float64(f.SeedUpstream)/5)
		scores[models.RoleCoordinator] = 0.5 + 0.5*mix
	}

	// terminal: подтверждённый сток; обрезанные обходом сюда не попадают
	if f.VerifiedSink && (f.InDegree >= terminalMinPayers || f.InKZT >= terminalMinKZT) {
		scores[models.RoleTerminal] = 0.5*clamp01(f.InKZT/terminalFullKZT) + 0.5*clamp01(float64(f.InDegree)/terminalFullPayers)
	}

	maxOther := 0.0
	for _, score := range scores {
		maxOther = math.Max(maxOther, score)
	}
	scores[models.RolePeripheral] = 1 - maxOther
	return scores
}

func buildEvidence(node *models.NodeResult) string {
	f := node.Features
	passThrough := "н/д"
	if f.PassThrough >= 0 {
		passThrough = fmt.Sprintf("%.0f%%", f.PassThrough*100)
	}
	truncatedNote := ""
	if f.Truncated {
		truncatedNote = "; исходящие не видны (4-е колено)"
	}

	var text string
	switch node.Role {
	case models.RoleConsolidator:
		seedNote := ""
		if f.SeedPayers > 0 {
			seedNote = fmt.Sprintf(" (из них seed: %d)", f.SeedPayers)
		}
		text = fmt.Sprintf("консолидация: получает от %d плательщиков%s %s KZT, отдаёт дальше %s (%d получ.)%s",
			f.InDegree, seedNote, formatKZT(f.InKZT), passThrough, f.OutDegree, truncatedNote)
	case models.RoleDistributor:
		text = fmt.Sprintf("веер: %d получателей, %d переводов на %s KZT; входов %d на %s KZT",
			f.OutDegree, f.OutTxCount, formatKZT(f.OutKZT), f.InDegree, formatKZT(f.InKZT))
	case models.RoleTransit:
		text = fmt.Sprintf("транзит: получил %s KZT от %d, отдал %s KZT на %d (пропуск %s); в течение 2 дней ушло %.0f%%",
			formatKZT(f.InKZT), f.InDegree, formatKZT(f.OutKZT), f.OutDegree, passThrough, f.FastForwardShare*100)
	case models.RoleCoordinator:
		text = fmt.Sprintf("координация: вход %d/%s KZT, выход %d/%s KZT; посредничество top-3%% (%.0f), кластеров на входе %d, seed выше по цепочке %d",
			f.InDegree, formatKZT(f.InKZT), f.OutDegree, formatKZT(f.OutKZT), f.Betweenness, f.ClustersIn, f.SeedUpstream)
	case models.RoleTerminal:
		text = fmt.Sprintf("конечный получатель: получил %s KZT от %d (%d перев.), исходящих ≥5000 нет, хотя обход шёл дальше (колено %d)",
			formatKZT(f.InKZT), f.InDegree, f.InTxCount, f.Depth)
	default:
		switch {
		case f.InDegree == 0 && f.OutDegree == 0 && f.IsSeed:
			text = "нет переводов ≥5000 KZT в июле; seed только по оперативной информации"
		case f.InDegree == 0 && f.OutDegree == 0:
			text = "нет переводов ≥5000 KZT в июле"
		case f.Truncated:
			text = fmt.Sprintf("получил %s KZT от %d%s; признаков роли нет", formatKZT(f.InKZT), f.InDegree, truncatedNote)
		default:
			text = fmt.Sprintf("вход %d/%s KZT, выход %d/%s KZT (пропуск %s): признаков роли нет",
				f.InDegree, formatKZT(f.InKZT), f.OutDegree, formatKZT(f.OutKZT), passThrough)
		}
	}
	if f.InCycle && node.Role != models.RolePeripheral {
		text += "; в возвратном цикле"
	}
	if f.RepeatRoutes > 0 && (node.Role == models.RoleTransit || node.Role == models.RoleCoordinator) {
		text += fmt.Sprintf("; устойчивых маршрутов через узел: %d", f.RepeatRoutes)
	}
	return text
}

func computeBetweennessThresholds(result *models.AnalysisResult) betweennessThresholds {
	values := make([]float64, len(result.Nodes))
	for i, node := range result.Nodes {
		values[i] = node.Features.Betweenness
	}
	sort.Float64s(values)
	quantile := func(p float64) float64 {
		return values[int(math.Min(float64(len(values)-1), p*float64(len(values))))]
	}
	th := betweennessThresholds{high: quantile(coordinatorBetwPercentil), top: quantile(0.99)}
	if th.top <= 0 {
		th.top = 1
	}
	return th
}

func clamp01(value float64) float64 {
	if math.IsNaN(value) {
		return 0
	}
	return math.Max(0, math.Min(1, value))
}
