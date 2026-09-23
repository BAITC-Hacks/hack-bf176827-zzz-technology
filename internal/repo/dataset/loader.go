// Package dataset — чтение выгрузки (edges/nodes/transactions.parquet) в доменные модели.
package dataset

import (
	"fmt"
	"path/filepath"
	"time"

	"hackaton/internal/data/models"

	"github.com/parquet-go/parquet-go"
)

// Строки parquet-файлов; наружу не выходят.
type edgeRow struct {
	Payer   int64   `parquet:"src"`
	Payee   int64   `parquet:"dst"`
	SumKZT  float64 `parquet:"sum_kzt"`
	TxCount int64   `parquet:"n_tx"`
	Depth   int8    `parquet:"depth"`
}

type nodeRow struct {
	GID    int64 `parquet:"gid"`
	Depth  int64 `parquet:"depth"`
	IsSeed bool  `parquet:"is_seed"`
}

type transactionRow struct {
	Payer  int64     `parquet:"src"`
	Payee  int64     `parquet:"dst"`
	Date   time.Time `parquet:"date,date"`
	SumKZT float64   `parquet:"sum_kzt"`
}

type Loader interface {
	Load(dir string) (models.Dataset, error)
}

type loader struct{}

func NewLoader() Loader { return &loader{} }

func (l *loader) Load(dir string) (models.Dataset, error) {
	edges, err := parquet.ReadFile[edgeRow](filepath.Join(dir, "edges.parquet"))
	if err != nil {
		return models.Dataset{}, fmt.Errorf("edges.parquet: %w", err)
	}
	nodes, err := parquet.ReadFile[nodeRow](filepath.Join(dir, "nodes.parquet"))
	if err != nil {
		return models.Dataset{}, fmt.Errorf("nodes.parquet: %w", err)
	}
	transactions, err := parquet.ReadFile[transactionRow](filepath.Join(dir, "transactions.parquet"))
	if err != nil {
		return models.Dataset{}, fmt.Errorf("transactions.parquet: %w", err)
	}
	return fromRows(edges, nodes, transactions), nil
}
