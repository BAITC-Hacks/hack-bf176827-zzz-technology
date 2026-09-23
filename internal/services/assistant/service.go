// Package assistant — LLM-слой поверх результатов анализа: гипотезы кластеров, карточка узла, вопрос-ответ по графу.
package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"hackaton/internal/analysis"
	"hackaton/internal/data/parquet"
	"hackaton/internal/llm"
)

const systemPrompt = `Ты — помощник AML-аналитика банка. Данные: граф внутрибанковских переводов за июль 2026, собранный от 81 seed-клиента
(известные участники незаконного оборота) по исходящим переводам на 4 колена, порог 5000 KZT. Атрибутов клиентов нет — только gid, структура и суммы.
Роли узлов уже посчитаны правилами: consolidator (собирает от многих, удерживает), transit (пропускает дальше), distributor (веер на многих),
terminal (получает и не отдаёт, подтверждено обходом), coordinator (связывает группы, высокое посредничество), peripheral (признаков нет).
Правила ответа: по-русски; кратко; всегда ссылайся на конкретные gid (полностью, 18 цифр); все выводы — гипотезы для проверки
("признаки консолидации"), не утверждения о виновности; ничего не выдумывай: если данных нет — так и скажи; не приписывай клиентам
пол, возраст, организацию и прочие атрибуты. Сначала вызывай инструменты, потом отвечай. Ограничения данных: у seed входящие занижены,
узлы 4-го колена без исходящих могут быть обрезаны обходом (truncated), суммы ниже 5000 не видны.`

type Service struct {
	res   *analysis.Result
	idx   *analysis.Index
	llm   *llm.Client
	cache *llm.Cache
}

func New(res *analysis.Result, client *llm.Client, cache *llm.Cache) *Service {
	return &Service{res: res, idx: analysis.BuildIndex(res), llm: client, cache: cache}
}

func (s *Service) Enabled() bool { return s.llm.Enabled() }

type Answer struct {
	Text   string   `json:"answer"`
	Gids   []string `json:"gids"`
	Steps  int      `json:"steps"`
	Cached bool     `json:"cached"`
}

var gidRe = regexp.MustCompile(`\b1\d{17}\b`)

// Ask — вопрос на естественном языке → ответ по графу через инструменты. Кэшируется по тексту вопроса.
func (s *Service) Ask(ctx context.Context, question string) (Answer, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return Answer{}, fmt.Errorf("пустой вопрос")
	}
	key := llm.Key("ask", s.llm.Model(), question)
	if s.cache != nil {
		if v, ok := s.cache.Get(key); ok {
			return Answer{Text: v, Gids: extractGids(v), Cached: true}, nil
		}
	}
	if !s.llm.Enabled() {
		return Answer{}, llm.ErrDisabled
	}
	ans, err := s.llm.RunTools(ctx, systemPrompt, question, s.tools(), s.exec, 8)
	if err != nil {
		return Answer{}, err
	}
	if s.cache != nil {
		s.cache.Put(key, ans.Text)
		_ = s.cache.Save()
	}
	return Answer{Text: ans.Text, Gids: extractGids(ans.Text), Steps: ans.Steps}, nil
}

