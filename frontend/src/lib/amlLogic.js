// Чистая логика экрана: форматирование, роли, уровни сигналов по метрикам, маршрут. Без React.

export const ROLES = {
  consolidator: {name: 'Консолидатор', token: '--aml-consolidator'},
  coordinator: {name: 'Координатор', token: '--aml-coordinator'},
  distributor: {name: 'Распределитель', token: '--aml-distributor'},
  transit: {name: 'Транзит', token: '--aml-transit'},
  terminal: {name: 'Конечный получатель', token: '--aml-terminal'},
  peripheral: {name: 'Периферия', token: '--aml-peripheral'},
};
export const roleName = role => ROLES[role]?.name || role || 'Узел';
export const roleColor = role => `var(${(ROLES[role] || ROLES.peripheral).token})`;

const nbsp = ' ';
export function fmtKzt(n) {
  if (!n) return '0 KZT';
  if (n >= 1e6) return (n / 1e6).toFixed(1).replace('.', ',') + ' млн KZT';
  return String(Math.round(n)).replace(/\B(?=(\d{3})+(?!\d))/g, nbsp) + ' KZT';
}
export function fmtShort(n) {
  return n >= 1e6 ? (n / 1e6).toFixed(1).replace('.', ',') + ' млн' : Math.round(n / 1000) + ' тыс';
}
export function plural(n, one, few, many) {
  const m10 = n % 10, m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20)) return few;
  return many;
}
export const tail6 = id => String(id).slice(-6);

// Статистика по карточке узла (входящие/исходящие уже отсортированы по сумме).
export function cardStats(card) {
  const inK = card.incoming.reduce((a, e) => a + e.sum_kzt, 0), outK = card.outgoing.reduce((a, e) => a + e.sum_kzt, 0);
  const pass = inK ? Math.min(100, Math.round(outK / inK * 100)) : 0;
  const seedIn = card.incoming.filter(e => e.is_seed).length;
  const biggest = [...card.incoming.map(e => ({e, dir: 'от', other: e.gid})), ...card.outgoing.map(e => ({e, dir: 'к', other: e.gid}))]
    .sort((a, b) => b.e.sum_kzt - a.e.sum_kzt)[0] || null;
  return {inK, outK, pass, retain: card.incoming.length ? 100 - pass : 0, seedIn, biggest};
}

// Уровни сигнала. Цвета статусов только здесь.
export const LEVELS = {
  high: {token: '--apx-danger', fill: 5, name: 'Сильный сигнал', rank: 0, legend: 'Сильные'},
  mid: {token: '--apx-warn', fill: 3, name: 'Умеренный сигнал', rank: 1, legend: 'Умеренные'},
  ok: {token: '--apx-success', fill: 1, name: 'Снижает подозрение', rank: 2, legend: 'Снижают подозрение'},
  neutral: {token: '--apx-ink-3', fill: 2, name: 'Нейтрально', rank: 3, legend: 'Нейтральные'},
  info: {token: null, fill: 0, name: 'Справочно', rank: 4, legend: 'Справочно'},
};
const byPercentile = (p, capAtMid = false) => {
  const level = p >= 90 ? 'high' : p >= 70 ? 'mid' : 'neutral';
  return capAtMid && level === 'high' ? 'mid' : level;
};
const pctNote = p => p >= 70 ? `выше, чем у ${Math.round(p)}% узлов сети` : '';

