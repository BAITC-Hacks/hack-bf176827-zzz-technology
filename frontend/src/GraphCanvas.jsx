import React, {useEffect, useRef} from 'react';
import cytoscape from 'cytoscape';
import {elements, largestComponent} from './graph.js';

const style = [
  {selector: 'node', style: {label: 'data(label)', 'font-size': 10, 'background-color': 'data(color)', width: 'data(size)', height: 'data(size)', shape: 'data(shape)', 'border-width': 2, 'border-color': '#fff', 'text-valign': 'bottom', 'text-margin-y': 5}},
  {selector: 'edge', style: {'curve-style': 'bezier', 'target-arrow-shape': 'triangle', width: 'data(width)', 'line-color': '#b7c2d4', 'target-arrow-color': '#b7c2d4', 'arrow-scale': .9}},
  {selector: '.dim', style: {opacity: .12}},
  {selector: '.incoming', style: {'line-color': '#25a9b6', 'target-arrow-color': '#25a9b6'}},
  {selector: '.outgoing', style: {'line-color': '#ed9650', 'target-arrow-color': '#ed9650'}},
  {selector: 'node:selected', style: {'border-width': 4, 'border-color': '#17243b'}},
  {selector: 'edge.hover', style: {label: 'data(amount)', 'font-size': 11, 'text-background-color': '#fff', 'text-background-opacity': 1}},
];
export default function GraphCanvas({view, selected, highlighted, fitTick, onSelect, onBusy}) {
  const container = useRef(null), cyRef = useRef(null), selectRef = useRef(onSelect), busyRef = useRef(onBusy);
  selectRef.current = onSelect; busyRef.current = onBusy;
  useEffect(() => {
    const cy = cytoscape({container: container.current, elements: [], style}); cyRef.current = cy;
    cy.on('tap', 'node', event => selectRef.current(event.target.id()));
    cy.on('mouseover', 'edge', event => event.target.addClass('hover'));
    cy.on('mouseout', 'edge', event => event.target.removeClass('hover'));
    cy.on('mouseover', 'node', event => { container.current.title = event.target.id(); });
    cy.on('mouseout', 'node', () => { container.current.title = ''; });
    const observer = new ResizeObserver(() => cy.resize()); observer.observe(container.current);
    return () => { observer.disconnect(); cy.destroy(); cyRef.current = null; };
  }, []);
  useEffect(() => {
    const cy = cyRef.current; if (!cy || !view) return;
    let layout, cancelled = false, frame;
    function run() {
      if (cancelled) return;
      if (view.append) {
        const center = cy.nodes(':selected').first().position() || {x: 0, y: 0};
        const added = cy.add(elements(view.graph).filter(e => !cy.getElementById(e.data.id).length));
        added.nodes().forEach((n, i) => n.position({x: center.x + Math.cos(i * 2.4) * (120 + i * 2), y: center.y + Math.sin(i * 2.4) * (120 + i * 2)}));
        busyRef.current(false); return;
      }
      cy.elements().remove(); cy.add(elements(view.graph));
      if (!cy.nodes().length) { busyRef.current(false); return; }
      const ids = view.all ? largestComponent(view.graph) : new Set(view.graph.nodes.map(n => n.id));
      const nodes = cy.nodes().filter(n => ids.has(n.id()));
      const edges = cy.edges().filter(e => ids.has(e.source().id()) && ids.has(e.target().id()));
      layout = nodes.union(edges).layout({name: 'cose', animate: false, fit: false, randomize: true, numIter: view.all ? 60 : 300, nodeRepulsion: 6000, stop: () => {
        if (cancelled) return;
        const others = cy.nodes().filter(n => !ids.has(n.id()));
        if (others.length) others.layout({name: 'grid', fit: false, boundingBox: {x1: nodes.boundingBox().x2 + 100, y1: 0, w: 1000, h: 1000}}).run();
        cy.fit(nodes, 45); busyRef.current(false);
      }});
      layout.run();
    }
    busyRef.current(true);
    frame = requestAnimationFrame(() => { frame = requestAnimationFrame(run); });
    return () => { cancelled = true; cancelAnimationFrame(frame); layout?.stop(); };
  }, [view]);
  useEffect(() => {
    const cy = cyRef.current; if (!cy) return;
    // После добавления окружения подсвечиваем узел на следующем кадре.
    let frame = requestAnimationFrame(() => { frame = requestAnimationFrame(() => {
      cy.elements().removeClass('dim incoming outgoing'); cy.nodes().unselect();
      if (highlighted.length) {
        cy.elements().addClass('dim');
        highlighted.forEach(id => cy.getElementById(id).removeClass('dim').select());
        if (cy.nodes(':selected').length) cy.fit(cy.nodes(':selected'), 70);
      } else if (selected && cy.getElementById(selected).length) {
        const node = cy.getElementById(selected); cy.elements().addClass('dim');
        node.closedNeighborhood().removeClass('dim'); node.incomers('edge').addClass('incoming'); node.outgoers('edge').addClass('outgoing'); node.select();
        cy.animate({center: {eles: node}, zoom: Math.max(1.2, cy.zoom())}, {duration: 200});
      }
    }); });
    return () => cancelAnimationFrame(frame);
  }, [view, selected, highlighted]);
  useEffect(() => { if (fitTick) cyRef.current?.fit(undefined, 40); }, [fitTick]);
  return <div ref={container} id="graph" role="img" aria-label="Сеть направленных переводов" data-nodes={view?.graph.nodes.length || 0} data-edges={view?.graph.edges.length || 0} />;
}
