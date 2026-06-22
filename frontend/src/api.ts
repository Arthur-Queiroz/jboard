export interface Reminder {
  id: number
  card_id: number
  reminder_at: string
  recipient: string
  message: string
  sent_at: string | null
  created_at: string
  updated_at: string
}

export interface Card {
  id: number
  column_id: number
  title: string
  description: string
  position: number
  reminders: Reminder[]
  created_at: string
  updated_at: string
}

export interface Column {
  id: number
  board_id: number
  title: string
  position: number
  cards: Card[]
  created_at: string
  updated_at: string
}

export interface Board {
  id: number
  title: string
  columns: Column[]
  created_at: string
  updated_at: string
}

// Board sem columns — é o que GET /api/boards retorna (lista leve).
export interface BoardSummary {
  id: number
  title: string
  created_at: string
  updated_at: string
}

// Base da API. Web e desktop-em-dev usam '/api' (proxy do Vite / mesma origem em
// produção). O build do desktop empacotado define VITE_JBOARD_API_BASE com a URL
// absoluta do backend (ex.: https://jboard.devarthur.com.br/api), já que o webview
// não tem proxy nem backend na própria origem. Sem barra final.
const BASE = import.meta.env.VITE_JBOARD_API_BASE ?? '/api'

// Token de auth injetado em produção via Vite env (VITE_JBOARD_API_TOKEN).
// Em dev local (token vazio no backend), fica undefined e o header não é enviado.
const apiToken = import.meta.env.VITE_JBOARD_API_TOKEN as string | undefined

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (apiToken) {
    headers['Authorization'] = `Bearer ${apiToken}`
  }
  const res = await fetch(`${BASE}${path}`, { headers, ...init })
  if (!res.ok) {
    throw new Error(`${res.status} ${await res.text()}`)
  }
  if (res.status === 204) {
    return undefined as T
  }
  return res.json() as Promise<T>
}

export const api = {
  listBoards: () => request<BoardSummary[]>('/boards'),
  getBoard: (id: number) => request<Board>(`/boards/${id}`),
  createBoard: (title: string) =>
    request<BoardSummary>('/boards', { method: 'POST', body: JSON.stringify({ title }) }),
  updateBoard: (id: number, title: string) =>
    request<Board>(`/boards/${id}`, { method: 'PUT', body: JSON.stringify({ title }) }),
  deleteBoard: (id: number) => request<void>(`/boards/${id}`, { method: 'DELETE' }),

  createColumn: (boardID: number, title: string, position: number) =>
    request<Column>(`/boards/${boardID}/columns`, {
      method: 'POST',
      body: JSON.stringify({ title, position }),
    }),
  updateColumn: (id: number, title: string, position: number) =>
    request<Column>(`/columns/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ title, position }),
    }),
  deleteColumn: (id: number) => request<void>(`/columns/${id}`, { method: 'DELETE' }),

  createCard: (columnID: number, title: string, position: number) =>
    request<Card>(`/columns/${columnID}/cards`, {
      method: 'POST',
      body: JSON.stringify({ title, position }),
    }),
  updateCard: (card: Partial<Card> & { id: number }) =>
    request<Card>(`/cards/${card.id}`, { method: 'PUT', body: JSON.stringify(card) }),
  // Reorder fixa a ordem dos cards de uma coluna após um drag-and-drop.
  // cardIDs na ordem visual desejada (position = índice no array).
  reorderCards: (columnID: number, cardIDs: number[]) =>
    request<void>(`/columns/${columnID}/cards/reorder`, {
      method: 'POST',
      body: JSON.stringify({ card_ids: cardIDs }),
    }),
  deleteCard: (id: number) => request<void>(`/cards/${id}`, { method: 'DELETE' }),

  createReminder: (cardID: number, reminderAt: string, message: string, recipient = '') =>
    request<Reminder>(`/cards/${cardID}/reminders`, {
      method: 'POST',
      body: JSON.stringify({ reminder_at: reminderAt, message, recipient }),
    }),
}
