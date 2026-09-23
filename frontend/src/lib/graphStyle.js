// Стили cytoscape по правилам дизайна. CSS-переменные cytoscape не читает: значения берём из :root.
import {ROLES} from './amlLogic.js';

export function readTokens() {
  const css = getComputedStyle(document.documentElement);
  return name => css.getPropertyValue(name).trim();
}

export function graphStyle(t) {
  const roleRules = Object.entries(ROLES).map(([key, r]) => ({selector: `node[role = "${key}"]`, style: {'background-color': t(r.token)}}));
  return [
    {selector: 'node', style: {
      width: 'mapData(priority, 0, 1, 22, 52)', height: 'mapData(priority, 0, 1, 22, 52)',
      'border-width': 1.5, 'border-color': t('--apx-surface'), label: '', 'font-family': t('--apx-font') || 'system-ui',
      'font-size': 11, 'font-weight': 500, color: t('--apx-ink-2'), 'text-valign': 'bottom', 'text-margin-y': 4,
      'text-outline-color': t('--apx-bg'), 'text-outline-width': 3, 'transition-property': 'opacity', 'transition-duration': 200,
    }},
    ...roleRules,
    {selector: 'node', style: {'z-index': 10}},
    {selector: 'node[role != "peripheral"]', style: {width: 'mapData(priority, 0, 1, 26, 52)', height: 'mapData(priority, 0, 1, 26, 52)'}},
    {selector: 'node[role = "peripheral"]', style: {opacity: 0.45, width: 'mapData(priority, 0, 1, 10, 18)', height: 'mapData(priority, 0, 1, 10, 18)', 'z-index': 1, 'border-width': 1}},
    {selector: 'node[?is_seed]', style: {'z-index': 20}},
    {selector: 'node.hidden', style: {display: 'none'}},
    {selector: 'node.labeled', style: {label: 'data(tail6)'}},
    {selector: 'node[?is_seed]', style: {
      shape: 'diamond', 'background-color': t('--apx-ink'), 'border-color': 'data(roleColor)', 'border-width': 3,
      width: 'mapData(priority, 0, 1, 34, 52)', height: 'mapData(priority, 0, 1, 34, 52)', label: 'data(tail6)', opacity: 1,
    }},
    {selector: 'node[?truncated]', style: {'border-style': 'dashed', 'border-color': t('--apx-ink-2')}},
    {selector: 'node.dim', style: {opacity: 0.28}},
    {selector: 'node.filtered-out', style: {opacity: 0.12}},
    // пройденный узел: номер шага в кружке справа сверху (SVG в data(badge)), текущий — брендовый
    {selector: 'node.route', style: {
      'outline-width': 1, 'outline-color': t('--apx-line-strong'), 'outline-offset': 3,
      'background-image': 'data(badge)', 'background-width': 16, 'background-height': 16, 'background-position-x': '100%', 'background-position-y': '0%',
      'background-offset-x': 6, 'background-offset-y': -6, 'background-clip': 'none', 'background-image-containment': 'over', 'bounds-expansion': 14,
    }},
    {selector: 'node:selected, node.selected', style: {
      'border-width': 3, 'border-color': t('--apx-brand'), 'border-style': 'solid', label: 'data(tail6)', 'font-weight': 600, 'font-size': 12, color: t('--apx-ink'),
      'underlay-color': t('--apx-brand'), 'underlay-opacity': 0.1, 'underlay-padding': 14, 'underlay-shape': 'ellipse', opacity: 1,
    }},
    {selector: 'node.candidate-1', style: {
      'underlay-color': t('--apx-brand-soft'), 'underlay-opacity': 1, 'underlay-padding': 12, 'underlay-shape': 'ellipse',
      'outline-width': 2, 'outline-color': t('--apx-brand'), 'outline-style': 'dashed', 'outline-offset': 7, opacity: 1,
    }},
    {selector: 'node.candidate-2, node.candidate-3', style: {'outline-width': 1.25, 'outline-color': t('--apx-brand'), 'outline-style': 'dashed', 'outline-opacity': 0.6, 'outline-offset': 6, opacity: 1}},
    {selector: 'node.assistant', style: {'outline-width': 1.5, 'outline-color': t('--apx-brand'), 'outline-style': 'dashed', 'outline-offset': 4, 'underlay-color': t('--apx-brand'), 'underlay-opacity': 0.12, 'underlay-padding': 10, opacity: 1}},

    {selector: 'edge', style: {
      width: 'mapData(log_sum, 0, 4, 1, 2.2)', 'line-color': t('--apx-line-strong'), 'curve-style': 'straight',
      'target-arrow-shape': 'triangle', 'target-arrow-color': t('--apx-line-strong'), 'arrow-scale': 0.6, 'z-index': 5,
    }},
    {selector: 'edge.peripheral', style: {opacity: 0.35, width: 1, 'z-index': 1}},
    {selector: 'edge.bg', style: {'target-arrow-shape': 'none', opacity: 0.35, width: 1}},
    {selector: 'edge.in, edge.out', style: {
      'curve-style': 'unbundled-bezier', 'control-point-distances': 40, 'control-point-weights': 0.5,
      width: 'mapData(log_sum, 0, 4, 1.4, 5)', 'arrow-scale': 1, opacity: 0.9, label: 'data(sum_short)',
      'font-family': t('--apx-font') || 'system-ui', 'font-size': 11, 'font-weight': 600, 'text-outline-color': t('--apx-bg'), 'text-outline-width': 3,
    }},
    {selector: 'edge.in', style: {'line-color': t('--apx-brand'), 'target-arrow-color': t('--apx-brand'), color: t('--apx-brand')}},
    {selector: 'edge.out', style: {'line-color': t('--apx-ink'), 'target-arrow-color': t('--apx-ink'), color: t('--apx-ink')}},
    // переводы с периферией у выбранного узла: тусклые, без подписи, чтобы основные связи читались первыми
    {selector: 'edge.in.muted', style: {'line-color': t('--apx-brand'), 'target-arrow-color': t('--apx-brand'), opacity: 0.35, width: 'mapData(log_sum, 0, 4, 1, 2)', label: ''}},
    {selector: 'edge.out.muted', style: {'line-color': t('--apx-line-strong'), 'target-arrow-color': t('--apx-line-strong'), opacity: 0.6, width: 'mapData(log_sum, 0, 4, 1, 2)', label: ''}},
    // накладка с бегущим белым пунктиром поверх входящих/исходящих рёбер выбранного узла (направление денег)
    {selector: 'edge.flow', style: {
      'curve-style': 'unbundled-bezier', 'control-point-distances': 40, 'control-point-weights': 0.5,
      'line-color': t('--apx-bg'), 'line-style': 'dashed', 'line-dash-pattern': [3, 9], width: 'mapData(log_sum, 0, 4, 1, 3)',
      'target-arrow-shape': 'none', opacity: 0.9, 'z-index': 30, events: 'no',
    }},
    // акцент направления с полосы над графом
    {selector: 'edge.emph', style: {opacity: 1, width: 'mapData(log_sum, 0, 4, 2.2, 6.5)', 'z-index': 40}},
    {selector: 'node.emph', style: {opacity: 1, 'z-index': 40}},
    {selector: 'edge.faded', style: {opacity: 0.12, label: ''}},
    {selector: 'node.faded', style: {opacity: 0.2}},
    {selector: 'edge.route', style: {'underlay-color': t('--apx-surface-3'), 'underlay-opacity': 1, 'underlay-padding': 6}},
    {selector: 'edge.hover', style: {label: 'data(sum_short)', 'font-size': 11, 'text-outline-color': t('--apx-bg'), 'text-outline-width': 3}},
  ];
}

