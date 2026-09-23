// Пайплайн: parquet → анализ → out/*.csv + out/graph.json. Один запуск, без ручных шагов.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"hackaton/internal/app"
	"hackaton/internal/data/models"
	"hackaton/internal/services/pipeline"

	"go.uber.org/fx"
)

func main() {
	dataDir := flag.String("data", "data", "папка с parquet-файлами")
	outDir := flag.String("out", "out", "куда писать выгрузки")
	topN := flag.Int("top", 30, "размер top_nodes.csv")
	useLLM := flag.Bool("llm", false, "дозапросить у LLM гипотезы кластеров, которых нет в кэше (нужен OPENAI_API_KEY)")
	flag.Parse()

	container := fx.New(
		app.ModuleBase(),
		app.ModuleRepositories(),
		app.ModuleServices(),
		fx.NopLogger,
		fx.Invoke(func(runner pipeline.Service) error {
			report, err := runner.Run(context.Background(), pipeline.RunParams{
				DataDir: *dataDir, OutDir: *outDir, TopN: *topN, UseLLM: *useLLM, WriteOutputs: true,
			})
			if err != nil {
				return err
			}
			printReport(report, *outDir)
			return nil
		}),
	)
	if err := container.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}

func printReport(report *pipeline.Report, outDir string) {
	stats, result := report.Stats, report.Result
	fmt.Printf("данные: узлов %d, рёбер %d, tx %d, seed %d, оборот %.0f KZT, период %s — %s, без рёбер %d (seed %d)\n",
		stats.Nodes, stats.Edges, stats.Transactions, stats.Seeds, stats.TotalKZT,
		stats.DateFrom.Format("2006-01-02"), stats.DateTo.Format("2006-01-02"), stats.Orphans, stats.OrphanSeeds)

	roles := map[models.Role]int{}
	for _, node := range result.Nodes {
		roles[node.Role]++
	}
	fmt.Printf("роли: %v\n", roles)
	fmt.Printf("кластеров: %d (гипотез из кэша/LLM: %d), top: %d, циклов ≤5: %d, устойчивых маршрутов: %d\n",
		len(result.Clusters), report.HypothesesEnriched, len(result.Top), len(result.Cycles), len(result.Routes))
	for _, step := range result.Robustness {
		fmt.Printf("изъятие top-%d: оборот −%.0f%%, компонент %d→%d, крупнейшая %d→%d, без плательщиков %d, seed отрезано %d\n",
			len(step.RemovedGIDs), step.LostTurnoverShare*100, step.ComponentsBefore, step.ComponentsAfter,
			step.LargestBefore, step.LargestAfter, step.NodesLostAllPayers, step.SeedsDisconnected)
	}
	fmt.Printf("готово за %s → %s/\n", report.Elapsed.Round(time.Millisecond), outDir)
}