// 18 метрик карточки: подпись, значение, уровень и пояснение.
export function metricRows(card) {
  const n = card.node, f = n.features || {}, p = card.percentiles || {}, st = cardStats(card);
  const pct = key => (key in p ? p[key] : null);
  const pctRow = (key, label, value, cap) => {
    const q = pct(key);
    return {key, label, value, level: q == null ? 'neutral' : byPercentile(q, cap), note: q == null ? '' : pctNote(q)};
  };
  const rows = [
    {key: 'role_score', label: 'Уверенность', value: (n.role_score ?? 0).toFixed(3), level: 'info', note: ''},
    {key: 'priority', label: 'Приоритет', value: (n.priority ?? 0).toFixed(3), level: 'info', note: ''},
    {key: 'cluster', label: 'Кластер', value: String(n.cluster), level: 'info', note: ''},
    {key: 'depth', label: 'Глубина', value: String(n.depth), level: n.depth <= 1 ? 'mid' : n.depth >= 3 ? 'ok' : 'neutral',
      note: n.depth === 0 ? 'известный участник из исходного списка' : n.depth === 1 ? 'в одном шаге от известного участника' : n.depth >= 3 ? 'далеко от известных участников' : ''},
    {key: 'is_seed', label: 'Seed', value: n.is_seed ? 'Да' : 'Нет', level: n.is_seed ? 'high' : 'neutral', note: n.is_seed ? 'известный участник из исходного списка' : ''},
    pctRow('in_deg', 'Входящих связей', String(f.in_deg ?? n.in_deg ?? 0)),
    pctRow('out_deg', 'Исходящих связей', String(f.out_deg ?? n.out_deg ?? 0), true),
    pctRow('in_kzt', 'Получено', fmtKzt(f.in_kzt ?? n.in_kzt ?? 0)),
    pctRow('out_kzt', 'Отправлено', fmtKzt(f.out_kzt ?? n.out_kzt ?? 0), true),
    pctRow('in_tx', 'Входящих переводов', String(f.in_tx ?? st.inTx ?? 0)),
    pctRow('out_tx', 'Исходящих переводов', String(f.out_tx ?? 0), true),
    {key: 'pass_through', label: 'Доля транзита', value: card.incoming.length ? st.pass + '%' : 'Нет входящих',
      level: !card.incoming.length ? 'neutral' : (st.pass < 30 && st.inK >= 1e6) ? 'high' : st.pass >= 90 ? 'mid' : 'neutral',
      note: !card.incoming.length ? '' : st.pass >= 90 ? 'почти всё полученное уходит дальше' : `удерживает ${st.retain}% полученного`},
    pctRow('pagerank', 'PageRank', Number((f.pagerank ?? 0).toPrecision(3)).toString()),
    pctRow('betweenness', 'Посредничество', String(Math.round(f.betweenness ?? 0))),
    {key: 'n_seed_upstream', label: 'Seed выше по цепочке', value: String(f.n_seed_upstream ?? 0),
      level: (f.n_seed_upstream ?? 0) >= 2 ? 'high' : (f.n_seed_upstream ?? 0) === 1 ? 'mid' : 'ok',
      note: (f.n_seed_upstream ?? 0) >= 1 ? 'деньги приходят от известного участника' : 'нет переводов от известных участников'},
    {key: 'component_id', label: 'Компонента', value: String(f.component_id ?? n.component ?? 0), level: 'info', note: ''},
    {key: 'truncated', label: 'Обрыв обхода', value: n.truncated ? 'Да' : 'Нет', level: n.truncated ? 'mid' : 'ok',
      note: n.truncated ? 'данные неполные, связи за границей обхода не видны' : 'связи узла собраны полностью'},
    {key: 'verified_sink', label: 'Подтверждённый сток', value: f.verified_sink ? 'Да' : 'Нет', level: f.verified_sink ? 'high' : 'neutral', note: f.verified_sink ? 'деньги пришли и остались' : ''},
    {key: 'structuring', label: 'Дробление сумм', value: f.structuring ? 'Да' : 'Нет',
      level: f.structuring ? 'high' : (f.small_tx_share ?? 0) >= 0.6 ? 'mid' : 'neutral',
      note: (f.in_tx ?? 0) > 0 ? `${Math.round((f.small_tx_share ?? 0) * 100)}% входящих переводов до 15 тыс KZT, сразу над порогом выгрузки` : ''},
  ];
  return rows.sort((a, b) => LEVELS[a.level].rank - LEVELS[b.level].rank);
}

export function signalSummary(rows) {
  return ['high', 'mid', 'neutral', 'ok'].map(level => ({level, n: rows.filter(r => r.level === level).length})).filter(x => x.n > 0);
}

// Разделитель между соседними узлами маршрута по рёбрам текущего графа.
export function routeSeparator(edges, a, b) {
  if (edges.some(e => e.source === a && e.target === b)) return {sign: '→', title: 'перевод к следующему узлу'};
  if (edges.some(e => e.source === b && e.target === a)) return {sign: '←', title: 'перевод от следующего узла'};
  return {sign: '⋯', title: 'прямой связи нет, переход через поиск или список'};
}
export const ROUTE_STORAGE_KEY = 'graf-deneg:route:v1';
export const ROUTE_MAX = 40;

export function candidateWhy(c, route) {
  const hop = c.hops === 1 ? (c.direction === 'down' ? 'прямой получатель' : 'прямой плательщик') : `через ${c.hops} ${plural(c.hops, 'шаг', 'шага', 'шагов')}`;
  const flow = `${c.direction === 'down' ? 'доходит' : 'приходит от него'} ${Math.max(1, Math.round(c.flow_share * 100))}% потока`;
  return `${hop}, ${flow}, приоритет ${c.priority.toFixed(3)}${route.includes(c.gid) ? ', уже в маршруте' : ''}`;
}
