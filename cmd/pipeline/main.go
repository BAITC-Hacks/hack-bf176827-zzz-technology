// Пайплайн: parquet → метрики → роли/кластеры/приоритеты → out/*.csv + out/graph.json.
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"context"
	"path/filepath"

	"hackaton/internal/analysis"
	"hackaton/internal/config"
	"hackaton/internal/data/parquet"
	"hackaton/internal/llm"
	"hackaton/internal/services/assistant"
)

func main() {
	data := flag.String("data", "data", "папка с parquet-файлами")
	out := flag.String("out", "out", "куда писать выгрузки")
	topN := flag.Int("top", 30, "размер top_nodes.csv")
	useLLM := flag.Bool("llm", false, "вызывать LLM для гипотез кластеров (нужен OPENAI_API_KEY); без флага — только кэш")
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
	enrichHypotheses(res, *out, *useLLM)
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
	fmt.Printf("роли: %v\nкластеров: %d, top: %d, циклов ≤5: %d, устойчивых маршрутов: %d\n", roles, len(res.Clusters), len(res.Top), len(res.Cycles), len(res.Routes))
	for _, r := range res.Robustness {
		fmt.Printf("изъятие top-%d: оборот −%.0f%%, компонент %d→%d, крупнейшая %d→%d, без плательщиков %d, seed отрезано %d\n",
			r.Removed, r.LostTurnoverShare*100, r.ComponentsBefore, r.ComponentsAfter, r.LargestBefore, r.LargestComponent, r.NodesLostAllPayers, r.SeedsDisconnected)
	}
	fmt.Printf("готово за %s → %s/\n", time.Since(start).Round(time.Millisecond), *out)
}

// enrichHypotheses — гипотезы кластеров из кэша llm_cache.json; с --llm и ключом — дозапрос недостающих.
func enrichHypotheses(res *analysis.Result, out string, useLLM bool) {
	lc := llm.Config{Model: "gpt-5.1"}
	if cfg, err := config.New(); err == nil {
		lc.Model, lc.BaseURL = cfg.LLM.Model, cfg.LLM.BaseURL
		if useLLM {
			lc.APIKey = cfg.LLM.APIKey
		}
	}
	if useLLM && lc.APIKey == "" {
		fmt.Println("llm: ключ не задан, используем только кэш")
	}
	cache := llm.OpenCache(filepath.Join(out, "llm_cache.json"))
	svc := assistant.New(res, llm.New(lc), cache)
	n, err := svc.EnrichHypotheses(context.Background(), 5, 10)
	if err != nil {
		fmt.Printf("llm: гипотезы не обновлены: %v\n", err)
	}
	fmt.Printf("гипотезы кластеров: %d из кэша/LLM, остальные по шаблону\n", n)
}
