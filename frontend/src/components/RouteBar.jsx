import React from 'react';
import {roleColor, routeSeparator, tail6} from '../lib/amlLogic.js';

const VISIBLE = 8;

// Маршрут просмотренных узлов внизу канваса.
export default function RouteBar({route, selected, edges, rolesById, onSelect, onBack, onClear}) {
  if (!route.length) return null;
  const hidden = Math.max(0, route.length - VISIBLE), visible = route.slice(hidden);
  const current = route.indexOf(selected);
  return <div className="routebar" aria-label="Маршрут просмотра">
    <span className="routebar__title">Маршрут <span className="routebar__count">{route.length}</span></span>
    {hidden > 0 && <span className="routebar__more">+{hidden}</span>}
    {visible.map((id, i) => {
      const index = hidden + i, sep = i > 0 ? routeSeparator(edges, visible[i - 1], id) : null;
      return <React.Fragment key={id}>
        {sep && <span className="routebar__sep" title={sep.title}>{sep.sign}</span>}
        <button className={`chip ${id === selected ? 'chip--current' : ''}`} onClick={() => onSelect(id)} title={id}>
          <span className="chip__num">{index + 1}</span><i className="dot dot--7" style={{background: roleColor(rolesById[id])}} />{tail6(id)}
        </button>
      </React.Fragment>;
    })}
    <span className="routebar__divider" />
    <button className="apx-btn apx-btn--ghost apx-btn--sm" onClick={onBack} disabled={current <= 0}>Назад</button>
    <button className="apx-btn apx-btn--ghost apx-btn--sm" onClick={onClear}>Очистить</button>
  </div>;
}
