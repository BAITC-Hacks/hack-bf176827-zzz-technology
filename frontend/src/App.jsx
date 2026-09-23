import React, {useCallback, useEffect, useRef, useState} from 'react';
import GraphCanvas from './GraphCanvas.jsx';
import NodeCard from './NodeCard.jsx';
import Assistant from './Assistant.jsx';
import {request} from './api.js';
import {filterGraph, mergeGraph, money, neighborhood, normalize, offlineCard, roleName, roles} from './graph.js';
import mock from './mock.json';

export default function App() {
  const [view, setView] = useState(null), [top, setTop] = useState([]), [clusters, setClusters] = useState([]);
  const [filters, setFilters] = useState({role: '', cluster: '', topOnly: true});
  const [query, setQuery] = useState(''), [suggestions, setSuggestions] = useState([]), [card, setCard] = useState(null);
  const [source, setSource] = useState('Загрузка данных'), [error, setError] = useState(''), [busy, setBusy] = useState(true);
  const [selected, setSelected] = useState(''), [highlighted, setHighlighted] = useState([]), [fitTick, setFitTick] = useState(0), [llm, setLLM] = useState(true);
  const [tab, setTab] = useState('node'), [ready, setReady] = useState(false);
  const fallback = useRef(null), version = useRef(0), topRef = useRef([]), activeFilters = useRef(filters);
  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        let graph, ranks, groups;
        if (new URLSearchParams(location.search).has('demo')) {
          const data = normalize(mock); fallback.current = data; graph = filterGraph(data, filters, data.top); ranks = data.top; groups = data.clusters;
          if (!cancelled) { setSource('ДЕМО · тестовые данные'); setLLM(false); }
        } else {
          try {
            [graph, ranks, groups] = await Promise.all([request('/v1/graph?top=30'), request('/v1/top?n=30'), request('/v1/clusters')]); graph = normalize(graph);
            if (!cancelled) setSource('Анализ данных');
          } catch {
            const data = normalize(await request('/graph.json')); fallback.current = data; ranks = data.top || []; groups = data.clusters || []; graph = filterGraph(data, filters, ranks);
            if (!cancelled) { setSource('Локальная выгрузка · API недоступен'); setLLM(false); }
          }
        }
        if (cancelled) return;
        topRef.current = ranks; setTop(ranks); setClusters(groups); setView({graph, append: false, all: false}); setReady(true);
      } catch (e) { if (!cancelled) { setError('Не удалось загрузить API или graph.json: ' + e.message); setBusy(false); } }
    }
    load(); return () => { cancelled = true; version.current++; };
  }, []);
  useEffect(() => {
    if (!ready || activeFilters.current === filters) return;
    activeFilters.current = filters; const current = ++version.current; setBusy(true); setError('');
    async function load() {
      try {
        const graph = fallback.current ? filterGraph(fallback.current, filters, topRef.current) : normalize(await request('/v1/graph?' + new URLSearchParams({role: filters.role, cluster: filters.cluster, top: filters.topOnly ? '30' : '0'})));
        if (current !== version.current) return;
        setSelected(''); setHighlighted([]); setCard(null); setView({graph, append: false, all: !filters.topOnly});
      } catch (e) { if (current === version.current) { setError(e.message); setBusy(false); } }
    }
    load();
  }, [filters, ready]);
  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(async () => {
      if (!/^\d{1,19}$/.test(query)) { setSuggestions([]); return; }
      try {
        const results = fallback.current ? fallback.current.nodes.filter(n => n.id.startsWith(query)).slice(0, 10) : await request('/v1/search?' + new URLSearchParams({q: query, limit: '10'}));
        if (!cancelled) setSuggestions(results);
      } catch { if (!cancelled) setSuggestions([]); }
    }, 180);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [query]);
  const getEgo = useCallback(async (id, depth = 1) => fallback.current ? neighborhood(fallback.current, [id], depth) : normalize(await request(`/v1/nodes/${encodeURIComponent(id)}/ego?depth=${depth}`)), []);
  const select = useCallback(async (id, depth = 1) => {
    const current = ++version.current; setError(''); setBusy(true);
    try {
      const [nextCard, graph] = await Promise.all([fallback.current ? offlineCard(fallback.current, id) : request(`/v1/nodes/${encodeURIComponent(id)}`), getEgo(id, depth)]);
      if (current !== version.current) return;
      setView(previous => ({graph: mergeGraph(previous?.graph || {nodes: [], edges: []}, graph), append: true, all: false}));
      setCard(nextCard); setQuery(id); setSelected(id); setHighlighted([]); setTab('node');
    } catch (e) { if (current === version.current) { setError(e.message); setBusy(false); } }
  }, [getEgo]);
  async function search(event) {
    event.preventDefault(); const q = query.trim();
    if (!/^\d{1,19}$/.test(q)) { setError('Введите gid или его числовой префикс'); return; }
    try {
      const results = fallback.current ? fallback.current.nodes.filter(n => n.id.startsWith(q)).slice(0, 10) : await request('/v1/search?' + new URLSearchParams({q, limit: '10'}));
      const exact = results.find(n => n.id === q);
      if (exact || results.length === 1) await select((exact || results[0]).id);
      else { setSuggestions(results); setError(results.length ? 'Несколько совпадений — выберите полный gid из подсказок' : 'Узел не найден'); }
    } catch (e) { setError(e.message); }
  }
  async function highlight(gids) {
    const current = ++version.current;
    const graphs = await Promise.all(gids.map(id => getEgo(id).catch(() => null)));
    if (current !== version.current) return;
    setView(previous => ({graph: graphs.filter(Boolean).reduce(mergeGraph, previous?.graph || {nodes: [], edges: []}), append: true, all: false}));
    setHighlighted(gids); setSelected('');
  }
  return <div className="app">
    <header><div className="heading"><div><div className="eyebrow">Исследование транзакционной сети</div><h1>Граф денег</h1></div><span className="source">{source}</span></div>
      <div className="toolbar"><form onSubmit={search} className="search-form"><input id="search" aria-label="Поиск по gid" placeholder="Полный gid или начало номера" value={query} onChange={e => setQuery(e.target.value)} list="suggestions" autoComplete="off" /><datalist id="suggestions">{suggestions.map(n => <option key={n.id} value={n.id}>{roleName(n.role)}</option>)}</datalist><button className="primary" disabled={!ready}>Найти</button></form>
        <select id="role" aria-label="Роль" value={filters.role} onChange={e => setFilters({...filters, role: e.target.value})} disabled={!ready}><option value="">Все роли</option>{Object.entries(roles).map(([key, [label]]) => <option key={key} value={key}>{label}</option>)}</select>
        <select id="cluster" aria-label="Кластер" value={filters.cluster} onChange={e => setFilters({...filters, cluster: e.target.value})} disabled={!ready}><option value="">Все кластеры</option>{clusters.map(c => <option key={c.id} value={String(c.id)}>Кластер {c.id}</option>)}</select>
        <button id="top" aria-pressed={filters.topOnly} onClick={() => setFilters({...filters, topOnly: true})} disabled={!ready}>Топ-30</button><button id="all" aria-pressed={!filters.topOnly} onClick={() => setFilters({...filters, topOnly: false})} disabled={!ready}>Вся сеть</button><button onClick={() => setFitTick(n => n + 1)} disabled={!ready}>Вместить граф</button>
      </div>
      <div className="legend">{Object.entries(roles).map(([role, [label, color]]) => <span key={role}><i style={{background: color}} />{label}</span>)}<span>◇ Seed</span></div>
    </header>
    <main><div className="canvas-wrap"><GraphCanvas view={view} selected={selected} highlighted={highlighted} fitTick={fitTick} onSelect={select} onBusy={setBusy} />{busy && <div id="busy" role="status">Загружаем и раскладываем сеть…</div>}<div id="status" role="status">{error ? <span className="error">{error}</span> : <>{view?.graph.nodes.length || 0} узлов · {view?.graph.edges.length || 0} связей{view?.all && ' · в фокусе крупнейшая компонента'}{selected && ` · выбран ${selected}`}</>}</div>{ready && !view?.graph.nodes.length && <div className="empty-graph">По выбранным фильтрам узлов нет</div>}</div>
      <aside><nav className="tabs" aria-label="Панель анализа">{[['node', 'Узел'], ['top', 'Топ-30'], ['clusters', 'Кластеры']].map(([key, label]) => <button key={key} aria-pressed={tab === key} onClick={() => setTab(key)}>{label}</button>)}</nav>
        {tab === 'node' && <NodeCard key={card?.node.id || 'empty'} card={card} onSelect={select} llm={llm} onDisableLLM={() => setLLM(false)} />}
        {tab === 'top' && <section id="top-list"><h2>Топ приоритетов</h2>{top.map(n => <button className="result" key={n.gid} onClick={() => select(n.gid)}><span className="rank">{n.rank}</span><span><span className="gid">{n.gid}</span><strong>{roleName(n.role)} · {n.priority.toFixed(3)}</strong><small>{n.why}</small></span></button>)}</section>}
        {tab === 'clusters' && <section id="cluster-list"><h2>Кластеры <span className="count">{clusters.length}</span></h2>{clusters.map(c => <button className="neighbor" key={c.id} onClick={() => setFilters({role: '', cluster: String(c.id), topOnly: false})}><strong>Кластер {c.id} · {c.n_nodes} узлов</strong><small>{money(c.sum_kzt_internal)} · {c.n_seed} seed</small><small>{c.hypothesis}</small></button>)}</section>}
        {llm && ready && <Assistant onDisable={() => setLLM(false)} onHighlight={highlight} onSelect={select} />}
      </aside>
    </main>
  </div>;
}
