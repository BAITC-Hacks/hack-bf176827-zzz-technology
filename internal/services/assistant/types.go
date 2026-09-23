package assistant

import (
	"regexp"

	graphdto "hackaton/internal/data/dto/graph"
	"hackaton/internal/data/models"
)

// Answer — ответ ассистента и gid, которые он упомянул (для подсветки в UI).
type Answer struct {
	Text   string
	GIDs   []string
	Steps  int
	Cached bool
}

// Card — справка по узлу.
type Card struct {
	Text  string
	ByLLM bool
}

var gidPattern = regexp.MustCompile(`\b1\d{17}\b`)

func extractGIDs(text string) []string {
	seen := map[string]bool{}
	var gids []string
	for _, match := range gidPattern.FindAllString(text, -1) {
		if !seen[match] {
			seen[match] = true
			gids = append(gids, match)
		}
	}
	return gids
}

// toolArgs — объединение аргументов всех инструментов (модель присылает JSON).
type toolArgs struct {
	GID         string   `json:"gid"`
	GIDs        []string `json:"gids"`
	Src         string   `json:"src"`
	Dst         string   `json:"dst"`
	Role        string   `json:"role"`
	ClusterID   *int     `json:"cluster_id"`
	IsSeed      *bool    `json:"is_seed"`
	MinPriority float64  `json:"min_priority"`
	Limit       int      `json:"limit"`
	Depth       int      `json:"depth"`
	MaxLen      int      `json:"max_len"`
	N           int      `json:"n"`
}

// nodeBrief — короткое описание узла в ответах инструментов.
type nodeBrief struct {
	GID      string  `json:"gid"`
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

func newNodeBrief(node *models.NodeResult) nodeBrief {
	f := node.Features
	return nodeBrief{GID: graphdto.GID(node.GID), Role: string(node.Role), Priority: node.PriorityScore, InDeg: f.InDegree, OutDeg: f.OutDegree,
		InKZT: f.InKZT, OutKZT: f.OutKZT, IsSeed: f.IsSeed, Cluster: node.ClusterID, Evidence: node.Evidence}
}

type counterparty struct {
	GID    string  `json:"gid"`
	Role   string  `json:"role"`
	SumKZT float64 `json:"sum_kzt"`
	TxN    int64   `json:"n_tx"`
	IsSeed bool    `json:"is_seed"`
}

type nodeCard struct {
	Node      nodeBrief                 `json:"node"`
	RoleScore float64                   `json:"role_score"`
	Features  graphdto.FeaturesResponse `json:"features"`
	Payers    []counterparty            `json:"payers"`
	Receivers []counterparty            `json:"receivers"`
}

type reachedNode struct {
	nodeBrief
	Distance   int                `json:"distance"`
	FromCount  int                `json:"reached_from_n_sources"`
	From       []string           `json:"reached_from"`
	DirectSums map[string]float64 `json:"direct_sum_kzt_from_source"`
}

type pathStep struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	ToRole string  `json:"to_role"`
	SumKZT float64 `json:"sum_kzt"`
}
