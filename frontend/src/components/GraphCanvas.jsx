import React, {useEffect, useRef} from 'react';
import cytoscape from 'cytoscape';
import {largestComponent} from '../graph.js';
import {badgeSvg, graphStyle, readTokens, toElements} from '../lib/graphStyle.js';

const minZoomPadding = 400;
// Нижняя граница зума: дальше «вместить граф с полем 400px» отдалять нельзя.
function limitZoomOut(cy) {
  const box = cy.elements().boundingBox();
  if (!box.w || !box.h) return;
  const pad = Math.min(minZoomPadding, cy.width() / 4, cy.height() / 4);
  const zoom = Math.min((cy.width() - 2 * pad) / box.w, (cy.height() - 2 * pad) / box.h);
  if (zoom > 0) { cy.minZoom(zoom); if (cy.zoom() < zoom) cy.zoom(zoom); }
}

// Расстояния примерно в 1.5 раза больше базовых cose (idealEdgeLength 32 → 48, отталкивание 6000 → 13500 ∝ квадрату).
const layoutOptions = all => ({name: 'cose', animate: false, fit: false, randomize: true, numIter: all ? 60 : 300, nodeRepulsion: () => 13500, idealEdgeLength: () => 48, componentSpacing: 60});

// Периферия скрывается, кроме seed, выбранного узла, его соседей, маршрута, кандидатов и подсветки ассистента.
function applyPeripheralVisibility(cy, hide, keep) {
  cy.nodes().removeClass('hidden');
  if (!hide) return;
  cy.nodes('[role = "peripheral"][!is_seed]').filter(n => !keep.has(n.id())).addClass('hidden');
}

