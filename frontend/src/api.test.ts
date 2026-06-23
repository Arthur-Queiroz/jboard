import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, UnauthorizedError } from './api'

// Mocka o fetch global pra inspecionar URL/método/headers/corpo sem rede.
function mockFetch(response: Partial<Response> & { json?: () => Promise<unknown> }) {
  const fn = vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ({}),
    text: async () => '',
    ...response,
  })
  vi.stubGlobal('fetch', fn)
  return fn
}

afterEach(() => vi.unstubAllGlobals())

describe('api client', () => {
  it('GET /boards monta a URL relativa e devolve o JSON', async () => {
    const fetchMock = mockFetch({ json: async () => [{ id: 1, title: 'A' }] })
    const boards = await api.listBoards()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/boards')
    expect(init.headers['Content-Type']).toBe('application/json')
    // Sem VITE_JBOARD_API_TOKEN no ambiente de teste → sem Authorization.
    expect(init.headers.Authorization).toBeUndefined()
    expect(boards).toEqual([{ id: 1, title: 'A' }])
  })

  it('createBoard faz POST com o título no corpo', async () => {
    const fetchMock = mockFetch({ status: 201, json: async () => ({ id: 9, title: 'Nova' }) })
    await api.createBoard('Nova')

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/boards')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ title: 'Nova' })
  })

  it('createReminder envia reminder_at, message e recipient', async () => {
    const fetchMock = mockFetch({ status: 201, json: async () => ({ id: 1 }) })
    await api.createReminder(5, '2030-01-01T10:00:00.000Z', 'oi', '5511999999999')

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/cards/5/reminders')
    expect(JSON.parse(init.body)).toEqual({
      reminder_at: '2030-01-01T10:00:00.000Z',
      message: 'oi',
      recipient: '5511999999999',
    })
  })

  it('204 No Content devolve undefined (sem tentar parsear JSON)', async () => {
    mockFetch({ status: 204 })
    await expect(api.deleteBoard(1)).resolves.toBeUndefined()
  })

  it('resposta não-ok vira erro com status e corpo', async () => {
    mockFetch({ ok: false, status: 500, text: async () => 'boom' })
    await expect(api.listBoards()).rejects.toThrow('500 boom')
  })

  it('401 vira UnauthorizedError (pra disparar a tela de login)', async () => {
    mockFetch({ ok: false, status: 401, text: async () => '' })
    await expect(api.listBoards()).rejects.toBeInstanceOf(UnauthorizedError)
  })

  it('login faz POST /login com a senha', async () => {
    const fetchMock = mockFetch({ status: 200, json: async () => ({ status: 'ok' }) })
    await api.login('minhasenha')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/login')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ password: 'minhasenha' })
  })
})
