'use strict';
const $ = id => document.getElementById(id);
const roles = {
  consolidator: ['Консолидатор', '#8054d9'], transit: ['Транзит', '#25a9b6'],
  distributor: ['Распределитель', '#ed9650'], terminal: ['Конечный получатель', '#db638c'],
  coordinator: ['Координатор', '#4778e5'], peripheral: ['Периферия', '#9aa8bc']
};
const money = n => new Intl.NumberFormat('ru-RU', {maximumFractionDigits: 0}).format(n || 0) + ' ₸';
let cy, offline, source = '', topOnly = true, topNodes = [], clusters = [];
let requestVersion = 0, searchVersion = 0, llmUnavailable = false;
function el(tag, text, cls) {
  const e = document.createElement(tag); e.textContent = text;
  if (cls) e.className = cls;
  return e;
}
function report(error) { $('status').textContent = error.message; }
async function get(path) {
  const response = await fetch(path);
  if (!response.ok) throw new Error(`Ошибка ${response.status}: ${path}`);
  return response.json();
}
function normalize(graph) {
  if (!Array.isArray(graph.nodes) || !Array.isArray(graph.edges)) throw new Error('Некорректный формат графа');
  for (const n of graph.nodes) {
    n.id = n.id ?? n.gid;
    if (typeof n.id !== 'string') throw new Error('gid должен быть строкой');
  }
  for (const e of graph.edges) if (typeof e.source !== 'string' || typeof e.target !== 'string') throw new Error('gid ребра должен быть строкой');
  return graph;
}
function neighborhood(ids, depth) {
  const found = new Set(ids);
  for (let i = 0; i < depth; i++) {
    const frontier = new Set(found);
    for (const e of offline.edges) if (frontier.has(e.source) || frontier.has(e.target)) { found.add(e.source); found.add(e.target); }
  }
  return {nodes: offline.nodes.filter(n => found.has(n.id)), edges: offline.edges.filter(e => found.has(e.source) && found.has(e.target))};
}
async function ego(id, depth) {
  return offline ? neighborhood([id], depth) : normalize(await get(`/v1/nodes/${encodeURIComponent(id)}/ego?depth=${depth}`));
}
function largestComponent(graph) {
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
function elements(graph) {
  return [...graph.nodes.map(n => ({data: {...n, label: n.id.slice(-6), color: (roles[n.role] || roles.peripheral)[1], size: 22 + 30 * (n.priority || 0), shape: n.is_seed ? 'diamond' : 'ellipse'}})),
    ...graph.edges.map(e => ({data: {...e, id: `edge:${e.source}:${e.target}`, width: 1 + Math.log10(1 + Math.max(0, e.sum_kzt)) / 2, amount: money(e.sum_kzt)}}))];
}
async function render(graph, all = false) {
  $('busy').hidden = false;
  // Даём браузеру отрисовать индикатор до расчёта раскладки.
  await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
  cy.elements().remove(); cy.add(elements(graph));
  const large = all ? largestComponent(graph) : new Set(graph.nodes.map(n => n.id));
  const layoutNodes = cy.nodes().filter(n => large.has(n.id()));
  const layoutEdges = cy.edges().filter(e => large.has(e.source().id()) && large.has(e.target().id()));
  await new Promise(resolve => {
    layoutNodes.union(layoutEdges).layout({name: 'cose', animate: false, fit: false, randomize: false, numIter: 400, nodeRepulsion: 6000, stop: resolve}).run();
  });
  const others = cy.nodes().filter(n => !large.has(n.id()));
  if (others.length) others.layout({name: 'grid', fit: false, boundingBox: {x1: layoutNodes.boundingBox().x2 + 100, y1: 0, w: 800, h: 800}}).run();
  if (layoutNodes.length) cy.fit(layoutNodes, 45); else cy.fit(undefined, 45);
  $('busy').hidden = true;
  $('status').textContent = `${source} · ${graph.nodes.length} узлов · ${graph.edges.length} связей${all ? ' · в фокусе крупнейшая компонента' : ''}`;
}
async function draw() {
  const version = ++requestVersion;
  const role = $('role').value, cluster = $('cluster').value;
  try {
    let graph;
    if (offline) {
      graph = topOnly ? neighborhood(topNodes.slice(0, 30).map(n => n.gid), 1) : offline;
      const nodes = graph.nodes.filter(n => (!role || n.role === role) && (!cluster || String(n.cluster) === cluster));
      const ids = new Set(nodes.map(n => n.id)); graph = {nodes, edges: graph.edges.filter(e => ids.has(e.source) && ids.has(e.target))};
    } else {
      const q = new URLSearchParams({top: topOnly ? '30' : '0', role, cluster});
      graph = normalize(await get('/v1/graph?' + q));
    }
    if (version !== requestVersion) return;
    await render(graph, !topOnly);
  } catch (error) { report(error); $('busy').hidden = true; }
}
function highlight(id) {
  const node = cy.getElementById(id);
  cy.elements().removeClass('dim incoming outgoing').addClass('dim');
  node.closedNeighborhood().removeClass('dim');
  node.incomers('edge').addClass('incoming'); node.outgoers('edge').addClass('outgoing');
  cy.nodes().unselect(); node.select();
  cy.animate({center: {eles: node}, zoom: Math.max(1.2, cy.zoom())}, {duration: 200});
}
async function focus(id, depth = 1) {
  const version = ++requestVersion;
  try {
    let card;
    if (offline) {
      const node = offline.nodes.find(n => n.id === id);
      if (!node) throw new Error('Узел не найден: ' + id);
      card = {node, incoming: offline.edges.filter(e => e.target === id).map(e => ({...e, gid: e.source})), outgoing: offline.edges.filter(e => e.source === id).map(e => ({...e, gid: e.target}))};
    } else card = await get('/v1/nodes/' + encodeURIComponent(id));
    // Догружаем окружение и для уже видимого узла: фильтр мог скрыть его соседей.
    const graph = await ego(id, depth);
    if (version !== requestVersion) return;
    const added = cy.add(elements(graph).filter(e => !cy.getElementById(e.data.id).length));
    if (added.length) {
      const center = cy.getElementById(id).position();
      added.nodes().forEach((node, i) => node.position({x: center.x + Math.cos(i * 2.4) * (100 + i), y: center.y + Math.sin(i * 2.4) * (100 + i)}));
    }
    highlight(id); $('search').value = id; showCard(card);
    $('status').textContent = `${source} · узел ${id} · окружение ${depth} шага`;
  } catch (error) { if (version === requestVersion) report(error); }
}
function showCard(card) {
  const n = card.node, box = $('card');
  box.replaceChildren(el('h2', (roles[n.role] || [n.role])[0]), el('p', n.id, 'gid'), el('p', n.evidence || 'Обоснование отсутствует'));
  const labels = {role_score: 'Уверенность', priority: 'Приоритет', cluster: 'Кластер', depth: 'Глубина', is_seed: 'Seed', in_deg: 'Входящих связей', out_deg: 'Исходящих связей', in_kzt: 'Получено', out_kzt: 'Отправлено', in_tx: 'Входящих переводов', out_tx: 'Исходящих переводов', pass_through: 'Доля транзита', pagerank: 'PageRank', betweenness: 'Посредничество', n_seed_upstream: 'Seed выше по цепочке', component_id: 'Компонента', truncated: 'Обрыв обхода', verified_sink: 'Подтверждённый сток'};
  for (const [key, label] of Object.entries(labels)) {
    let value = n[key]; if (value === undefined) continue;
    if (key.endsWith('kzt')) value = money(value);
    else if (typeof value === 'boolean') value = value ? 'Да' : 'Нет';
    else if (typeof value === 'number' && !Number.isInteger(value)) value = Number(value.toPrecision(4));
    const row = el('div', '', 'metric'); row.append(el('span', label), el('strong', value)); box.append(row);
  }
  for (const [label, entries] of [['Входящие', card.incoming], ['Исходящие', card.outgoing]]) {
    box.append(el('h3', label));
    if (!entries.length) box.append(el('p', 'Нет переводов', 'empty'));
    for (const entry of entries) { const button = el('button', `${entry.gid} · ${money(entry.sum_kzt)} · ${entry.n_tx} переводов`, 'neighbor'); button.onclick = () => focus(entry.gid); box.append(button); }
  }
  if (!offline && !llmUnavailable) {
    const button = el('button', 'Справка'), text = el('p', ''); text.style.whiteSpace = 'pre-wrap';
    button.onclick = async () => {
      button.disabled = true; text.textContent = 'Готовим справку…';
      try {
        const response = await fetch(`/v1/nodes/${encodeURIComponent(n.id)}/card`);
        if ([404, 503].includes(response.status)) { button.remove(); text.remove(); if (response.status === 503) disableLLM(); return; }
        if (!response.ok) throw new Error('Не удалось получить справку');
        const result = await response.json(); text.textContent = typeof result === 'string' ? result : result.text ?? result.card ?? result.answer ?? 'Пустая справка';
      } catch (error) { text.textContent = error.message; } finally { button.disabled = false; }
    };
    button.className = 'llm-card'; box.append(button, text);
  }
  const egoButton = el('button', 'Ego 2 шага'); egoButton.onclick = () => focus(n.id, 2); box.append(egoButton);
  const copy = el('button', 'Скопировать'); copy.onclick = async () => { try { await navigator.clipboard.writeText(box.innerText); copy.textContent = 'Скопировано'; } catch { report(new Error('Браузер не разрешил копирование')); } }; box.append(copy);
}
async function search() {
  const q = $('search').value.trim(); if (!q) return;
  if (!/^\d{1,19}$/.test(q)) { report(new Error('Введите gid или его числовой префикс')); return; }
  try {
    const matches = offline ? offline.nodes.filter(n => n.id.startsWith(q)).slice(0, 10) : await get('/v1/search?' + new URLSearchParams({q, limit: '10'}));
    const exact = matches.find(n => n.id === q);
    if (exact || matches.length === 1) await focus((exact || matches[0]).id);
    else report(new Error(matches.length ? 'Несколько совпадений — выберите gid из подсказок' : 'Узел не найден'));
  } catch (error) { report(error); }
}
let debounce;
$('search').oninput = () => {
  const version = ++searchVersion; clearTimeout(debounce);
  debounce = setTimeout(async () => {
    try {
      const q = $('search').value.trim(); if (!/^\d{1,19}$/.test(q)) return;
      const matches = offline ? offline.nodes.filter(n => n.id.startsWith(q)).slice(0, 10) : await get('/v1/search?' + new URLSearchParams({q, limit: '10'}));
      if (version !== searchVersion) return;
      $('suggestions').replaceChildren(...matches.map(n => { const option = el('option', ''); option.value = n.id; return option; }));
    } catch (error) { report(error); }
  }, 180);
};
async function init() {
  for (const [key, [label, color]] of Object.entries(roles)) {
    const option = el('option', label); option.value = key; $('role').append(option);
    const legend = el('span', label), dot = el('span', '', 'dot'); dot.style.background = color; legend.prepend(dot); $('legend').append(legend);
  }
  let initial;
  if (location.protocol !== 'file:') {
    try {
      [initial, topNodes, clusters] = await Promise.all([get('/v1/graph?top=30'), get('/v1/top?n=30'), get('/v1/clusters')]);
      normalize(initial); source = 'Анализ данных';
    } catch {
      try { offline = normalize(await get('graph.json')); source = 'Локальная выгрузка'; } catch { offline = null; }
    }
  }
  if (!initial && !offline) {
    if (location.protocol !== 'file:') throw new Error('API и graph.json недоступны. Проверьте запуск сервера и пайплайна.');
    offline = normalize(window.MOCK_GRAPH); source = 'ДЕМО · тестовые данные';
  }
  if (offline) { topNodes = offline.top || []; clusters = offline.clusters || []; $('assistant').hidden = true; }
  if (typeof cytoscape !== 'function') throw new Error('Не загрузилась локальная библиотека Cytoscape');
  cy = cytoscape({container: $('graph'), elements: [], style: [
    {selector: 'node', style: {'label': 'data(label)', 'font-size': 10, 'background-color': 'data(color)', width: 'data(size)', height: 'data(size)', shape: 'data(shape)', 'border-width': 2, 'border-color': '#fff', 'text-valign': 'bottom', 'text-margin-y': 5}},
    {selector: 'edge', style: {'curve-style': 'bezier', 'target-arrow-shape': 'triangle', width: 'data(width)', 'line-color': '#b7c2d4', 'target-arrow-color': '#b7c2d4', 'arrow-scale': .9}},
    {selector: '.dim', style: {opacity: .12}},
    {selector: '.incoming', style: {'line-color': '#25a9b6', 'target-arrow-color': '#25a9b6'}},
    {selector: '.outgoing', style: {'line-color': '#ed9650', 'target-arrow-color': '#ed9650'}},
    {selector: 'node:selected', style: {'border-width': 4, 'border-color': '#17243b'}},
    {selector: 'edge.hover', style: {label: 'data(amount)', 'font-size': 11, 'text-background-color': '#fff', 'text-background-opacity': 1}}
  ]});
  cy.on('tap', 'node', e => focus(e.target.id()));
  cy.on('mouseover', 'edge', e => e.target.addClass('hover')); cy.on('mouseout', 'edge', e => e.target.removeClass('hover'));
  cy.on('mouseover', 'node', e => $('graph').title = e.target.id()); cy.on('mouseout', 'node', () => $('graph').title = '');
  for (const n of topNodes) {
    const button = el('button', `${n.rank}. ${n.gid}\n${(roles[n.role] || [n.role])[0]} · ${n.priority.toFixed(3)}\n${n.why}`, 'neighbor'); button.onclick = () => focus(n.gid); $('top-list').append(button);
  }
  for (const c of clusters) {
    const option = el('option', 'Кластер ' + c.id); option.value = c.id; $('cluster').append(option);
    const button = el('button', `Кластер ${c.id} · ${c.n_nodes} узлов · ${money(c.sum_kzt_internal)}\n${c.hypothesis}`, 'neighbor');
    button.onclick = () => { topOnly = false; $('role').value = ''; $('cluster').value = c.id; draw(); }; $('cluster-list').append(button);
  }
  $('find').onclick = search; $('search').onkeydown = e => { if (e.key === 'Enter') search(); };
  $('role').onchange = draw; $('cluster').onchange = draw;
  $('top').onclick = () => { topOnly = true; draw(); }; $('all').onclick = () => { topOnly = false; draw(); };
  $('fit').onclick = () => cy.fit(undefined, 40);
  if (initial && !offline) await render(initial); else await draw();
}
function disableLLM() {
  llmUnavailable = true; $('assistant').hidden = true;
  document.querySelectorAll('.llm-card').forEach(button => button.remove());
}
$('ask-form').onsubmit = async event => {
  event.preventDefault(); const question = $('question').value.trim(); if (!question) return;
  $('ask-submit').disabled = true; $('answer').textContent = 'Готовим ответ…';
  try {
    const response = await fetch('/v1/assistant', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({question})});
    if ([404, 503].includes(response.status)) { disableLLM(); return; }
    if (!response.ok) throw new Error('Не удалось получить ответ ассистента');
    const answer = await response.json(); $('answer').replaceChildren(el('p', answer.answer || 'Пустой ответ'));
    const gids = (answer.gids || []).filter(id => typeof id === 'string' && /^\d{1,19}$/.test(id));
    for (const id of gids) { const button = el('button', id, 'neighbor'); button.onclick = () => focus(id); $('answer').append(button); }
    const graphs = await Promise.all(gids.slice(0, 30).map(id => ego(id, 1).catch(() => null)));
    for (const graph of graphs) if (graph) cy.add(elements(graph).filter(e => !cy.getElementById(e.data.id).length));
    cy.elements().removeClass('dim incoming outgoing').addClass('dim'); cy.nodes().unselect();
    for (const id of gids) cy.getElementById(id).removeClass('dim').select();
    const selected = cy.nodes(':selected'); if (selected.length) cy.fit(selected, 60);
  } catch (error) { $('answer').textContent = error.message; } finally { $('ask-submit').disabled = false; }
};
init().catch(report);
