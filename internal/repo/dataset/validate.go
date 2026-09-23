package dataset

import (
	"errors"
	"fmt"
	"math"

	"hackaton/internal/data/models"
)

var ErrInconsistent = errors.New("dataset inconsistent")

// Validate — транзакции сходятся с рёбрами по парам, числу и суммам; считает сводку.
func Validate(ds models.Dataset) (models.DatasetStats, error) {
	stats := models.DatasetStats{Nodes: len(ds.Nodes), Edges: len(ds.Edges), Transactions: len(ds.Transactions)}

	type pair struct{ payer, payee int64 }
	type aggregate struct {
		sum   float64
		count int64
	}
	byPair := map[pair]aggregate{}
	for _, tx := range ds.Transactions {
		key := pair{tx.Payer, tx.Payee}
		agg := byPair[key]
		agg.sum += tx.SumKZT
		agg.count++
		byPair[key] = agg
		if stats.DateFrom.IsZero() || tx.Date.Before(stats.DateFrom) {
			stats.DateFrom = tx.Date
		}
		if tx.Date.After(stats.DateTo) {
			stats.DateTo = tx.Date
		}
	}
	if len(byPair) != len(ds.Edges) {
		return stats, fmt.Errorf("%w: пар в transactions %d, рёбер %d", ErrInconsistent, len(byPair), len(ds.Edges))
	}

	hasEdge := map[int64]bool{}
	for _, edge := range ds.Edges {
		agg, ok := byPair[pair{edge.Payer, edge.Payee}]
		if !ok || agg.count != edge.TxCount || math.Abs(agg.sum-edge.SumKZT) > 0.01 {
			return stats, fmt.Errorf("%w: ребро %d→%d не сходится с транзакциями", ErrInconsistent, edge.Payer, edge.Payee)
		}
		stats.TotalKZT += edge.SumKZT
		hasEdge[edge.Payer], hasEdge[edge.Payee] = true, true
	}
	for _, node := range ds.Nodes {
		if node.IsSeed {
			stats.Seeds++
		}
		if !hasEdge[node.GID] {
			stats.Orphans++
			if node.IsSeed {
				stats.OrphanSeeds++
			}
		}
	}
	return stats, nil
}
