package analysis

import (
	"fmt"
	"math"
	"sort"

	"hackaton/internal/graph"
)

// Пороги ролей (обоснование — .agent/plan.md и README).
const (
	consMinIn      = 3    // consolidator: плательщиков ≥
	consFullIn     = 6.0  // consolidator: in_deg, при котором скор = 1
	distMinOut     = 5    // distributor: получателей ≥
	distFullOut    = 30.0 // distributor: out_deg, при котором скор = 1
	transitLo      = 0.7  // transit: pass_through в [lo, hi]
	transitHi      = 1.3
	termMinIn      = 2      // terminal: плательщиков ≥ или сумма ≥
	termMinKZT     = 100000 //
	termFullKZT    = 5e5
	coordMinDeg    = 2 // coordinator: in_deg и out_deg ≥
	coordSeedUp    = 3 // coordinator: seed выше по цепочке ≥ (или кластеров на входе ≥ 2)
	coordBtwPct    = 0.97
	truncatedShare = 0.7 // множитель скора, если исходящие обрезаны обходом
)

// порядок при равенстве скоров
var rolePriority = []string{RoleCoordinator, RoleConsolidator, RoleDistributor, RoleTransit, RoleTerminal, RolePeripheral}

type thresholds struct{ btwP95, btwP99 float64 }

func assignRoles(g *graph.Graph, res *Result) {
	th := btwThresholds(res)
	for i := range res.Nodes {
		n := &res.Nodes[i]
		scores := roleScores(n.Features, th)
		best, bestScore := RolePeripheral, -1.0
		for _, r := range rolePriority {
			if scores[r] > bestScore {
				best, bestScore = r, scores[r]
			}
		}
		n.Role, n.RoleScore = best, round2(bestScore)
		ev := evidence(n, th)
		if n.Features.InCycle && n.Role != RolePeripheral {
			ev += "; в возвратном цикле"
		}
		if n.Features.RepeatRoutes > 0 && (n.Role == RoleTransit || n.Role == RoleCoordinator) {
			ev += fmt.Sprintf("; устойчивых маршрутов через узел: %d", n.Features.RepeatRoutes)
		}
		n.Evidence = truncate(ev, 200)
	}
}

// roleScores — скор 0..1 для каждой роли; 0 = правило не сработало.
func roleScores(f Features, th thresholds) map[string]float64 {
	s := map[string]float64{}

	// consolidator: много плательщиков, деньги в основном остаются
	if f.InDeg >= consMinIn {
		retention := truncatedShare // исходящие не видны — считаем неизвестными
		if !f.Truncated {
			retention = 1 - clamp(f.PassThrough, 0, 1)
		}
		v := clamp(float64(f.InDeg)/consFullIn, 0, 1) * retention
		if f.NSeedPayers >= 2 {
			v += 0.15
		}
		s[RoleConsolidator] = clamp(v, 0, 1)
	}

	// distributor: веер на много получателей; двусторонний хаб штрафуется (это ближе к координатору)
	if f.OutDeg >= distMinOut && f.OutDeg >= 2*f.InDeg {
		v := clamp(float64(f.OutDeg)/distFullOut, 0, 1)
		if f.InDeg >= consMinIn {
			v *= 0.6
		}
		s[RoleDistributor] = v
	}

	// transit: пришло ≈ ушло; seed исключены (их входящие занижены выгрузкой)
	if !f.IsSeed && f.InDeg >= 1 && f.OutDeg >= 1 && f.PassThrough >= transitLo && f.PassThrough <= transitHi {
		v := 0.5 + 0.5*(1-math.Abs(f.PassThrough-1)/(transitHi-1))
		v += 0.2 * f.FastForwardShare
		s[RoleTransit] = clamp(v, 0, 1)
	}

	// coordinator: и вход, и выход, высокое посредничество, связывает кластеры / несколько seed
	if f.InDeg >= coordMinDeg && f.OutDeg >= coordMinDeg && f.Betweenness >= th.btwP95 &&
		(f.NClustersIn >= 2 || f.NSeedUpstream >= coordSeedUp) {
		v := 0.5*clamp(f.Betweenness/th.btwP99, 0, 1) + 0.25*clamp(float64(f.NClustersIn)/3, 0, 1) + 0.25*clamp(float64(f.NSeedUpstream)/5, 0, 1)
		s[RoleCoordinator] = 0.5 + 0.5*v
	}

	// terminal: подтверждённый сток (обход шёл дальше и ничего не нашёл), обрезанные сюда не попадают
	if f.VerifiedSink && (f.InDeg >= termMinIn || f.InKZT >= termMinKZT) {
		s[RoleTerminal] = 0.5*clamp(f.InKZT/termFullKZT, 0, 1) + 0.5*clamp(float64(f.InDeg)/3, 0, 1)
	}

	maxOther := 0.0
	for _, v := range s {
		maxOther = math.Max(maxOther, v)
	}
	s[RolePeripheral] = 1 - maxOther
	return s
}

