package models

// Cycle — возвратный поток: деньги возвращаются к отправителю.
type Cycle struct {
	GIDs   []int64
	SumKZT float64 // минимальная сумма по рёбрам цикла
}

// Route — устойчивая цепочка A→B→C с повторными переводами на обоих рёбрах.
type Route struct {
	From, Via, To int64
	TxCount       int64 // min по двум рёбрам
	SumKZT        float64
}

// RobustnessStep — состояние сети после изъятия набора узлов.
type RobustnessStep struct {
	RemovedGIDs        []int64
	LostTurnoverShare  float64
	ComponentsBefore   int
	ComponentsAfter    int
	LargestBefore      int
	LargestAfter       int
	NodesLostAllPayers int // узлы, у которых не осталось ни одного плательщика
	SeedsDisconnected  int // seed, чьи все исходящие цепочки оборваны
}