func extractGids(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range gidRe.FindAllString(text, -1) {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// ---- инструменты (все читают Result в памяти)

func (s *Service) tools() []llm.Tool {
	obj := func(props map[string]any, required ...string) map[string]any {
		m := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
		if len(required) > 0 {
			m["required"] = required
		}
		return m
	}
	str := map[string]any{"type": "string"}
	num := map[string]any{"type": "integer"}
	return []llm.Tool{
		{Name: "get_node", Description: "Карточка узла: роль, скоры, evidence, метрики, топ входящих и исходящих контрагентов с суммами.",
			Parameters: obj(map[string]any{"gid": str}, "gid")},
		{Name: "find_nodes", Description: "Поиск узлов по фильтрам, отсортировано по приоритету. role: consolidator|transit|distributor|terminal|coordinator|peripheral.",
			Parameters: obj(map[string]any{"role": str, "cluster_id": num, "is_seed": map[string]any{"type": "boolean"}, "min_priority": map[string]any{"type": "number"}, "limit": num})},
		{Name: "who_receives_from", Description: "Кто получает деньги от заданных узлов вниз по цепочке (≤ depth шагов): общие получатели, суммы, роли. Используй для «кто собирает деньги с этих N».",
			Parameters: obj(map[string]any{"gids": map[string]any{"type": "array", "items": str}, "depth": num}, "gids")},
		{Name: "who_pays_to", Description: "Кто платит узлу вверх по цепочке (≤ depth шагов): плательщики, суммы, роли, сколько среди них seed.",
			Parameters: obj(map[string]any{"gid": str, "depth": num}, "gid")},
		{Name: "path", Description: "Кратчайший маршрут денег от src к dst (≤ max_len шагов) с суммами по рёбрам.",
			Parameters: obj(map[string]any{"src": str, "dst": str, "max_len": num}, "src", "dst")},
		{Name: "cluster_info", Description: "Кластер: размер, seed, обороты, состав ролей, топ-узлы, гипотеза.",
			Parameters: obj(map[string]any{"cluster_id": num}, "cluster_id")},
		{Name: "top", Description: "Топ узлов по приоритету для проверки (опционально по роли).",
			Parameters: obj(map[string]any{"n": num, "role": str})},
		{Name: "remove_nodes", Description: "Что будет с сетью, если изъять (заблокировать) узлы: сколько компонент, какая доля оборота отвалится, кто потеряет входящие.",
			Parameters: obj(map[string]any{"gids": map[string]any{"type": "array", "items": str}}, "gids")},
	}
}

func (s *Service) exec(name, argsJSON string) (string, error) {
	var a struct {
		Gid         string   `json:"gid"`
		Gids        []string `json:"gids"`
		Src, Dst    string
		Role        string  `json:"role"`
		ClusterID   *int    `json:"cluster_id"`
		IsSeed      *bool   `json:"is_seed"`
		MinPriority float64 `json:"min_priority"`
		Limit       int     `json:"limit"`
		Depth       int     `json:"depth"`
		MaxLen      int     `json:"max_len"`
		N           int     `json:"n"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &a); err != nil {
		return "", fmt.Errorf("bad args: %w", err)
	}
	var v any
	var err error
	switch name {
	case "get_node":
		v, err = s.nodeCard(a.Gid)
	case "find_nodes":
		v = s.findNodes(a.Role, a.ClusterID, a.IsSeed, a.MinPriority, a.Limit)
	case "who_receives_from":
		v, err = s.whoReceivesFrom(a.Gids, a.Depth)
	case "who_pays_to":
		v, err = s.whoPaysTo(a.Gid, a.Depth)
	case "path":
		v, err = s.path(a.Src, a.Dst, a.MaxLen)
	case "cluster_info":
		v, err = s.clusterInfo(a.ClusterID)
	case "top":
		v = s.top(a.N, a.Role)
	case "remove_nodes":
		v, err = s.removeNodes(a.Gids)
	default:
		return "", fmt.Errorf("unknown tool %s", name)
	}
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(v)
	return string(b), nil
}

func parseGid(s string) (int64, error) {
	g, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("некорректный gid %q", s)
	}
	return g, nil
}

type brief struct {
	Gid      string  `json:"gid"`
	Role     string  `json:"role"`
	Priority float64 `json:"priority"`
	InDeg    int     `json:"in_deg"`
	OutDeg   int     `json:"out_deg"`
	InKZT    float64 `json:"in_kzt"`
	OutKZT   float64 `json:"out_kzt"`
	IsSeed   bool    `json:"is_seed"`
	Cluster  int     `json:"cluster_id"`
	Evidence string  `json:"evidence"`
}

func (s *Service) brief(n *analysis.NodeResult) brief {
	f := n.Features
	return brief{Gid: strconv.FormatInt(n.Gid, 10), Role: n.Role, Priority: n.PriorityScore, InDeg: f.InDeg, OutDeg: f.OutDeg,
		InKZT: f.InKZT, OutKZT: f.OutKZT, IsSeed: f.IsSeed, Cluster: n.ClusterID, Evidence: n.Evidence}
}

type counterparty struct {
	Gid    string  `json:"gid"`
	Role   string  `json:"role"`
	SumKZT float64 `json:"sum_kzt"`
	NTx    int64   `json:"n_tx"`
	IsSeed bool    `json:"is_seed"`
}

func (s *Service) nodeCard(gid string) (any, error) {
	g, err := parseGid(gid)
	if err != nil {
		return nil, err
	}
	n, ok := s.res.ByGid[g]
	if !ok {
		return nil, fmt.Errorf("узел %s не найден в выгрузке", gid)
	}
	cp := func(es []parquet.Edge, pick func(e parquet.Edge) int64) []counterparty {
		var out []counterparty
		for i, e := range es {
			if i >= 10 {
				break
			}
			o := s.res.ByGid[pick(e)]
			c := counterparty{Gid: strconv.FormatInt(pick(e), 10), SumKZT: e.SumKZT, NTx: e.NTx}
			if o != nil {
				c.Role, c.IsSeed = o.Role, o.Features.IsSeed
			}
			out = append(out, c)
		}
		return out
	}
	return map[string]any{
		"node":       s.brief(n),
		"features":   n.Features,
		"role_score": n.RoleScore,
		"payers":     cp(s.idx.In[g], func(e parquet.Edge) int64 { return e.Src }),
		"receivers":  cp(s.idx.Out[g], func(e parquet.Edge) int64 { return e.Dst }),
	}, nil
}

func (s *Service) findNodes(role string, cluster *int, isSeed *bool, minPriority float64, limit int) any {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var out []brief
	for _, i := range sortedIdx(s.res) {
		n := &s.res.Nodes[i]
		if role != "" && n.Role != role {
			continue
		}
		if cluster != nil && n.ClusterID != *cluster {
			continue
		}
		if isSeed != nil && n.Features.IsSeed != *isSeed {
			continue
		}
		if n.PriorityScore < minPriority {
			continue
		}
		out = append(out, s.brief(n))
		if len(out) >= limit {
			break
		}
	}
	return map[string]any{"count": len(out), "nodes": out}
}

func (s *Service) whoReceivesFrom(gids []string, depth int) (any, error) {
	if depth <= 0 || depth > 3 {
		depth = 2
	}
	var start []int64
	for _, g := range gids {
		v, err := parseGid(g)
		if err != nil {
			return nil, err
		}
		start = append(start, v)
	}
	type hit struct {
		brief
		Dist       int                `json:"distance"`
		FromCount  int                `json:"reached_from_n_sources"`
		From       []string           `json:"reached_from"`
		DirectSums map[string]float64 `json:"direct_sum_kzt_from_source"` // прямые переводы источник→узел
	}
	agg := map[int64]*hit{}
	for _, st := range start {
		for gid, d := range s.idx.Downstream([]int64{st}, depth) {
			if gid == st {
				continue
			}
			n := s.res.ByGid[gid]
			if n == nil {
				continue
			}
			h := agg[gid]
			if h == nil {
				h = &hit{brief: s.brief(n), Dist: d}
				agg[gid] = h
			}
			h.FromCount++
			h.From = append(h.From, strconv.FormatInt(st, 10))
			if d < h.Dist {
				h.Dist = d
			}
			for _, e := range s.idx.Out[st] {
				if e.Dst == gid {
					if h.DirectSums == nil {
						h.DirectSums = map[string]float64{}
					}
					h.DirectSums[strconv.FormatInt(st, 10)] = e.SumKZT
				}
			}
		}
	}
	out := make([]hit, 0, len(agg))
	for _, h := range agg {
		out = append(out, *h)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FromCount != out[j].FromCount {
			return out[i].FromCount > out[j].FromCount
		}
		return out[i].Priority > out[j].Priority
	})
	if len(out) > 25 {
		out = out[:25]
	}
	return map[string]any{"sources": len(start), "depth": depth, "receivers": out}, nil
}

func (s *Service) whoPaysTo(gid string, depth int) (any, error) {
	g, err := parseGid(gid)
	if err != nil {
		return nil, err
	}
	if depth <= 0 || depth > 3 {
		depth = 2
	}
	type hit struct {
		brief
		Dist int `json:"distance"`
	}
	var out []hit
	seeds := 0
	for u, d := range s.idx.Upstream([]int64{g}, depth) {
		if u == g {
			continue
		}
		n := s.res.ByGid[u]
		if n == nil {
			continue
		}
		if n.Features.IsSeed {
			seeds++
		}
		out = append(out, hit{brief: s.brief(n), Dist: d})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dist != out[j].Dist {
			return out[i].Dist < out[j].Dist
		}
		return out[i].OutKZT > out[j].OutKZT
	})
	if len(out) > 30 {
		out = out[:30]
	}
	return map[string]any{"gid": gid, "depth": depth, "n_payers": len(out), "n_seed_among": seeds, "payers": out}, nil
}

func (s *Service) path(src, dst string, maxLen int) (any, error) {
	a, err := parseGid(src)
	if err != nil {
		return nil, err
	}
	b, err := parseGid(dst)
	if err != nil {
		return nil, err
	}
	if maxLen <= 0 || maxLen > 6 {
		maxLen = 4
	}
	p := s.idx.Path(a, b, maxLen)
	if p == nil {
		return map[string]any{"found": false}, nil
	}
	type step struct {
		From, To string
		Role     string  `json:"to_role"`
		SumKZT   float64 `json:"sum_kzt"`
	}
	var steps []step
	for i := 0; i+1 < len(p); i++ {
		var sum float64
		for _, e := range s.idx.Out[p[i]] {
			if e.Dst == p[i+1] {
				sum = e.SumKZT
			}
		}
		role := ""
		if n := s.res.ByGid[p[i+1]]; n != nil {
			role = n.Role
		}
		steps = append(steps, step{From: strconv.FormatInt(p[i], 10), To: strconv.FormatInt(p[i+1], 10), Role: role, SumKZT: sum})
	}
	return map[string]any{"found": true, "hops": len(steps), "steps": steps}, nil
}

func (s *Service) clusterInfo(id *int) (any, error) {
	if id == nil {
		return nil, fmt.Errorf("cluster_id обязателен")
	}
	for _, c := range s.res.Clusters {
		if c.ClusterID == *id {
			var top []brief
			for _, g := range c.TopGids {
				if n := s.res.ByGid[g]; n != nil {
					top = append(top, s.brief(n))
				}
			}
			return map[string]any{"cluster_id": c.ClusterID, "n_nodes": c.NNodes, "n_seed": c.NSeed, "sum_kzt_internal": c.SumKZTInternal,
				"sum_kzt_in": c.SumKZTIn, "sum_kzt_out": c.SumKZTOut, "roles": c.RoleCounts, "top": top, "hypothesis": c.Hypothesis}, nil
		}
	}
	return nil, fmt.Errorf("кластер %d не найден", *id)
}

func (s *Service) top(n int, role string) any {
	if n <= 0 || n > 50 {
		n = 10
	}
	var out []brief
	for _, i := range sortedIdx(s.res) {
		nd := &s.res.Nodes[i]
		if role != "" && nd.Role != role {
			continue
		}
		out = append(out, s.brief(nd))
		if len(out) >= n {
			break
		}
	}
	return out
}

// removeNodes — устойчивость: изъятие узлов → компоненты, потерянный оборот, кто лишился входящих.
func (s *Service) removeNodes(gids []string) (any, error) {
	removed := map[int64]bool{}
	for _, g := range gids {
		v, err := parseGid(g)
		if err != nil {
			return nil, err
		}
		removed[v] = true
	}
	return analysis.Robustness(s.res, removed), nil
}

func sortedIdx(res *analysis.Result) []int {
	idx := make([]int, len(res.Nodes))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		if res.Nodes[idx[a]].PriorityScore != res.Nodes[idx[b]].PriorityScore {
			return res.Nodes[idx[a]].PriorityScore > res.Nodes[idx[b]].PriorityScore
		}
		return res.Nodes[idx[a]].Gid < res.Nodes[idx[b]].Gid
	})
	return idx
}
