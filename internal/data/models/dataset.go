// Package models — доменные модели без json-тегов. Наружу отдаются через dto.
package models

import "time"

// Edge — агрегированная пара плательщик→получатель за период.
type Edge struct {
	Payer   int64
	Payee   int64
	SumKZT  float64
	TxCount int64
	Depth   int // колено обхода, на котором найдено ребро (1–4)
}

// Node — клиент в выгрузке.
type Node struct {
	GID    int64
	Depth  int // минимальное колено, 0 = seed
	IsSeed bool
}

// Transaction — отдельный перевод.
type Transaction struct {
	Payer  int64
	Payee  int64
	Date   time.Time
	SumKZT float64
}

type Dataset struct {
	Edges        []Edge
	Nodes        []Node
	Transactions []Transaction
}

// DatasetStats — сводка после проверки консистентности.
type DatasetStats struct {
	Nodes, Edges, Transactions int
	Seeds                      int
	Orphans, OrphanSeeds       int // узлы без единого ребра
	TotalKZT                   float64
	DateFrom, DateTo           time.Time
}
