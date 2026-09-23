import React, {useState} from 'react';
import {request} from '../api.js';
import {LEVELS, candidateWhy, fmtKzt, metricRows, plural, roleColor, roleName, signalSummary} from '../lib/amlLogic.js';

const COLLAPSED_ROWS = 8;
const HELP_TITLES = ['Роль и почему', 'Потоки', 'Связи', 'На что обратить внимание'];

// Разбить текст справки на четыре блока по заголовкам (LLM и шаблон используют одни и те же заголовки).
function splitHelp(text) {
  const clean = text.replace(/\*\*/g, '');
  const parts = HELP_TITLES.map(title => ({title, at: clean.indexOf(title)})).filter(p => p.at >= 0).sort((a, b) => a.at - b.at);
  if (parts.length < 2) return [{title: '', text: clean.trim()}];
  return parts.map((p, i) => ({title: p.title, text: clean.slice(p.at + p.title.length, parts[i + 1]?.at ?? clean.length).replace(/^[\s:.\-–—]+/, '').trim()}));
}

export default function NodeTab({card, route, llm, hidePeripheral, onSelect, onEgo, onDisableLLM}) {
  const [help, setHelp] = useState({state: 'none', text: '', byLLM: false});
  const [allMetrics, setAllMetrics] = useState(false);
  const [copied, setCopied] = useState(false);
  if (!card) return <section className="panel__section"><h2 className="apx-h3">Выберите узел</h2><p className="apx-hint">Найдите gid или нажмите на узел графа. Стрелки показывают направление переводов, ромб обозначает seed.</p></section>;

  const n = card.node, rows = metricRows(card), shown = allMetrics ? rows : rows.slice(0, COLLAPSED_ROWS), summary = signalSummary(rows);
  const candidates = card.next_candidates || [], nearest = card.nearest_seed;
  const candsTitle = candidates[0]?.direction === 'up' ? 'Дальше переводов нет. Кто стоит выше по цепочке' : 'Кто вероятнее всего следующий ключевой узел';

  async function loadHelp() {
    setHelp({state: 'loading', text: '', byLLM: false});
    try {
      const result = await request(`/v1/nodes/${encodeURIComponent(n.id)}/card`);
      setHelp({state: 'ready', text: result.text || '', byLLM: !!result.by_llm});
    } catch (error) {
      if ([404, 503].includes(error.status)) { onDisableLLM(); setHelp({state: 'none', text: '', byLLM: false}); } else setHelp({state: 'error', text: error.message, byLLM: false});
    }
  }
  async function copy() {
    try { await navigator.clipboard.writeText(n.id); setCopied(true); setTimeout(() => setCopied(false), 1500); } catch { /* буфер недоступен */ }
  }
  const neighborRow = (e, i) => <button className="neighbor" key={`${e.gid}:${i}`} onClick={() => onSelect(e.gid)}>
    <span className="neighbor__who">{e.is_seed ? <i className="diamond diamond--7" /> : <i className="dot dot--8" style={{background: roleColor(e.role)}} />}<span className="gid gid--link">{e.gid}</span></span>
    <span className="neighbor__meta">{fmtKzt(e.sum_kzt)} · {e.n_tx} {plural(e.n_tx, 'перевод', 'перевода', 'переводов')}</span>
  </button>;
  // Основные переводы цепочки первыми; периферия отдельной группой, при исключении — только счётчик.
  const neighborList = (title, entries) => {
    const main = entries.filter(e => e.is_seed || e.role !== 'peripheral'), periphery = entries.filter(e => !e.is_seed && e.role === 'peripheral');
    const peripherySum = periphery.reduce((a, e) => a + e.sum_kzt, 0);
    return <div className="panel__section">
      <h3 className="section-title">{title} <span className="count">{entries.length}</span></h3>
      {!entries.length && <p className="apx-hint">Нет переводов</p>}
      {main.map(neighborRow)}
      {periphery.length > 0 && <details className="periphery">
        <summary className="periphery__summary">
          <span>{hidePeripheral ? 'Периферия исключена' : 'Периферия'} <span className="count">{periphery.length}</span></span>
          <span className="neighbor__meta">{fmtKzt(peripherySum)}</span>
        </summary>
        {periphery.map(neighborRow)}
      </details>}
    </div>;
  };

  return <div className="node-tab">
    <section className="panel__section">
      <p className="apx-eyebrow">Карточка узла</p>
      <h2 className="apx-h3 role-title"><i className="dot dot--10" style={{background: roleColor(n.role)}} />{roleName(n.role)}</h2>
      <p className="gid gid--big">{n.id}</p>
      <div className="badges">
        <span className="apx-badge">Приоритет {(n.priority ?? 0).toFixed(3)}</span>
        {n.is_seed && <span className="apx-badge apx-badge--outline">seed</span>}
        {n.truncated && <span className="apx-badge apx-badge--outline">обрезан обходом</span>}
        {n.features?.verified_sink && <span className="apx-badge apx-badge--outline">подтверждённый сток</span>}
        {n.features?.structuring && <span className="apx-badge apx-badge--warn">признаки дробления</span>}
      </div>
      <p className="evidence apx-sm">{n.evidence}</p>
      <p className="seed-line apx-sm"><i className="diamond diamond--10" />
        {n.is_seed ? 'Это seed: исходный участник из запроса. Смотрите, куда его деньги ушли дальше.'
          : nearest ? <>Ближайший seed выше по цепочке: {nearest.steps} {plural(nearest.steps, 'шаг', 'шага', 'шагов')} · <button className="link" onClick={() => onSelect(nearest.gid)}>{nearest.gid}</button></>
          : 'Выше по цепочке seed нет: деньги не приходят от исходных участников.'}
      </p>
      <div className="actions">
        <button className="apx-btn apx-btn--secondary apx-btn--sm" onClick={() => onEgo(n.id)}>Ego 2 шага</button>
        <button className="apx-btn apx-btn--ghost apx-btn--sm" onClick={copy}>{copied ? 'Скопировано' : 'Скопировать'}</button>
        {llm && <button className="apx-btn apx-btn--ghost apx-btn--sm" onClick={loadHelp} disabled={help.state === 'loading'}>Справка</button>}
      </div>
    </section>

    {candidates.length > 0 && <section className="panel__section">
      <p className="section-title">Куда смотреть дальше</p>
      <p className="apx-hint">{candsTitle}. Оценка по доле потока и приоритету, это гипотеза.</p>
      {candidates.map((c, i) => <button key={c.gid} className={`candidate ${i === 0 ? 'candidate--first' : ''}`} onClick={() => onSelect(c.gid)}>
        <span className="candidate__rank">{i + 1}</span>
        <span className="candidate__body">
          <span className="candidate__role"><i className="dot dot--8" style={{background: roleColor(c.role)}} />{roleName(c.role)}</span>
          <span className="gid">{c.gid}</span>
          <span className="candidate__why">{candidateWhy(c, route)}</span>
        </span>
        <span className="candidate__pct">{c.score_pct}%</span>
      </button>)}
    </section>}

    {help.state === 'loading' && <section className="panel__section help"><p className="apx-hint">Готовим справку…</p><span className="apx-skeleton" style={{width: '70%', height: 12}} /><span className="apx-skeleton" style={{height: 12}} /><span className="apx-skeleton" style={{width: '85%', height: 12}} /></section>}
    {help.state === 'error' && <p className="apx-hint danger" role="alert">{help.text}</p>}
    {help.state === 'ready' && <section className="panel__section help" role="status">
      {splitHelp(help.text).map((s, i) => <div key={i}>{s.title && <p className="section-title">{s.title}</p>}<p className="apx-sm ink-2 prewrap">{s.text}</p></div>)}
      {!help.byLLM && <p className="apx-hint">Шаблонная справка без LLM.</p>}
    </section>}

    <section className="panel__section signals">
      <div className="signals__head"><p className="section-title">Сигналы по метрикам</p><span className="apx-hint">относительно всей сети</span></div>
      <div className="signals__bar">{summary.map(s => <span key={s.level} style={{flex: s.n, background: `var(${LEVELS[s.level].token})`}} />)}</div>
      <div className="signals__legend">{summary.map(s => <span key={s.level}><i style={{background: `var(${LEVELS[s.level].token})`}} />{LEVELS[s.level].legend}: {s.n}</span>)}</div>
      <div className="metrics">
        {shown.map(m => <div key={m.key} className="metric">
          <span className="metric__label">{m.label}</span>
          <span className="metric__value" style={{fontWeight: m.level === 'high' ? 600 : m.level === 'mid' ? 500 : 400}}>{m.value}</span>
          <span className="metric__meter" title={LEVELS[m.level].name}>{[0, 1, 2, 3, 4].map(i => <i key={i} style={{background: i < LEVELS[m.level].fill ? `var(${LEVELS[m.level].token})` : 'var(--apx-surface-3)', visibility: m.level === 'info' ? 'hidden' : 'visible'}} />)}</span>
          {m.note && <span className="metric__note">{m.note}</span>}
        </div>)}
      </div>
      {rows.length > COLLAPSED_ROWS && <button className="apx-btn apx-btn--ghost apx-btn--sm" onClick={() => setAllMetrics(v => !v)}>{allMetrics ? 'Свернуть метрики' : `Все ${rows.length} метрик`}</button>}
    </section>

    {neighborList('Входящие', card.incoming)}
    {neighborList('Исходящие', card.outgoing)}
  </div>;
}
