package models

// Features — метрики узла. Описание каждой — docs/methodology.md.
type Features struct {
	Depth  int
	IsSeed bool

	InDegree   int
	OutDegree  int
	InKZT      float64
	OutKZT     float64
	InTxCount  int
	OutTxCount int
	AvgInTx    float64
	AvgOutTx   float64
	// OutKZT / InKZT; -1, если входящих нет
	PassThrough float64

	PageRank    float64
	Hub         float64
	Authority   float64
	Betweenness float64

	SeedPayers    int // seed, платящих напрямую
	SeedUpstream  int // seed, достигающих узла за ≤ 4 шага
	ClustersIn    int // разных кластеров среди плательщиков
	ComponentID   int
	ComponentSize int

	Truncated    bool // depth == 4 и нет исходящих: обрезан обходом
	VerifiedSink bool // depth < 4, есть входящие, нет исходящих: подтверждённый сток

	ActiveDays       int
	MaxSameDayPayers int
	FastForwardShare float64 // доля исходящего, ушедшего ≤ 2 дней после входящего
	Reciprocal       bool
	InCycle          bool
	RepeatRoutes     int // устойчивых маршрутов A→B→C через узел
}

type NodeResult struct {
	GID           int64
	Role          Role
	RoleScore     float64
	ClusterID     int
	PriorityScore float64
	Evidence      string
	Features      Features
}

type ClusterResult struct {
	ClusterID      int
	ComponentID    int
	NodeCount      int
	SeedCount      int
	SumKZTInternal float64
	SumKZTIn       float64
	SumKZTOut      float64
	RoleCounts     map[Role]int
	CycleCount     int
	TopGIDs        []int64
	Hypothesis     string
}

type TopNode struct {
	Rank          int
	GID           int64
	Role          Role
	PriorityScore float64
	Why           string
}

// AnalysisResult — полный результат: узлы, кластеры, топ, паттерны.
type AnalysisResult struct {
	Nodes      []NodeResult
	Clusters   []ClusterResult
	Top        []TopNode
	Edges      []Edge
	Cycles     []Cycle
	Routes     []Route
	Robustness []RobustnessStep

	byGID map[int64]*NodeResult
}

// Index строит индекс по GID; вызывается после заполнения Nodes.
func (r *AnalysisResult) Index() {
	r.byGID = make(map[int64]*NodeResult, len(r.Nodes))
	for i := range r.Nodes {
		r.byGID[r.Nodes[i].GID] = &r.Nodes[i]
	}
}

func (r *AnalysisResult) Node(gid int64) (*NodeResult, bool) {
	node, ok := r.byGID[gid]
	return node, ok
}

func (r *AnalysisResult) Cluster(id int) (*ClusterResult, bool) {
	for i := range r.Clusters {
		if r.Clusters[i].ClusterID == id {
			return &r.Clusters[i], true
		}
	}
	return nil, false
}
