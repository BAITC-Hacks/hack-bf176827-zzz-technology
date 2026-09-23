import {test} from 'node:test';
import assert from 'node:assert/strict';
import {filterGraph, largestComponent, mergeGraph, neighborhood, normalize, offlineCard} from './graph.js';
const ids = ['100000003684369100', '100000003684369101', '100000003684369102', '100000003684369103'];
const graph = {nodes: ids.map((id, i) => ({id, role: i ? 'terminal' : 'transit', cluster: i ? 1 : 0})), edges: [{source: ids[0], target: ids[1], sum_kzt: 123, n_tx: 2}, {source: ids[1], target: ids[2], sum_kzt: 456, n_tx: 3}]};
test('gid сохраняется строкой и числовой gid отклоняется', () => {
  assert.equal(normalize(graph).nodes[0].id, ids[0]);
  assert.throws(() => normalize({...graph, nodes: [{id: 100000003684369100}]}), /строкой/);
});
test('окружение включает оба направления и ограничивает глубину', () => {
  assert.equal(neighborhood(graph, [ids[0]], 1).nodes.length, 2);
  assert.equal(neighborhood(graph, [ids[0]], 2).nodes.length, 3);
  assert.equal(neighborhood(graph, [ids[2]], 1).nodes.length, 2);
});
test('кластер 0 и фильтры не теряются, нет висячих рёбер', () => {
  const result = filterGraph(graph, {cluster: '0', role: '', topOnly: false}, []);
  assert.equal(result.nodes.length, 1); assert.equal(result.edges.length, 0);
});
test('дозагрузка не дублирует узлы и рёбра', () => assert.deepEqual(mergeGraph(graph, neighborhood(graph, [ids[0]], 2)), graph));
test('крупнейшая компонента исключает изолированный узел', () => assert.deepEqual([...largestComponent(graph)], ids.slice(0, 3)));
test('карточка точно сохраняет направление и сумму', () => {
  const card = offlineCard(graph, ids[1]); assert.equal(card.incoming[0].gid, ids[0]); assert.equal(card.outgoing[0].sum_kzt, 456);
});
