package analysis

import (
	"fmt"

	"hackaton/internal/graph"
)

// Заглушки фундамента: заменяются реальными реализациями в metrics/clusters/roles/priority.

func computeFeatures(g *graph.Graph) map[int64]Features {
	out := make(map[int64]Features, len(g.Gids))
	for _, gid := range g.Gids {
		f := Features{Depth: g.Depth(gid), IsSeed: g.IsSeed(gid), PassThrough: -1}
		for _, e := range g.In[gid] {
			f.InDeg++
			f.InKZT += e.SumKZT
			f.InTx += int(e.NTx)
		}
		for _, e := range g.Out[gid] {
			f.OutDeg++
			f.OutKZT += e.SumKZT
			f.OutTx += int(e.NTx)
		}
		if f.InKZT > 0 {
			f.PassThrough = f.OutKZT / f.InKZT
		}
		f.Truncated = f.Depth == 4 && f.OutDeg == 0
		f.VerifiedSink = f.Depth < 4 && f.OutDeg == 0 && f.InDeg > 0
		out[gid] = f
	}
	return out
}

func assignClusters(g *graph.Graph, res *Result) {
	for i := range res.Nodes {
		res.Nodes[i].ClusterID = 0
	}
}

func assignRoles(g *graph.Graph, res *Result) {
	for i := range res.Nodes {
		n := &res.Nodes[i]
		n.Role = RolePeripheral
		n.RoleScore = 0
		n.Evidence = fmt.Sprintf("заглушка: in %d/%.0f, out %d/%.0f", n.Features.InDeg, n.Features.InKZT, n.Features.OutDeg, n.Features.OutKZT)
	}
}

func assignPriority(g *graph.Graph, res *Result, topN int) {
	var maxIn float64
	for _, n := range res.Nodes {
		if n.Features.InKZT > maxIn {
			maxIn = n.Features.InKZT
		}
	}
	for i := range res.Nodes {
		if maxIn > 0 {
			res.Nodes[i].PriorityScore = res.Nodes[i].Features.InKZT / maxIn
		}
	}
	res.Top = res.Top[:0]
	for rank, i := range sortedByPriority(res) {
		if rank >= topN {
			break
		}
		n := res.Nodes[i]
		res.Top = append(res.Top, TopNode{Rank: rank + 1, Gid: n.Gid, Role: n.Role, PriorityScore: n.PriorityScore, Why: n.Evidence})
	}
}

func describeClusters(g *graph.Graph, res *Result) {
	c := ClusterResult{ClusterID: 0, RoleCounts: map[string]int{}}
	for _, n := range res.Nodes {
		c.NNodes++
		if n.Features.IsSeed {
			c.NSeed++
		}
		c.RoleCounts[n.Role]++
	}
	for _, e := range res.Edges {
		c.SumKZTInternal += e.SumKZT
	}
	for _, t := range res.Top {
		if len(c.TopGids) < 5 {
			c.TopGids = append(c.TopGids, t.Gid)
		}
	}
	c.Hypothesis = "заглушка: все узлы в одном кластере"
	res.Clusters = []ClusterResult{c}
}
