import React from 'react';
import {ROLES, roleColor, roleName} from '../lib/amlLogic.js';

const sourceTone = {api: '', offline: 'apx-badge--warn', demo: 'apx-badge--info'};

const MODE_HINT = {
  top: 'Топ-30: 30 узлов с наибольшим приоритетом проверки и их прямые контрагенты. Seed показаны всегда.',
  all: 'Вся сеть: все 2 248 узлов, в фокусе крупнейшая компонента. Seed показаны всегда.',
};

export default function Toolbar({source, sourceKind, query, onQuery, onSearch, suggestions, searchError, filters, onFilters, clusters, ready, hidePeripheral, onHidePeripheral}) {
  return <header className="topbar">
    <div className="topbar__head">
      <div><p className="apx-eyebrow">Исследование транзакционной сети</p><h1 className="apx-h2">Граф денег</h1></div>
      <span className={`apx-badge ${sourceTone[sourceKind] || ''}`}>{source}</span>
    </div>
    <form className="toolbar" onSubmit={onSearch}>
      <input className="apx-input toolbar__search" id="search" aria-label="Поиск по gid" placeholder="Полный gid или начало номера" value={query}
        onChange={e => onQuery(e.target.value)} list="gid-hints" autoComplete="off" disabled={!ready} />
      <datalist id="gid-hints">{suggestions.map(n => <option key={n.id} value={n.id}>{roleName(n.role)}</option>)}</datalist>
      <button className="apx-btn apx-btn--primary" disabled={!ready}>Найти</button>
      <span className="toolbar__divider" />
      <div className="apx-select-wrap toolbar__role"><select className="apx-select" aria-label="Роль" value={filters.role} onChange={e => onFilters({...filters, role: e.target.value})} disabled={!ready}>
        <option value="">Все роли</option>{Object.entries(ROLES).map(([key, r]) => <option key={key} value={key}>{r.name}</option>)}
      </select></div>
      <div className="apx-select-wrap toolbar__cluster"><select className="apx-select" aria-label="Кластер" value={filters.cluster} onChange={e => onFilters({...filters, cluster: e.target.value})} disabled={!ready}>
        <option value="">Все кластеры</option>{clusters.map(c => <option key={c.id} value={String(c.id)}>Кластер {c.id}</option>)}
      </select></div>
      <div className="segment" role="group" aria-label="Объём сети" title={MODE_HINT[filters.topOnly ? 'top' : 'all']}>
        <button type="button" className={`apx-btn apx-btn--sm ${filters.topOnly ? 'apx-btn--secondary' : 'apx-btn--ghost'}`} aria-pressed={filters.topOnly} onClick={() => onFilters({...filters, topOnly: true})} disabled={!ready}>Топ-30</button>
        <button type="button" className={`apx-btn apx-btn--sm ${!filters.topOnly ? 'apx-btn--secondary' : 'apx-btn--ghost'}`} aria-pressed={!filters.topOnly} onClick={() => onFilters({...filters, topOnly: false})} disabled={!ready}>Вся сеть</button>
      </div>
      <button type="button" className={`apx-btn apx-btn--sm ${hidePeripheral ? 'apx-btn--secondary' : 'apx-btn--ghost'}`} aria-pressed={hidePeripheral} onClick={() => onHidePeripheral(!hidePeripheral)} disabled={!ready}
        title="Периферия: узлы без признаков роли, 84% сети. Seed, выбранный узел и его контрагенты остаются видны.">Исключить периферию</button>
    </form>
    <p className="apx-hint mode-hint">{MODE_HINT[filters.topOnly ? 'top' : 'all']}{hidePeripheral ? ' Периферия исключена, кроме связей выбранного узла.' : ''}</p>
    {searchError && <p className="apx-hint search-error" role="alert">{searchError}</p>}
    <div className="legend">
      {Object.entries(ROLES).map(([key, r]) => <span key={key}><i className="dot" style={{background: roleColor(key)}} />{r.name}</span>)}
      <span><i className="diamond" />Seed, исходный участник</span>
      <span><i className="dot dashed" />Обрезан обходом</span>
    </div>
  </header>;
}