export default function GraphCanvas({view, selected, highlighted, route, candidates, hidePeripheral, emphasis, onSelect, onBusy}) {
  const container = useRef(null), cyRef = useRef(null), tokens = useRef(null), selectRef = useRef(onSelect), busyRef = useRef(onBusy);
  selectRef.current = onSelect; busyRef.current = onBusy;

  useEffect(() => {
    tokens.current = readTokens();
    const cy = cytoscape({container: container.current, elements: [], style: graphStyle(tokens.current)}); cyRef.current = cy;
    cy.on('tap', 'node', event => selectRef.current(event.target.id()));
    cy.on('mouseover', 'edge', event => event.target.addClass('hover'));
    cy.on('mouseout', 'edge', event => event.target.removeClass('hover'));
    cy.on('mouseover', 'node', event => { container.current.title = event.target.id(); });
    cy.on('mouseout', 'node', () => { container.current.title = ''; });
    const observer = new ResizeObserver(() => { cy.resize(); limitZoomOut(cy); }); observer.observe(container.current);
    // тема: пересобрать стили при смене prefers-color-scheme
    const media = matchMedia('(prefers-color-scheme: dark)');
    const retheme = () => { tokens.current = readTokens(); cy.style(graphStyle(tokens.current)); };
    media.addEventListener('change', retheme);
    // движение потока по рёбрам выбранного узла
    let offset = 0, frame;
    const tick = () => { offset = (offset - 1) % 24; cy.edges('.flow').style('line-dash-offset', offset); frame = requestAnimationFrame(tick); };
    frame = requestAnimationFrame(tick);
    return () => { cancelAnimationFrame(frame); media.removeEventListener('change', retheme); observer.disconnect(); cy.destroy(); cyRef.current = null; };
  }, []);

  useEffect(() => {
    const cy = cyRef.current; if (!cy || !view) return;
    let layout, cancelled = false, frame;
    function run() {
      if (cancelled) return;
      if (view.append) {
        const center = cy.nodes(':selected').first().position() || {x: 0, y: 0};
        const added = cy.add(toElements(view.graph, tokens.current).filter(e => !cy.getElementById(e.data.id).length));
        added.nodes().forEach((n, i) => n.position({x: center.x + Math.cos(i * 2.4) * (180 + i * 3), y: center.y + Math.sin(i * 2.4) * (180 + i * 3)}));
        limitZoomOut(cy); busyRef.current(false); return;
      }
      cy.elements().remove(); cy.minZoom(1e-3); cy.add(toElements(view.graph, tokens.current));
      if (!cy.nodes().length) { busyRef.current(false); return; }
      const ids = view.all ? largestComponent(view.graph) : new Set(view.graph.nodes.map(n => n.id));
      const nodes = cy.nodes().filter(n => ids.has(n.id()));
      const edges = cy.edges().filter(e => ids.has(e.source().id()) && ids.has(e.target().id()));
      layout = nodes.union(edges).layout({...layoutOptions(view.all), stop: () => {
        if (cancelled) return;
        const others = cy.nodes().filter(n => !ids.has(n.id()));
        if (others.length) others.layout({name: 'grid', fit: false, boundingBox: {x1: nodes.boundingBox().x2 + 100, y1: 0, w: 1000, h: 1000}}).run();
        limitZoomOut(cy); cy.fit(nodes, 45); busyRef.current(false);
      }});
      layout.run();
    }
    busyRef.current(true);
    frame = requestAnimationFrame(() => { frame = requestAnimationFrame(run); });
    return () => { cancelled = true; cancelAnimationFrame(frame); layout?.stop(); };
  }, [view]);

  // подсветка: фокус на выбранном, кандидаты, ответ ассистента, маршрут, подписи
  useEffect(() => {
    const cy = cyRef.current; if (!cy) return;
    let frame = requestAnimationFrame(() => { frame = requestAnimationFrame(() => {
      cy.edges('.flow').remove();
      cy.elements().removeClass('dim bg in out muted route labeled candidate-1 candidate-2 candidate-3 assistant'); cy.nodes().unselect();
      const keep = new Set([...route, ...highlighted, ...candidates.map(c => c.gid)]);
      if (selected) keep.add(selected); // при исключении периферии контрагенты-периферия тоже скрываются
      applyPeripheralVisibility(cy, hidePeripheral, keep);
      cy.nodes().filter(n => n.data('priority') > 0.6 || n.data('is_seed') || route.includes(n.id())).addClass('labeled');
      route.forEach((id, i) => { const el = cy.getElementById(id); if (el.length) el.addClass('route').data('badge', badgeSvg(i + 1, id === selected, tokens.current)); });
      for (let i = 1; i < route.length; i++) {
        cy.edges().filter(e => (e.source().id() === route[i - 1] && e.target().id() === route[i]) || (e.source().id() === route[i] && e.target().id() === route[i - 1])).addClass('route');
      }
      if (highlighted.length) {
        cy.nodes().addClass('dim'); cy.edges().addClass('bg');
        highlighted.forEach(id => cy.getElementById(id).removeClass('dim').addClass('assistant labeled'));
        cy.nodes('[?is_seed]').removeClass('dim');
        if (cy.nodes('.assistant').length) cy.fit(cy.nodes('.assistant'), 70);
      } else if (selected && cy.getElementById(selected).length) {
        const node = cy.getElementById(selected);
        cy.nodes().addClass('dim'); cy.edges().addClass('bg');
        node.closedNeighborhood().removeClass('dim');
        node.incomers('edge').removeClass('bg').addClass('in'); node.outgoers('edge').removeClass('bg').addClass('out');
        const isPeripheral = n => n.data('role') === 'peripheral' && !n.data('is_seed');
        node.incomers('edge').filter(e => isPeripheral(e.source())).addClass('muted');
        node.outgoers('edge').filter(e => isPeripheral(e.target())).addClass('muted');
        cy.add(node.connectedEdges('.in, .out').not('.muted').map(e => ({group: 'edges', classes: 'flow',
          data: {id: 'flow:' + e.id(), source: e.source().id(), target: e.target().id(), log_sum: e.data('log_sum')}})));
        cy.nodes('[?is_seed], .route').removeClass('dim');
        candidates.forEach((c, i) => { const el = cy.getElementById(c.gid); if (el.length) el.removeClass('dim').addClass(`candidate-${i + 1} labeled`); });
        node.select();
        cy.animate({center: {eles: node}, zoom: Math.max(1.2, cy.zoom())}, {duration: 200});
      }
    }); });
    return () => cancelAnimationFrame(frame);
  }, [view, selected, highlighted, route, candidates, hidePeripheral]);

  // акцент направления с полосы выбранного узла: одно направление поверх, другое гасится
  useEffect(() => {
    const cy = cyRef.current; if (!cy) return;
    cy.elements().removeClass('emph faded');
    if (!emphasis || !selected) return;
    const node = cy.getElementById(selected); if (!node.length) return;
    const keep = emphasis === 'in' ? node.incomers('edge') : node.outgoers('edge');
    const fade = emphasis === 'in' ? node.outgoers('edge') : node.incomers('edge');
    keep.addClass('emph'); keep.connectedNodes().not(node).addClass('emph');
    fade.addClass('faded'); fade.connectedNodes().not(node).not(keep.connectedNodes()).addClass('faded');
    cy.edges('.flow').filter(e => fade.some(f => 'flow:' + f.id() === e.id())).addClass('faded');
  }, [emphasis, selected, view]);

  return <div ref={container} id="graph" role="img" aria-label="Сеть направленных переводов" data-nodes={view?.graph.nodes.length || 0} data-edges={view?.graph.edges.length || 0} />;
}
