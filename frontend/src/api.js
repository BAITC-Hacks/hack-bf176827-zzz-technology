export class APIError extends Error {
  constructor(message, status) { super(message); this.status = status; }
}
export async function request(path, options = {}) {
  const response = await fetch(path, options);
  if (!response.ok) {
    let message = `Ошибка ${response.status}`;
    try { const body = await response.json(); message = body.errors?.[0]?.msg || message; } catch { /* Ответ прокси может не быть JSON. */ }
    throw new APIError(message, response.status);
  }
  return response.json();
}
