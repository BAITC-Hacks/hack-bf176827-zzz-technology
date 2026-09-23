package graph

import (
	"sort"

	"hackaton/internal/data/models"
)

const (
	nearestSeedMaxSteps = 6
	candidateHops       = 3
	candidateLimit      = 3
	candidateHopDecay   = 0.8 // штраф за каждый следующий шаг
	peripheralPenalty   = 0.3
)

// percentileKeys — метрики, по которым считаем «выше, чем у N% узлов».
var percentileKeys = map[string]func(models.Features) float64{
	"in_deg":              func(f models.Features) float64 { return float64(f.InDegree) },
	"out_deg":             func(f models.Features) float64 { return float64(f.OutDegree) },
	"in_kzt":              func(f models.Features) float64 { return f.InKZT },
	"out_kzt":             func(f models.Features) float64 { return f.OutKZT },
	"in_tx":               func(f models.Features) float64 { return float64(f.InTxCount) },
	"out_tx":              func(f models.Features) float64 { return float64(f.OutTxCount) },
	"avg_in_tx":           func(f models.Features) float64 { return f.AvgInTx },
	"avg_out_tx":          func(f models.Features) float64 { return f.AvgOutTx },
	"pagerank":            func(f models.Features) float64 { return f.PageRank },
	"betweenness":         func(f models.Features) float64 { return f.Betweenness },
	"hub":                 func(f models.Features) float64 { return f.Hub },
	"authority":           func(f models.Features) float64 { return f.Authority },
	"n_seed_payers":       func(f models.Features) float64 { return float64(f.SeedPayers) },
	"n_seed_upstream":     func(f models.Features) float64 { return float64(f.SeedUpstream) },
	"n_clusters_in":       func(f models.Features) float64 { return float64(f.ClustersIn) },
	"active_days":         func(f models.Features) float64 { return float64(f.ActiveDays) },
	"max_same_day_payers": func(f models.Features) float64 { return float64(f.MaxSameDayPayers) },
	"fast_forward_share":  func(f models.Features) float64 { return f.FastForwardShare },
	"repeat_routes":       func(f models.Features) float64 { return float64(f.RepeatRoutes) },
}

// buildPercentileIndex — отсортированные значения каждой метрики по всем узлам.
func buildPercentileIndex(nodes []models.NodeResult) map[string][]float64 {
	index := make(map[string][]float64, len(percentileKeys))
	for key, value := range percentileKeys {
		values := make([]float64, len(nodes))
		for i, node := range nodes {
			values[i] = value(node.Features)
		}
		sort.Float64s(values)
		index[key] = values
	}
	return index
}

// percentiles — доля узлов со строго меньшим значением, 0–100.
func (s *service) percentiles(f models.Features) map[string]float64 {
	out := make(map[string]float64, len(percentileKeys))
	for key, value := range percentileKeys {
		sorted := s.sorted[key]
		if len(sorted) == 0 {
			continue
		}
		below := sort.SearchFloat64s(sorted, value(f))
		out[key] = float64(below) / float64(len(sorted)) * 100
	}
	return out
}

// nearestSeed — ближайший seed вверх по цепочке денег (nil для самого seed и если seed не доходит).
func (s *service) nearestSeed(gid int64) *NearestSeed {
	if node, ok := s.result.Node(gid); !ok || node.Features.IsSeed {
		return nil
	}
	var best *NearestSeed
	for upstream, steps := range s.index.Upstream([]int64{gid}, nearestSeedMaxSteps) {
		node, ok := s.result.Node(upstream)
		if !ok || !node.Features.IsSeed || upstream == gid {
			continue
		}
		if best == nil || steps < best.Steps || (steps == best.Steps && upstream < best.GID) {
			best = &NearestSeed{GID: upstream, Steps: steps}
		}
	}
	return best
}

// nextCandidates — «куда смотреть дальше»: доля потока × приоритет на ≤ 3 шагах вниз; если исходящих нет — вверх.
func (s *service) nextCandidates(gid int64, visited map[int64]bool) []Candidate {
	candidates := s.walkCandidates(gid, "down", visited)
	if len(candidates) == 0 {
		candidates = s.walkCandidates(gid, "up", visited)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].GID < candidates[j].GID
	})
	if len(candidates) > candidateLimit {
		candidates = candidates[:candidateLimit]
	}
	total := 0.0
	for _, c := range candidates {
		total += c.Score
	}
	for i := range candidates {
		if total > 0 {
			candidates[i].ScorePct = max(1, int(candidates[i].Score/total*100+0.5))
		}
	}
	return candidates
}

type flowFront struct {
	gid   int64
	share float64
	path  map[int64]bool
}

func (s *service) walkCandidates(start int64, direction string, visited map[int64]bool) []Candidate {
	best := map[int64]Candidate{}
	frontier := []flowFront{{gid: start, share: 1, path: map[int64]bool{start: true}}}
	for hop := 1; hop <= candidateHops; hop++ {
		var next []flowFront
		for _, front := range frontier {
			edges := s.index.Outgoing[front.gid]
			if direction == "up" {
				edges = s.index.Incoming[front.gid]
			}
			total := 0.0
			for _, edge := range edges {
				total += edge.SumKZT
			}
			if total == 0 {
				continue
			}
			for _, edge := range edges {
				other := edge.Payee
				if direction == "up" {
					other = edge.Payer
				}
				if front.path[other] {
					continue
				}
				node, ok := s.result.Node(other)
				if !ok {
					continue
				}
				share := front.share * edge.SumKZT / total
				score := share * (0.25 + 0.75*node.PriorityScore)
				for i := 1; i < hop; i++ {
					score *= candidateHopDecay
				}
				if visited[other] {
					score *= 0.4
				}
				if node.Role == models.RolePeripheral {
					score *= peripheralPenalty
				}
				if prev, seen := best[other]; !seen || prev.Score < score {
					best[other] = Candidate{GID: other, Role: node.Role, Priority: node.PriorityScore, Hops: hop, Direction: direction, FlowShare: share, Score: score}
				}
				path := make(map[int64]bool, len(front.path)+1)
				for k := range front.path {
					path[k] = true
				}
				path[other] = true
				next = append(next, flowFront{gid: other, share: share, path: path})
			}
		}
		frontier = next
	}
	out := make([]Candidate, 0, len(best))
	for _, c := range best {
		out = append(out, c)
	}
	return out
}

// Seeds — исходные участники с крупнейшими получателями, по убыванию отправленного.
func (s *service) Seeds() []SeedInfo {
	var seeds []SeedInfo
	for _, node := range s.result.Nodes {
		if !node.Features.IsSeed {
			continue
		}
		out := s.index.Outgoing[node.GID]
		if len(out) > seedNextLimit {
			out = out[:seedNextLimit]
		}
		seeds = append(seeds, SeedInfo{Node: node, Next: out})
	}
	sort.SliceStable(seeds, func(i, j int) bool {
		if seeds[i].Node.Features.OutKZT != seeds[j].Node.Features.OutKZT {
			return seeds[i].Node.Features.OutKZT > seeds[j].Node.Features.OutKZT
		}
		return seeds[i].Node.GID < seeds[j].Node.GID
	})
	return seeds
}

func (s *service) Robustness() []models.RobustnessStep { return s.result.Robustness }
