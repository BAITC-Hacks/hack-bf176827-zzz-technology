import React from 'react';
import NodeTab from './NodeTab.jsx';
import Assistant from './Assistant.jsx';
import {fmtKzt, plural, roleColor, roleName} from '../lib/amlLogic.js';

const TABS = [['node', 'Узел'], ['seeds', 'Seed'], ['top', 'Топ-30'], ['clusters', 'Кластеры']];

export default function SidePanel({tab, onTab, card, route, llm, selected, seeds, top, clusters, robustness, ready, hidePeripheral, onSelect, onEgo, onCluster, onDisableLLM, onHighlight}) {
  return <aside className="panel">
    <nav className="tabs" aria-label="Панель анализа">
      {TABS.map(([key, label]) => <button key={key} className={`tab ${tab === key ? 'tab--active' : ''}`} aria-pressed={tab === key} onClick={() => onTab(key)}>
        {key === 'seeds' && <i className="diamond diamond--7" />}{label}{key === 'seeds' && seeds.length > 0 && <span className="tab__count"> {seeds.length}</span>}
      </button>)}
    </nav>

    {tab === 'node' && <NodeTab card={card} route={route} llm={llm} hidePeripheral={hidePeripheral} onSelect={onSelect} onEgo={onEgo} onDisableLLM={onDisableLLM} />}

    {tab === 'seeds' && <section className="panel__section">
      <h2 className="apx-h3">Исходные участники</h2>
      <p className="apx-hint">Seed из запроса, с них начинается обход. На схеме {seeds.length} из 81. Под каждым показано, куда деньги ушли дальше.</p>
      <div className="list">
        {seeds.map(s => <div key={s.gid} className={`seed ${selected === s.gid ? 'seed--selected' : ''}`}>
          <button className="seed__head" onClick={() => onSelect(s.gid)}>
            <i className="diamond diamond--12" />
            <span><span className="gid gid--link">{s.gid}</span><span className="apx-hint">Кластер {s.cluster}, отправил {fmtKzt(s.out_kzt)}</span></span>
          </button>
          {s.next?.length > 0 && <div className="seed__next"><span className="apx-hint">Дальше</span>
            {s.next.map(x => <button key={x.gid} className="seed__card" onClick={() => onSelect(x.gid)}>
              <span className="seed__arrow">→</span><span className="seed__who"><i className="dot dot--8" style={{background: roleColor(x.role)}} />{roleName(x.role)} <span className="gid">{x.gid}</span></span><span className="seed__sum">{fmtKzt(x.sum_kzt)}</span>
            </button>)}
          </div>}
        </div>)}
      </div>
    </section>}

    {tab === 'top' && <section className="panel__section">
      <h2 className="apx-h3">Кого смотреть первым</h2>
      <p className="apx-hint">30 узлов с наибольшим приоритетом. Seed в список не входят.</p>
      <div className="list">
        {top.map(t => <button key={t.gid} className={`toprow ${selected === t.gid ? 'toprow--selected' : ''}`} onClick={() => onSelect(t.gid)}>
          <span className="toprow__rank">{t.rank}</span>
          <span className="toprow__body">
            <span className="gid">{t.gid}</span>
            <span className="toprow__role"><i className="dot dot--8" style={{background: roleColor(t.role)}} />{roleName(t.role)} · {t.priority.toFixed(3)}</span>
            <span className="toprow__why">{t.why}</span>
          </span>
        </button>)}
      </div>
    </section>}

    {tab === 'clusters' && <>
      {robustness.length > 0 && <section className="panel__section">
        <h2 className="apx-h3">Устойчивость сети</h2>
        <p className="apx-hint">Что будет с оборотом, если изъять узлы из топа.</p>
        {robustness.map(r => <div key={r.removed} className="resil">
          <span className="ink-2">Топ-{r.removed}</span>
          <span className="bar bar--8"><span style={{width: Math.round(r.lost_turnover_share * 100) + '%'}} /></span>
          <span className="resil__drop">−{Math.round(r.lost_turnover_share * 100)}%</span>
          <span className="apx-hint resil__comp">Компонент {r.components_before} → {r.components_after}</span>
        </div>)}
      </section>}
      <section className="panel__section">
        <div className="title-row"><h2 className="apx-h3">Кластеры</h2><span className="count count--big">{clusters.length}</span></div>
        <p className="apx-hint">Клик по кластеру показывает его на всей сети.</p>
        <div className="list">
          {clusters.map(c => <button key={c.id} className="cluster" onClick={() => onCluster(c.id)}>
            <span className="cluster__head"><strong>Кластер {c.id}</strong><span className="apx-hint">{c.n_nodes} {plural(c.n_nodes, 'узел', 'узла', 'узлов')}</span></span>
            <span className="apx-hint">{fmtKzt(c.sum_kzt_internal)}, {c.n_seed} seed</span>
            <span className="cluster__hyp">{c.hypothesis}</span>
          </button>)}
        </div>
      </section>
    </>}

    {llm && ready && <Assistant selected={selected} route={route} onDisable={onDisableLLM} onHighlight={onHighlight} onSelect={onSelect} />}
  </aside>;
}
