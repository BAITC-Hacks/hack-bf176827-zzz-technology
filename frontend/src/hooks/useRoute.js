import {useCallback, useEffect, useState} from 'react';
import {ROUTE_MAX, ROUTE_STORAGE_KEY} from '../lib/amlLogic.js';

// Маршрут просмотренных узлов с сохранением в localStorage.
export function useRoute() {
  const [route, setRoute] = useState(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(ROUTE_STORAGE_KEY) || '[]');
      return Array.isArray(saved) ? saved.filter(id => /^\d{1,19}$/.test(id)).slice(-ROUTE_MAX) : [];
    } catch { return []; }
  });
  useEffect(() => { try { localStorage.setItem(ROUTE_STORAGE_KEY, JSON.stringify(route)); } catch { /* приватный режим */ } }, [route]);
  const push = useCallback(id => setRoute(r => r.includes(id) ? r : [...r, id].slice(-ROUTE_MAX)), []);
  const clear = useCallback(() => setRoute([]), []);
  return {route, push, clear};
}
