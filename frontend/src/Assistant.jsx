import React, {useState} from 'react';
import {request} from './api.js';

export default function Assistant({onDisable, onHighlight, onSelect}) {
  const [question, setQuestion] = useState(''), [answer, setAnswer] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState('');
  async function ask(event) {
    event.preventDefault(); if (!question.trim()) return;
    setBusy(true); setError('');
    try {
      const result = await request('/v1/assistant', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({question: question.trim()})});
      const gids = [...new Set((result.gids || []).filter(id => typeof id === 'string' && /^\d{1,19}$/.test(id)))];
      setAnswer({...result, gids}); await onHighlight(gids);
    } catch (e) { if ([404, 503].includes(e.status)) onDisable(); else setError(e.message); } finally { setBusy(false); }
  }
  return <section id="assistant"><h2>Вопрос ассистенту</h2><form onSubmit={ask}><textarea aria-label="Вопрос ассистенту" value={question} onChange={e => setQuestion(e.target.value)} rows={3} maxLength={2000} placeholder="Кто собирает деньги в этой группе?" required /><button className="primary" disabled={busy}>{busy ? 'Готовим ответ…' : 'Спросить'}</button></form>{error && <p role="alert">{error}</p>}{answer && <div id="answer"><p className="prewrap">{answer.answer}</p>{answer.gids.map(id => <button className="neighbor gid" key={id} onClick={() => onSelect(id)}>{id}</button>)}</div>}</section>;
}
