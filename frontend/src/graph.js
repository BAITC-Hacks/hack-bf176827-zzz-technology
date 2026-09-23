export const roles = {
  consolidator: ['Консолидатор', '#8054d9'], transit: ['Транзит', '#25a9b6'],
  distributor: ['Распределитель', '#ed9650'], terminal: ['Конечный получатель', '#db638c'],
  coordinator: ['Координатор', '#4778e5'], peripheral: ['Периферия', '#9aa8bc'],
};
export const money = value => new Intl.NumberFormat('ru-RU', {maximumFractionDigits: 0}).format(value || 0) + ' ₸';
export const roleName = role => roles[role]?.[0] || role;

export function normalize(graph) {
  if (!Array.isArray(graph.nodes) || !Array.isArray(graph.edges)) throw new Error('Некорректный формат графа');
  const nodes = graph.nodes.map(n => ({...n, id: n.id ?? n.gid}));
  if (nodes.some(n => typeof n.id !== 'string') || graph.edges.some(e => typeof e.source !== 'string' || typeof e.target !== 'string')) {
    throw new Error('gid должен передаваться строкой без потери точности');
  }
  return {...graph, nodes};
}
export function subset(graph, ids) {
  return {...graph, nodes: graph.nodes.filter(n => ids.has(n.id)), edges: graph.edges.filter(e => ids.has(e.source) && ids.has(e.target))};
}
export function neighborhood(graph, ids, depth = 1) {
  const found = new Set(ids);
  for (let i = 0; i < depth; i++) {
    const frontier = new Set(found);
    for (const e of graph.edges) if (frontier.has(e.source) || frontier.has(e.target)) { found.add(e.source); found.add(e.target); }
  }
  return subset(graph, found);
}
export function filterGraph(graph, {role, cluster, topOnly}, top) {
  const base = topOnly ? neighborhood(graph, top.slice(0, 30).map(n => n.gid)) : graph;
  const ids = new Set(base.nodes.filter(n => (!role || n.role === role) && (cluster === '' || String(n.cluster) === cluster)).map(n => n.id));
  return subset(base, ids);
}
export function mergeGraph(graph, extra) {
  const nodes = new Map(graph.nodes.map(n => [n.id, n]));
  const edges = new Map(graph.edges.map(e => [`${e.source}:${e.target}`, e]));
  extra.nodes.forEach(n => nodes.set(n.id, n)); extra.edges.forEach(e => edges.set(`${e.source}:${e.target}`, e));
  return {nodes: [...nodes.values()], edges: [...edges.values()]};
}
export function largestComponent(graph) {
  const adjacent = new Map(graph.nodes.map(n => [n.id, []]));
  for (const e of graph.edges) { adjacent.get(e.source)?.push(e.target); adjacent.get(e.target)?.push(e.source); }
  const seen = new Set(); let largest = [];
  for (const id of adjacent.keys()) {
    if (seen.has(id)) continue;
    const component = [id]; seen.add(id);
    for (let i = 0; i < component.length; i++) for (const next of adjacent.get(component[i]) || []) {
      if (!seen.has(next)) { seen.add(next); component.push(next); }
    }
    if (component.length > largest.length) largest = component;
  }
  return new Set(largest);
}
export function elements(graph) {
  return [...graph.nodes.map(n => ({data: {...n, label: n.id.slice(-6), color: (roles[n.role] || roles.peripheral)[1], size: 22 + 30 * (n.priority || 0), shape: n.is_seed ? 'diamond' : 'ellipse'}})),
    ...graph.edges.map(e => ({data: {...e, id: `edge:${e.source}:${e.target}`, width: 1 + Math.log10(1 + Math.max(0, e.sum_kzt)) / 2, amount: money(e.sum_kzt)}}))];
}
export function offlineCard(graph, id) {
  const node = graph.nodes.find(n => n.id === id);
  if (!node) throw new Error('Узел не найден');
  return {node, incoming: graph.edges.filter(e => e.target === id).map(e => ({...e, gid: e.source})), outgoing: graph.edges.filter(e => e.source === id).map(e => ({...e, gid: e.target}))};
}
