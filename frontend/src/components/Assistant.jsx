import React, {useState} from 'react';
import {request} from '../api.js';
import {roleColor} from '../lib/amlLogic.js';

export default function Assistant({selected, route, onDisable, onHighlight, onSelect}) {
  const [question, setQuestion] = useState(''), [answer, setAnswer] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState('');
  const [withRoute, setWithRoute] = useState(false);
  async function ask(event) {
    event.preventDefault(); if (!question.trim()) return;
    setBusy(true); setError('');
    try {
      const result = await request('/v1/assistant', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({question: question.trim(), gid: selected || undefined, route: withRoute && route.length ? route : undefined})});
      const gids = [...new Set((result.gids || []).filter(id => typeof id === 'string' && /^\d{1,19}$/.test(id)))];
      setAnswer({...result, gids}); await onHighlight(gids);
    } catch (e) { if ([404, 503].includes(e.status)) onDisable(); else setError(e.message); } finally { setBusy(false); }
  }
  const paragraphs = (answer?.answer || '').replace(/\*\*/g, '').split(/\n+/).filter(Boolean);
  return <section className="panel__section assistant">
    <h2 className="apx-h3">Вопрос ассистенту</h2>
    <form onSubmit={ask} className="apx-field">
      <label className="apx-label" htmlFor="ask">Ваш вопрос</label>
      <textarea id="ask" className="apx-textarea" value={question} onChange={e => setQuestion(e.target.value)} rows={3} maxLength={2000} placeholder="Кто собирает деньги в этой группе?" required />
      <span className="apx-hint">{selected ? <>Контекст: выбранный узел <span className="gid">{selected}</span>. </> : 'Выберите узел, чтобы спрашивать «про этот узел». '}Ответ займёт 8–12 секунд.</span>
      <label className="check" title="Ассистент получит список узлов, которые вы открывали, как подсказку о вашем интересе. Это не цепочка переводов и не источник истины.">
        <input type="checkbox" checked={withRoute} onChange={e => setWithRoute(e.target.checked)} disabled={!route.length} />
        <span>Выбранный путь{route.length ? ` (${route.length})` : ''}</span>
      </label>
      <div><button className={`apx-btn apx-btn--secondary ${busy ? 'apx-btn--loading' : ''}`} disabled={busy}>{busy ? 'Готовим ответ…' : 'Спросить'}</button></div>
    </form>
    {error && <p className="apx-hint danger" role="alert">{error}</p>}
    {answer && <div className="answer">
      {paragraphs.map((p, i) => <p key={i} className="apx-sm ink-2">{p}</p>)}
      {answer.gids.length > 0 && <>
        <p className="apx-hint">Упомянутые узлы подсвечены на схеме</p>
        <div className="pills">{answer.gids.map(id => <button className="chip chip--pill" key={id} onClick={() => onSelect(id)}><i className="dot dot--7" style={{background: roleColor(answer.roles?.[id])}} />{id}</button>)}</div>
      </>}
    </div>}
  </section>;
}
