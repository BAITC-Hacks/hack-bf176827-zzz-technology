package graphdto

import "hackaton/internal/data/models"

// SubgraphResponse — ответ /graph и /nodes/{gid}/ego.
type SubgraphResponse struct {
	Nodes []NodeResponse `json:"nodes"`
	Edges []EdgeResponse `json:"edges"`
}

// NodeDetailResponse — узел с полным набором метрик (карточка, поиск).
type NodeDetailResponse struct {
	NodeResponse
	Features FeaturesResponse `json:"features"`
}

type NeighborResponse struct {
	Gid    string  `json:"gid"`
	Role   string  `json:"role"`
	IsSeed bool    `json:"is_seed"`
	SumKZT float64 `json:"sum_kzt"`
	NTx    int64   `json:"n_tx"`
}

// NodeCardResponse — ответ /nodes/{gid}: узел и его контрагенты по убыванию суммы.
type NodeCardResponse struct {
	Node           NodeDetailResponse   `json:"node"`
	Incoming       []NeighborResponse   `json:"incoming"`
	Outgoing       []NeighborResponse   `json:"outgoing"`
	Percentiles    map[string]float64   `json:"percentiles"` // метрика → доля узлов сети с меньшим значением, 0–100
	NearestSeed    *NearestSeedResponse `json:"nearest_seed"`
	NextCandidates []CandidateResponse  `json:"next_candidates"`
}

func NewSubgraphResponse(nodes []models.NodeResult, edges []models.Edge) SubgraphResponse {
	resp := SubgraphResponse{Nodes: make([]NodeResponse, 0, len(nodes)), Edges: make([]EdgeResponse, 0, len(edges))}
	for _, node := range nodes {
		resp.Nodes = append(resp.Nodes, NewNodeResponse(node))
	}
	for _, edge := range edges {
		resp.Edges = append(resp.Edges, NewEdgeResponse(edge))
	}
	return resp
}

func NewNodeDetailResponse(node models.NodeResult) NodeDetailResponse {
	return NodeDetailResponse{NodeResponse: NewNodeResponse(node), Features: NewFeaturesResponse(node.Features)}
}

func NewNodeDetailResponses(nodes []models.NodeResult) []NodeDetailResponse {
	resp := make([]NodeDetailResponse, 0, len(nodes))
	for _, node := range nodes {
		resp = append(resp, NewNodeDetailResponse(node))
	}
	return resp
}

// NewNeighborResponses — контрагенты по рёбрам; infoOf отдаёт роль и seed-признак узла.
func NewNeighborResponses(edges []models.Edge, other func(models.Edge) int64, infoOf func(int64) (string, bool)) []NeighborResponse {
	resp := make([]NeighborResponse, 0, len(edges))
	for _, edge := range edges {
		gid := other(edge)
		role, isSeed := infoOf(gid)
		resp = append(resp, NeighborResponse{Gid: GID(gid), Role: role, IsSeed: isSeed, SumKZT: edge.SumKZT, NTx: edge.TxCount})
	}
	return resp
}

func NewClusterResponses(clusters []models.ClusterResult) []ClusterResponse {
	resp := make([]ClusterResponse, 0, len(clusters))
	for _, cluster := range clusters {
		resp = append(resp, NewClusterResponse(cluster))
	}
	return resp
}

func NewTopNodeResponses(top []models.TopNode) []TopNodeResponse {
	resp := make([]TopNodeResponse, 0, len(top))
	for _, node := range top {
		resp = append(resp, NewTopNodeResponse(node))
	}
	return resp
}

// NearestSeedResponse — ближайший seed выше по цепочке денег.
type NearestSeedResponse struct {
	Gid   string `json:"gid"`
	Steps int    `json:"steps"`
}

// CandidateResponse — «куда смотреть дальше»: следующий вероятный ключевой узел.
type CandidateResponse struct {
	Gid       string  `json:"gid"`
	Role      string  `json:"role"`
	Priority  float64 `json:"priority"`
	Hops      int     `json:"hops"`
	Direction string  `json:"direction"` // down — получатель, up — плательщик
	FlowShare float64 `json:"flow_share"`
	ScorePct  int     `json:"score_pct"`
}

// SeedResponse — исходный участник и куда его деньги ушли дальше.
type SeedResponse struct {
	Gid      string             `json:"gid"`
	Role     string             `json:"role"`
	Cluster  int                `json:"cluster"`
	OutKZT   float64            `json:"out_kzt"`
	OutDeg   int                `json:"out_deg"`
	Priority float64            `json:"priority"`
	Next     []NeighborResponse `json:"next"` // крупнейшие получатели
}