func evidence(n *NodeResult, th thresholds) string {
	f := n.Features
	pt := "н/д"
	if f.PassThrough >= 0 {
		pt = fmt.Sprintf("%.0f%%", f.PassThrough*100)
	}
	trunc := ""
	if f.Truncated {
		trunc = "; исходящие не видны (4-е колено)"
	}
	switch n.Role {
	case RoleConsolidator:
		seed := ""
		if f.NSeedPayers > 0 {
			seed = fmt.Sprintf(" (из них seed: %d)", f.NSeedPayers)
		}
		return fmt.Sprintf("консолидация: получает от %d плательщиков%s %s KZT, отдаёт дальше %s (%d получ.)%s",
			f.InDeg, seed, kzt(f.InKZT), pt, f.OutDeg, trunc)
	case RoleDistributor:
		return fmt.Sprintf("веер: %d получателей, %d переводов на %s KZT; входов %d на %s KZT",
			f.OutDeg, f.OutTx, kzt(f.OutKZT), f.InDeg, kzt(f.InKZT))
	case RoleTransit:
		return fmt.Sprintf("транзит: получил %s KZT от %d, отдал %s KZT на %d (пропуск %s); в течение 2 дней ушло %.0f%%",
			kzt(f.InKZT), f.InDeg, kzt(f.OutKZT), f.OutDeg, pt, f.FastForwardShare*100)
	case RoleCoordinator:
		return fmt.Sprintf("координация: вход %d/%s KZT, выход %d/%s KZT; посредничество top-3%% (%.0f), кластеров на входе %d, seed выше по цепочке %d",
			f.InDeg, kzt(f.InKZT), f.OutDeg, kzt(f.OutKZT), f.Betweenness, f.NClustersIn, f.NSeedUpstream)
	case RoleTerminal:
		return fmt.Sprintf("конечный получатель: получил %s KZT от %d (%d перев.), исходящих ≥5000 нет, хотя обход шёл дальше (колено %d)",
			kzt(f.InKZT), f.InDeg, f.InTx, f.Depth)
	default:
		switch {
		case f.InDeg == 0 && f.OutDeg == 0:
			if f.IsSeed {
				return "нет переводов ≥5000 KZT в июле; seed только по оперативной информации"
			}
			return "нет переводов ≥5000 KZT в июле"
		case f.Truncated:
			return fmt.Sprintf("получил %s KZT от %d%s; признаков роли нет", kzt(f.InKZT), f.InDeg, trunc)
		default:
			return fmt.Sprintf("вход %d/%s KZT, выход %d/%s KZT (пропуск %s): признаков роли нет", f.InDeg, kzt(f.InKZT), f.OutDeg, kzt(f.OutKZT), pt)
		}
	}
}

func btwThresholds(res *Result) thresholds {
	vals := make([]float64, len(res.Nodes))
	for i, n := range res.Nodes {
		vals[i] = n.Features.Betweenness
	}
	sort.Float64s(vals)
	q := func(p float64) float64 { return vals[int(math.Min(float64(len(vals)-1), p*float64(len(vals))))] }
	th := thresholds{btwP95: q(coordBtwPct), btwP99: q(0.99)}
	if th.btwP99 <= 0 {
		th.btwP99 = 1
	}
	return th
}

func clamp(v, lo, hi float64) float64 {
	if math.IsNaN(v) {
		return lo
	}
	return math.Max(lo, math.Min(hi, v))
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
