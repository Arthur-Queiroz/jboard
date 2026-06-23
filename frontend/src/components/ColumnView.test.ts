import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ColumnView from './ColumnView.vue'
import type { Column } from '../api'
import { api } from '../api'

// O client é mockado: os testes verificam render/interação, não a rede.
vi.mock('../api', () => ({
  api: {
    createCard: vi.fn(),
    deleteCard: vi.fn(),
    deleteColumn: vi.fn(),
    updateColumn: vi.fn(),
    updateCard: vi.fn(),
    reorderCards: vi.fn(),
    createReminder: vi.fn(),
  },
}))

// VueDraggable é stubado por um <div> que renderiza o slot — assim os cards
// aparecem no DOM sem depender da lib de drag.
const DraggableStub = { template: '<div><slot /></div>' }

function makeColumn(): Column {
  return {
    id: 1,
    board_id: 1,
    title: 'A fazer',
    position: 0,
    created_at: '',
    updated_at: '',
    cards: [
      {
        id: 10,
        column_id: 1,
        title: 'Estudar Go',
        description: 'caps de concorrência',
        position: 0,
        created_at: '',
        updated_at: '',
        reminders: [
          { id: 100, card_id: 10, reminder_at: '2030-01-01T10:00:00Z', recipient: '', message: 'x', sent_at: null, created_at: '', updated_at: '' },
        ],
      },
    ],
  }
}

function mountColumn(column = makeColumn()) {
  return mount(ColumnView, {
    props: { column },
    global: { stubs: { VueDraggable: DraggableStub } },
  })
}

describe('ColumnView', () => {
  beforeEach(() => vi.clearAllMocks())

  it('renderiza título da coluna, card e chip de lembrete (data formatada)', () => {
    const wrapper = mountColumn()
    expect(wrapper.find('h3').text()).toBe('A fazer')
    expect(wrapper.find('.card-title').text()).toBe('Estudar Go')
    const chip = wrapper.find('.reminder-chip')
    expect(chip.exists()).toBe(true)
    expect(chip.text()).toMatch(/\d/) // mostra a data formatada (ícone WA + data)
    // pendente (sent_at null) → sem classe .sent
    expect(chip.classes()).not.toContain('sent')
  })

  it('chip de lembrete enviado ganha a classe .sent', () => {
    const col = makeColumn()
    col.cards[0].reminders[0].sent_at = '2030-01-01T10:00:00Z'
    const wrapper = mountColumn(col)
    expect(wrapper.find('.reminder-chip').classes()).toContain('sent')
  })

  it('botão WhatsApp abre o formulário de lembrete', async () => {
    const wrapper = mountColumn()
    expect(wrapper.find('.reminder-form').exists()).toBe(false)
    const waButton = wrapper.findAll('button').find((b) => b.text() === 'WhatsApp')!
    await waButton.trigger('click')
    expect(wrapper.find('.reminder-form').exists()).toBe(true)
  })

  it('adicionar card chama api.createCard e injeta o retorno na coluna', async () => {
    const novo = { id: 11, column_id: 1, title: 'Novo', description: '', position: 1, reminders: [], created_at: '', updated_at: '' }
    vi.mocked(api.createCard).mockResolvedValue(novo)

    const wrapper = mountColumn()
    await wrapper.find('.add-form input').setValue('Novo')
    await wrapper.find('.add-form button').trigger('click')
    await Promise.resolve() // deixa o await do handler resolver

    expect(api.createCard).toHaveBeenCalledWith(1, 'Novo', 1)
    expect(wrapper.props('column').cards.map((c) => c.title)).toContain('Novo')
  })
})
