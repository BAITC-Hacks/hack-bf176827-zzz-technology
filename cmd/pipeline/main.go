// Пайплайн: parquet → метрики → роли/кластеры/приоритеты → out/*.csv + out/graph.json.
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"hackaton/internal/analysis"
	"hackaton/internal/data/parquet"
)

func main() {
	data := flag.String("data", "data", "папка с parquet-файлами")
	out := flag.String("out", "out", "куда писать выгрузки")
	topN := flag.Int("top", 30, "размер top_nodes.csv")
	flag.Parse()

	start := time.Now()
	ds, err := parquet.Load(*data)
	if err != nil {
		log.Fatalf("load: %v", err)
	}
	st, err := ds.SanityCheck()
	if err != nil {
		log.Fatalf("sanity: %v", err)
	}
	fmt.Printf("данные: узлов %d, рёбер %d, tx %d, seed %d, оборот %.0f KZT, период %s — %s, без рёбер %d (seed %d)\n",
		st.Nodes, st.Edges, st.Tx, st.Seeds, st.TotalKZT, st.DateMin.Format("2006-01-02"), st.DateMax.Format("2006-01-02"), st.Orphans, st.OrphanSeeds)

	res, err := analysis.Run(ds, analysis.Options{TopN: *topN})
	if err != nil {
		log.Fatalf("analysis: %v", err)
	}
	if err := analysis.WriteCSV(res, *out); err != nil {
		log.Fatalf("csv: %v", err)
	}
	if err := analysis.WriteGraphJSON(res, *out); err != nil {
		log.Fatalf("json: %v", err)
	}

	roles := map[string]int{}
	for _, n := range res.Nodes {
		roles[n.Role]++
	}
	fmt.Printf("роли: %v\nкластеров: %d, top: %d\nготово за %s → %s/\n", roles, len(res.Clusters), len(res.Top), time.Since(start).Round(time.Millisecond), *out)
}
