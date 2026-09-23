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
	SumKZT float64 `json:"sum_kzt"`
	NTx    int64   `json:"n_tx"`
}

// NodeCardResponse — ответ /nodes/{gid}: узел и его контрагенты по убыванию суммы.
type NodeCardResponse struct {
	Node     NodeDetailResponse `json:"node"`
	Incoming []NeighborResponse `json:"incoming"`
	Outgoing []NeighborResponse `json:"outgoing"`
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

func NewNodeCardResponse(node models.NodeResult, incoming, outgoing []models.Edge) NodeCardResponse {
	resp := NodeCardResponse{Node: NewNodeDetailResponse(node), Incoming: []NeighborResponse{}, Outgoing: []NeighborResponse{}}
	for _, edge := range incoming {
		resp.Incoming = append(resp.Incoming, NeighborResponse{Gid: GID(edge.Payer), SumKZT: edge.SumKZT, NTx: edge.TxCount})
	}
	for _, edge := range outgoing {
		resp.Outgoing = append(resp.Outgoing, NeighborResponse{Gid: GID(edge.Payee), SumKZT: edge.SumKZT, NTx: edge.TxCount})
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
