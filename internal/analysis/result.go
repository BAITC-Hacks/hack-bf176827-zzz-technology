// Package analysis — метрики, роли, кластеры и приоритеты узлов. Контракт см. .agent/team.md.
package analysis

import "hackaton/internal/data/parquet"

const (
	RoleConsolidator = "consolidator"
	RoleTransit      = "transit"
	RoleDistributor  = "distributor"
	RoleTerminal     = "terminal"
	RoleCoordinator  = "coordinator"
	RolePeripheral   = "peripheral"
)

var Roles = []string{RoleConsolidator, RoleTransit, RoleDistributor, RoleTerminal, RoleCoordinator, RolePeripheral}

// Features — все метрики узла (заполняются в metrics.go).
type Features struct {
	Depth  int  `json:"depth"`
	IsSeed bool `json:"is_seed"`

	InDeg  int     `json:"in_deg"`
	OutDeg int     `json:"out_deg"`
	InKZT  float64 `json:"in_kzt"`
	OutKZT float64 `json:"out_kzt"`
	InTx   int     `json:"in_tx"`
	OutTx  int     `json:"out_tx"`
	AvgIn  float64 `json:"avg_in_tx"`
	AvgOut float64 `json:"avg_out_tx"`
	// out_kzt / in_kzt; NaN → -1 (нет входящих)
	PassThrough float64 `json:"pass_through"`

	PageRank    float64 `json:"pagerank"`
	Hub         float64 `json:"hub"`
	Authority   float64 `json:"authority"`
	Betweenness float64 `json:"betweenness"`

	NSeedPayers   int `json:"n_seed_payers"`
	NSeedUpstream int `json:"n_seed_upstream"`
	NClustersIn   int `json:"n_clusters_in"`
	ComponentID   int `json:"component_id"`
	ComponentSize int `json:"component_size"`

	Truncated    bool `json:"truncated"`     // depth==4 && out_deg==0: обрезан обходом
	VerifiedSink bool `json:"verified_sink"` // depth<4 && out_deg==0 && in_deg>0

	ActiveDays       int     `json:"active_days"`
	MaxSameDayPayers int     `json:"max_same_day_payers"`
	FastForwardShare float64 `json:"fast_forward_share"` // доля out, ушедшая ≤2 дней после in
	Reciprocal       bool    `json:"reciprocal"`
	InCycle          bool    `json:"in_cycle"`
	RepeatRoutes     int     `json:"repeat_routes"` // устойчивых маршрутов A→B→C через узел
}

type NodeResult struct {
	Gid           int64
	Role          string
	RoleScore     float64
	ClusterID     int
	PriorityScore float64
	Evidence      string
	Features      Features
}

type ClusterResult struct {
	ClusterID      int
	ComponentID    int
	NNodes         int
	NSeed          int
	SumKZTInternal float64
	SumKZTIn       float64
	SumKZTOut      float64
	RoleCounts     map[string]int
	NCycles        int
	TopGids        []int64
	Hypothesis     string
}

type TopNode struct {
	Rank          int
	Gid           int64
	Role          string
	PriorityScore float64
	Why           string
}

type Result struct {
	Nodes      []NodeResult
	Clusters   []ClusterResult
	Top        []TopNode
	Edges      []parquet.Edge
	ByGid      map[int64]*NodeResult
	Cycles     []Cycle
	Routes     []Route
	Robustness []RobustnessStep
}

type Options struct {
	TopN int // размер top_nodes (по умолчанию 30)
}
