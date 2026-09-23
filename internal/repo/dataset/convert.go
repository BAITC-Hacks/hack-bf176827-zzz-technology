package dataset

import "hackaton/internal/data/models"

func fromRows(edges []edgeRow, nodes []nodeRow, transactions []transactionRow) models.Dataset {
	ds := models.Dataset{
		Edges:        make([]models.Edge, 0, len(edges)),
		Nodes:        make([]models.Node, 0, len(nodes)),
		Transactions: make([]models.Transaction, 0, len(transactions)),
	}
	for _, row := range edges {
		ds.Edges = append(ds.Edges, models.Edge{Payer: row.Payer, Payee: row.Payee, SumKZT: row.SumKZT, TxCount: row.TxCount, Depth: int(row.Depth)})
	}
	for _, row := range nodes {
		ds.Nodes = append(ds.Nodes, models.Node{GID: row.GID, Depth: int(row.Depth), IsSeed: row.IsSeed})
	}
	for _, row := range transactions {
		ds.Transactions = append(ds.Transactions, models.Transaction{Payer: row.Payer, Payee: row.Payee, Date: row.Date, SumKZT: row.SumKZT})
	}
	return ds
}