export function toElements(graph, t) {
  const peripheralIds = new Set(graph.nodes.filter(n => n.role === 'peripheral' && !n.is_seed).map(n => n.id));
  return [
    ...graph.nodes.map(n => ({data: {...n, tail6: String(n.id).slice(-6), priority: n.priority || 0, roleColor: t((ROLES[n.role] || ROLES.peripheral).token)}})),
    ...graph.edges.map(e => ({data: {...e, id: `edge:${e.source}:${e.target}`, log_sum: Math.max(0, Math.log10(Math.max(1, e.sum_kzt) / 5000)), sum_short: shortSum(e.sum_kzt)},
      classes: peripheralIds.has(e.source) || peripheralIds.has(e.target) ? 'peripheral' : ''})),
  ];
}
const shortSum = n => n >= 1e6 ? (n / 1e6).toFixed(1).replace('.', ',') + ' млн' : Math.round(n / 1000) + ' тыс';

// Кружок с номером шага маршрута для background-image узла.
export function badgeSvg(step, current, t) {
  const fill = current ? t('--apx-brand') : t('--apx-ink'), ink = current ? t('--apx-brand-contrast') : t('--apx-ink-inverse');
  const size = step >= 10 ? 8.5 : 10;
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16"><circle cx="8" cy="8" r="8" fill="${fill}"/><text x="8" y="11.6" text-anchor="middle" font-family="Golos Text, system-ui, sans-serif" font-size="${size}" font-weight="600" fill="${ink}">${step}</text></svg>`;
  return 'data:image/svg+xml;utf8,' + encodeURIComponent(svg);
}
