import React, {useCallback, useEffect, useMemo, useRef, useState} from 'react';
import Toolbar from './components/Toolbar.jsx';
import SelectedStrip from './components/SelectedStrip.jsx';
import RouteBar from './components/RouteBar.jsx';
import GraphCanvas from './components/GraphCanvas.jsx';
import SidePanel from './components/SidePanel.jsx';
import {request} from './api.js';
import {filterGraph, mergeGraph, neighborhood, normalize, offlineCard} from './graph.js';
import {useRoute} from './hooks/useRoute.js';
import mock from './mock.json';

const SOURCES = {api: 'Анализ данных', offline: 'Локальная выгрузка · API недоступен', demo: 'ДЕМО · тестовые данные'};

export default function App() {
  const [view, setView] = useState(null), [top, setTop] = useState([]), [clusters, setClusters] = useState([]), [seeds, setSeeds] = useState([]), [robustness, setRobustness] = useState([]);
  const [filters, setFilters] = useState({role: '', cluster: '', topOnly: true});
  const [query, setQuery] = useState(''), [suggestions, setSuggestions] = useState([]), [card, setCard] = useState(null);
  const [sourceKind, setSourceKind] = useState('loading'), [error, setError] = useState(''), [searchError, setSearchError] = useState(''), [busy, setBusy] = useState(true);
  const [selected, setSelected] = useState(''), [highlighted, setHighlighted] = useState([]), [llm, setLLM] = useState(true);
  const [tab, setTab] = useState('node'), [ready, setReady] = useState(false), [hidePeripheral, setHidePeripheral] = useState(false);
  const {route, push: pushRoute, clear: clearRoute} = useRoute();
  const fallback = useRef(null), version = useRef(0), topRef = useRef([]), activeFilters = useRef(filters), restored = useRef(false);
  const pendingSelection = useRef(null); // выбор, который надо сохранить при смене фильтра на кластер узла
  const [toast, setToast] = useState(null), toastTimer = useRef(null);
  // toast: {text, action?: {label, run}}; с действием держится дольше
  const notify = useCallback((text, action) => { setToast({text, action}); clearTimeout(toastTimer.current); toastTimer.current = setTimeout(() => setToast(null), action ? 6000 : 2500); }, []);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        let graph, ranks, groups;
        if (new URLSearchParams(location.search).has('demo')) {
          const data = normalize(mock); fallback.current = data; graph = filterGraph(data, filters, data.top); ranks = data.top; groups = data.clusters;
          if (!cancelled) { setSourceKind('demo'); setLLM(false); }
        } else {
          try {
            [graph, ranks, groups] = await Promise.all([request('/v1/graph?top=30'), request('/v1/top?n=30'), request('/v1/clusters')]); graph = normalize(graph);
            const [seedList, resil, status] = await Promise.all([request('/v1/seeds').catch(() => []), request('/v1/robustness').catch(() => []), request('/v1/assistant/status').catch(() => ({enabled: false}))]);
            if (!cancelled) { setSeeds(seedList); setRobustness(resil); setLLM(!!status.enabled); setSourceKind('api'); }
          } catch {
            const data = normalize(await request('/graph.json')); fallback.current = data; ranks = data.top || []; groups = data.clusters || []; graph = filterGraph(data, filters, ranks);
            if (!cancelled) { setSourceKind('offline'); setLLM(false); setRobustness(data.robustness || []); setSeeds(data.nodes.filter(n => n.is_seed).map(n => ({gid: n.id, role: n.role, cluster: n.cluster, out_kzt: n.out_kzt, next: []}))); }
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
        let graph = fallback.current ? filterGraph(fallback.current, filters, topRef.current) : normalize(await request('/v1/graph?' + new URLSearchParams({role: filters.role, cluster: filters.cluster, top: filters.topOnly ? '30' : '0'})));
        if (current !== version.current) return;
        const pending = pendingSelection.current; pendingSelection.current = null;
        if (pending) { graph = mergeGraph(graph, pending.ego); setView({graph, append: false, all: !filters.topOnly}); return; }
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
    const current = ++version.current; setError(''); setSearchError(''); setBusy(true);
    try {
      const [nextCard, graph] = await Promise.all([fallback.current ? offlineCard(fallback.current, id) : request(`/v1/nodes/${encodeURIComponent(id)}`), getEgo(id, depth)]);
      if (current !== version.current) return;
      setCard(nextCard); setQuery(id); setSelected(id); setHighlighted([]); setTab('node'); pushRoute(id);
      const cluster = String(nextCard.node.cluster), active = activeFilters.current.cluster;
      const goToCluster = () => { pendingSelection.current = {id, ego: graph}; setFilters(f => ({...f, cluster, topOnly: false})); notify(`Выбран кластер ${cluster}`); };
      if (active === '') {
        // фильтра нет: кластер узла становится фильтром, режим «Вся сеть», выбор сохраняется
        goToCluster();
      } else {
        setView(previous => ({graph: mergeGraph(previous?.graph || {nodes: [], edges: []}, graph), append: true, all: false}));
        // фильтр задан, узел из другого кластера: вид не меняем, предлагаем перейти
        if (cluster !== active) notify(`Узел из кластера ${cluster}, показан с окружением`, {label: `Перейти в кластер ${cluster}`, run: goToCluster});
      }
    } catch (e) { if (current === version.current) { setError(e.message); setBusy(false); } }
  }, [getEgo, pushRoute, notify]);

  // восстановить последний узел маршрута после загрузки
  useEffect(() => { if (ready && !restored.current) { restored.current = true; if (route.length) select(route[route.length - 1]); } }, [ready, route, select]);

  async function search(event) {
    event.preventDefault(); const q = query.trim();
    if (!/^\d{1,19}$/.test(q)) { setSearchError('Введите gid или его числовой префикс'); return; }
    try {
      const results = fallback.current ? fallback.current.nodes.filter(n => n.id.startsWith(q)).slice(0, 10) : await request('/v1/search?' + new URLSearchParams({q, limit: '10'}));
      const exact = results.find(n => n.id === q);
      if (exact || results.length === 1) await select((exact || results[0]).id);
      else { setSuggestions(results); setSearchError(results.length ? 'Несколько совпадений. Выберите полный gid из подсказок' : 'Узел не найден'); }
    } catch (e) { setSearchError(e.message); }
  }
  async function highlight(gids) {
    const current = ++version.current;
    const graphs = await Promise.all(gids.map(id => getEgo(id).catch(() => null)));
    if (current !== version.current) return;
    setView(previous => ({graph: graphs.filter(Boolean).reduce(mergeGraph, previous?.graph || {nodes: [], edges: []}), append: true, all: false}));
    setHighlighted(gids); setSelected('');
  }
  function back() { const i = route.indexOf(selected); if (i > 0) select(route[i - 1]); }
  function clear() { clearRoute(); setSelected(''); setCard(null); setHighlighted([]); }

  const rolesById = useMemo(() => Object.fromEntries((view?.graph.nodes || []).map(n => [n.id, n.role])), [view]);
  const nodesCount = view?.graph.nodes.length || 0, edgesCount = view?.graph.edges.length || 0;
  const status = error ? '' : `${nodesCount} узлов, ${edgesCount} связей${view?.all ? ', в фокусе крупнейшая компонента' : ''}`;

  return <div className="app">
    <Toolbar source={SOURCES[sourceKind] || 'Загрузка данных'} sourceKind={sourceKind} query={query} onQuery={v => { setQuery(v); setSearchError(''); }} onSearch={search}
      suggestions={suggestions} searchError={searchError || error} filters={filters} onFilters={setFilters} clusters={clusters} ready={ready}
      hidePeripheral={hidePeripheral} />
    <main className="body">
      <div className="canvas-col">
        <SelectedStrip card={card} status={status} />
        <div className="canvas-wrap">
          <GraphCanvas view={view} selected={selected} highlighted={highlighted} route={route} candidates={card?.next_candidates || []} hidePeripheral={hidePeripheral} onSelect={select} onBusy={setBusy} />
          <label className="canvas-toggle" title="Периферия: узлы без признаков роли, 84% сети. Seed, выбранный узел и маршрут остаются видны.">
            <input type="checkbox" checked={hidePeripheral} onChange={e => { setHidePeripheral(e.target.checked); notify(e.target.checked ? 'Периферия исключена' : 'Периферия показана'); }} disabled={!ready} /><span className="switch" aria-hidden="true" /><span>Исключить периферию</span>
          </label>
          <div className="canvas-legend"><span><i className="line line--brand" />Входящие</span><span><i className="line line--ink" />Исходящие</span><span><i className="line line--route" />Маршрут</span><span><i className="line line--cand" />Следующий кандидат</span></div>
          <RouteBar route={route} selected={selected} edges={view?.graph.edges || []} rolesById={rolesById} onSelect={select} onBack={back} onClear={clear} />
          {busy && <div className="overlay" role="status"><span className="apx-progress overlay__progress"><span className="apx-progress__bar" /></span><span className="apx-sm">Загружаем и раскладываем сеть…</span></div>}
          {ready && !busy && !nodesCount && <div className="overlay overlay--empty"><h3 className="apx-h3">По фильтрам узлов нет</h3><p className="apx-hint">Смените роль или кластер, либо переключитесь на всю сеть.</p><button className="apx-btn apx-btn--secondary" onClick={() => setFilters({role: '', cluster: '', topOnly: false})}>Сбросить фильтры</button></div>}
        </div>
      </div>
      <SidePanel tab={tab} onTab={setTab} card={card} route={route} llm={llm} selected={selected} seeds={seeds} top={top} clusters={clusters} robustness={robustness} ready={ready} hidePeripheral={hidePeripheral}
        onSelect={select} onEgo={id => select(id, 2)} onCluster={id => { setFilters({role: '', cluster: String(id), topOnly: false}); notify(`Выбран кластер ${id}`); }} onDisableLLM={() => setLLM(false)} onHighlight={highlight} />
    </main>
    {toast && <div className="toast-host"><div className="apx-toast" role="status"><span className="apx-toast__text">{toast.text}</span>
      {toast.action && <button className="toast__action" onClick={() => { toast.action.run(); setToast(null); }}>{toast.action.label}</button>}</div></div>}
  </div>;
}
