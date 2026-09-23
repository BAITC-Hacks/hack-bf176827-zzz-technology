import React, {useState} from 'react';
import {money, roleName} from './graph.js';
import {request} from './api.js';

const metrics = {role_score: 'Уверенность', priority: 'Приоритет', cluster: 'Кластер', depth: 'Глубина', is_seed: 'Seed', in_deg: 'Входящих связей', out_deg: 'Исходящих связей', in_kzt: 'Получено', out_kzt: 'Отправлено', in_tx: 'Входящих переводов', out_tx: 'Исходящих переводов', pass_through: 'Доля транзита', pagerank: 'PageRank', betweenness: 'Посредничество', n_seed_upstream: 'Seed выше по цепочке', component_id: 'Компонента', truncated: 'Обрыв обхода', verified_sink: 'Подтверждённый сток'};
function display(key, value) {
  if (key.endsWith('kzt')) return money(value);
  if (typeof value === 'boolean') return value ? 'Да' : 'Нет';
  if (key === 'pass_through' && value === -1) return 'Нет входящих';
  if (typeof value === 'number' && !Number.isInteger(value)) return Number(value.toPrecision(4));
  return value;
}
export default function NodeCard({card, onSelect, llm, onDisableLLM}) {
  const [text, setText] = useState(''), [loading, setLoading] = useState(false), [copyState, setCopyState] = useState('Скопировать');
  if (!card) return <section id="card"><h2>Выберите узел</h2><p className="empty">Найдите gid или нажмите на узел графа. Стрелки показывают направление переводов, ромб обозначает seed.</p></section>;
  const n = card.node;
  async function reference() {
    setLoading(true); setText('Готовим справку…');
    try {
      const result = await request(`/v1/nodes/${encodeURIComponent(n.id)}/card`);
      setText(typeof result === 'string' ? result : result.text ?? result.card ?? result.answer ?? 'Пустая справка');
    } catch (error) {
      if ([404, 503].includes(error.status)) { onDisableLLM(); setText(''); } else setText(error.message);
    } finally { setLoading(false); }
  }
  async function copy() {
    const lines = [n.id, roleName(n.role), n.evidence, ...Object.entries(metrics).filter(([k]) => n[k] !== undefined).map(([k, label]) => `${label}: ${display(k, n[k])}`)];
    for (const [label, entries] of [['Входящие', card.incoming], ['Исходящие', card.outgoing]]) lines.push(label, ...entries.map(e => `${e.gid}: ${money(e.sum_kzt)}, ${e.n_tx} переводов`));
    try { await navigator.clipboard.writeText(lines.join('\n')); setCopyState('Скопировано'); } catch { setCopyState('Копирование недоступно'); }
  }
  return <section id="card">
    <div className="eyebrow">Карточка узла</div><h2>{roleName(n.role)}</h2><p className="gid">{n.id}</p>
    <p className="evidence">{n.evidence}</p>
    <div className="actions"><button onClick={() => onSelect(n.id, 2)}>Ego 2 шага</button><button onClick={copy}>{copyState}</button>{llm && <button onClick={reference} disabled={loading}>Справка</button>}</div>
    {text && <p className="prewrap" role="status">{text}</p>}
    <dl className="metrics">{Object.entries(metrics).filter(([key]) => n[key] !== undefined).map(([key, label]) => <div key={key}><dt>{label}</dt><dd>{display(key, n[key])}</dd></div>)}</dl>
    {[['Входящие', card.incoming], ['Исходящие', card.outgoing]].map(([label, entries]) => <div key={label}><h3>{label} <span className="count">{entries.length}</span></h3>{!entries.length && <p className="empty">Нет переводов</p>}{entries.map((entry, i) => <button className="neighbor" key={`${entry.gid}:${i}`} onClick={() => onSelect(entry.gid)}><span className="gid">{entry.gid}</span><small>{money(entry.sum_kzt)} · {entry.n_tx} переводов</small></button>)}</div>)}
  </section>;
}
