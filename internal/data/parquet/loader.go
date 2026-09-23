// Package parquet — загрузка датасета (edges/nodes/transactions) и проверка консистентности.
package parquet

import (
	"fmt"
	"path/filepath"
	"time"

	pq "github.com/parquet-go/parquet-go"
)

type Edge struct {
	Src    int64   `parquet:"src"`
	Dst    int64   `parquet:"dst"`
	SumKZT float64 `parquet:"sum_kzt"`
	NTx    int64   `parquet:"n_tx"`
	Depth  int8    `parquet:"depth"`
}

type Node struct {
	Gid    int64 `parquet:"gid"`
	Depth  int64 `parquet:"depth"`
	IsSeed bool  `parquet:"is_seed"`
}

type Tx struct {
	Src    int64     `parquet:"src"`
	Dst    int64     `parquet:"dst"`
	Date   time.Time `parquet:"date,date"`
	SumKZT float64   `parquet:"sum_kzt"`
}

type Dataset struct {
	Edges []Edge
	Nodes []Node
	Tx    []Tx
}

func Load(dir string) (Dataset, error) {
	var ds Dataset
	var err error
	if ds.Edges, err = pq.ReadFile[Edge](filepath.Join(dir, "edges.parquet")); err != nil {
		return ds, fmt.Errorf("edges: %w", err)
	}
	if ds.Nodes, err = pq.ReadFile[Node](filepath.Join(dir, "nodes.parquet")); err != nil {
		return ds, fmt.Errorf("nodes: %w", err)
	}
	if ds.Tx, err = pq.ReadFile[Tx](filepath.Join(dir, "transactions.parquet")); err != nil {
		return ds, fmt.Errorf("transactions: %w", err)
	}
	return ds, nil
}

// Stats — сводка для лога и sanity-check.
type Stats struct {
	Nodes, Edges, Tx, Seeds, Orphans, OrphanSeeds int
	TotalKZT                                      float64
	DateMin, DateMax                              time.Time
}

// SanityCheck: транзакции сходятся с рёбрами по парам и суммам; считает узлы без рёбер.
func (ds Dataset) SanityCheck() (Stats, error) {
	st := Stats{Nodes: len(ds.Nodes), Edges: len(ds.Edges), Tx: len(ds.Tx)}
	type key struct{ s, d int64 }
	agg := map[key]struct {
		sum float64
		n   int64
	}{}
	for _, t := range ds.Tx {
		a := agg[key{t.Src, t.Dst}]
		a.sum += t.SumKZT
		a.n++
		agg[key{t.Src, t.Dst}] = a
		if st.DateMin.IsZero() || t.Date.Before(st.DateMin) {
			st.DateMin = t.Date
		}
		if t.Date.After(st.DateMax) {
			st.DateMax = t.Date
		}
	}
	if len(agg) != len(ds.Edges) {
		return st, fmt.Errorf("пар в transactions %d, рёбер %d", len(agg), len(ds.Edges))
	}
	inEdges := map[int64]bool{}
	for _, e := range ds.Edges {
		a, ok := agg[key{e.Src, e.Dst}]
		if !ok || a.n != e.NTx || abs(a.sum-e.SumKZT) > 0.01 {
			return st, fmt.Errorf("ребро %d→%d не сходится с транзакциями", e.Src, e.Dst)
		}
		st.TotalKZT += e.SumKZT
		inEdges[e.Src], inEdges[e.Dst] = true, true
	}
	for _, n := range ds.Nodes {
		if n.IsSeed {
			st.Seeds++
		}
		if !inEdges[n.Gid] {
			st.Orphans++
			if n.IsSeed {
				st.OrphanSeeds++
			}
		}
	}
	return st, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
