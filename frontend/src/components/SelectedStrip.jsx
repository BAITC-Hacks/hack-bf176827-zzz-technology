import React from 'react';
import {cardStats, fmtKzt, plural, roleColor, roleName} from '../lib/amlLogic.js';

// Полоса над канвасом: ключевые цифры выбранного узла.
export default function SelectedStrip({card, status}) {
  if (!card) return <div className="strip"><span className="apx-hint strip__empty">Выберите узел на схеме или найдите его по gid.</span><span className="apx-hint strip__status">{status}</span></div>;
  const n = card.node, st = cardStats(card);
  const inSub = card.incoming.length ? `от ${card.incoming.length} ${plural(card.incoming.length, 'плательщика', 'плательщиков', 'плательщиков')}${st.seedIn ? `, seed: ${st.seedIn}` : ''}` : 'входящих нет';
  const outSub = card.outgoing.length ? `${card.outgoing.length} ${plural(card.outgoing.length, 'получателю', 'получателям', 'получателям')}` : 'дальше не отправляет';
  return <div className="strip">
    <div className="strip__block strip__block--first">
      <span className="strip__role"><i className="dot dot--10" style={{background: roleColor(n.role)}} />{roleName(n.role)}</span>
      <span className="strip__gid">{n.id}</span>
    </div>
    <div className="strip__block"><span className="strip__label"><i className="swatch swatch--brand" />Получено</span><span className="strip__value">{fmtKzt(st.inK)}</span><span className="strip__sub">{inSub}</span></div>
    <div className="strip__block"><span className="strip__label"><i className="swatch swatch--ink" />Отправлено</span><span className="strip__value">{fmtKzt(st.outK)}</span><span className="strip__sub">{outSub}</span></div>
    <div className="strip__block strip__block--retain"><span className="strip__label">Удерживает</span><span className="strip__value">{card.incoming.length ? st.retain + '%' : 'н/д'}</span><span className="bar"><span style={{width: st.retain + '%'}} /></span></div>
    {st.biggest && <div className="strip__block"><span className="strip__label">Крупнейший перевод</span><span className="strip__value">{fmtKzt(st.biggest.e.sum_kzt)}</span><span className="strip__sub strip__sub--gid">{st.biggest.dir} {st.biggest.other}</span></div>}
    <span className="apx-hint strip__status">{status}</span>
  </div>;
}
