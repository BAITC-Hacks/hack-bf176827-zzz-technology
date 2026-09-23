// Команда check проверяет выгрузки для сдачи жюри.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var roles = []string{"consolidator", "transit", "distributor", "terminal", "coordinator", "peripheral"}

type row map[string]string

func readCSV(dir, name string, columns []string, visit func(row) error) error {
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return fmt.Errorf("%s: заголовок: %w", name, err)
	}
	seen := map[string]bool{}
	for _, h := range header {
		if seen[h] {
			return fmt.Errorf("%s: повтор колонки %s", name, h)
		}
		seen[h] = true
	}
	for _, c := range columns {
		if !seen[c] {
			return fmt.Errorf("%s: отсутствует колонка %s", name, c)
		}
	}
	for {
		values, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		line, _ := r.FieldPos(0)
		data := row{}
		for i, h := range header {
			data[h] = values[i]
		}
		if err := visit(data); err != nil {
			return fmt.Errorf("%s: строка %d: %w", name, line, err)
		}
	}
	return nil
}
func integer(s, field string) (int64, error) {
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil {
		return 0, fmt.Errorf("%s: ожидается целое число, получено %q", field, s)
	}
	return n, nil
}
func score(s, field string) (float64, error) {
	n, e := strconv.ParseFloat(s, 64)
	if e != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 1 {
		return 0, fmt.Errorf("%s: ожидается число от 0 до 1, получено %q", field, s)
	}
	return n, nil
}
func validate(dir string) (map[string]int, error) {
	counts := map[string]int{}
	for _, r := range roles {
		counts[r] = 0
	}
	nodes := map[string]bool{}
	clusters := map[int64]int{}
	known := map[int64]bool{}
	err := readCSV(dir, "nodes_roles.csv", []string{"gid", "role", "role_score", "cluster_id", "priority_score", "evidence"}, func(r row) error {
		gid, e := integer(r["gid"], "gid")
		if e != nil {
			return e
		}
		key := strconv.FormatInt(gid, 10)
		if nodes[key] {
			return fmt.Errorf("повтор gid %s", key)
		}
		nodes[key] = true
		if _, ok := counts[r["role"]]; !ok {
			return fmt.Errorf("неизвестная роль %q", r["role"])
		}
		counts[r["role"]]++
		for _, f := range []string{"role_score", "priority_score"} {
			if _, e := score(r[f], f); e != nil {
				return e
			}
		}
		if strings.TrimSpace(r["evidence"]) == "" || utf8.RuneCountInString(r["evidence"]) > 200 {
			return fmt.Errorf("evidence должен содержать 1–200 символов")
		}
		cluster, e := integer(r["cluster_id"], "cluster_id")
		if e != nil {
			return e
		}
		clusters[cluster]++
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(nodes) != 2248 {
		return nil, fmt.Errorf("nodes_roles.csv: ожидается 2248 узлов, получено %d", len(nodes))
	}
	err = readCSV(dir, "clusters.csv", []string{"cluster_id", "n_nodes", "n_seed", "sum_kzt_internal", "top_gids", "hypothesis"}, func(r row) error {
		id, e := integer(r["cluster_id"], "cluster_id")
		if e != nil {
			return e
		}
		if known[id] {
			return fmt.Errorf("повтор cluster_id %d", id)
		}
		known[id] = true
		if strings.TrimSpace(r["hypothesis"]) == "" {
			return fmt.Errorf("hypothesis пустой")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(known) == 0 {
		return nil, fmt.Errorf("clusters.csv: нет кластеров")
	}
	ids := make([]int64, 0, len(clusters))
	for id := range clusters {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		if !known[id] {
			return nil, fmt.Errorf("clusters.csv: отсутствует cluster_id %d", id)
		}
	}
	n := 0
	previous := 1.0
	topSeen := map[string]bool{}
	err = readCSV(dir, "top_nodes.csv", []string{"rank", "gid", "role", "priority_score", "why"}, func(r row) error {
		n++
		rank, e := integer(r["rank"], "rank")
		if e != nil {
			return e
		}
		if rank != int64(n) {
			return fmt.Errorf("ожидается rank %d, получено %d", n, rank)
		}
		gid, e := integer(r["gid"], "gid")
		if e != nil {
			return e
		}
		key := strconv.FormatInt(gid, 10)
		if !nodes[key] || topSeen[key] {
			return fmt.Errorf("неизвестный или повторный gid %s", key)
		}
		topSeen[key] = true
		if _, ok := counts[r["role"]]; !ok {
			return fmt.Errorf("неизвестная роль %q", r["role"])
		}
		priority, e := score(r["priority_score"], "priority_score")
		if e != nil {
			return e
		}
		if priority > previous {
			return fmt.Errorf("priority_score должен невозрастать")
		}
		previous = priority
		if strings.TrimSpace(r["why"]) == "" {
			return fmt.Errorf("why пустой")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if n < 20 {
		return nil, fmt.Errorf("top_nodes.csv: нужно минимум 20 строк, получено %d", n)
	}
	return counts, nil
}
func main() {
	dir := flag.String("out", "out", "каталог выгрузок")
	flag.Parse()
	counts, err := validate(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Проверка пройдена: 2248 уникальных узлов")
	for _, r := range roles {
		fmt.Printf("  %s: %d\n", r, counts[r])
	}
}
