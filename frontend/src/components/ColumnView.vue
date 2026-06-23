<script setup lang="ts">
import { ref } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import { api, type Card, type Column } from '../api'

const props = defineProps<{ column: Column }>()
const emit = defineEmits<{ (e: 'changed'): void }>()

const newCardTitle = ref('')
const openReminderFor = ref<number | null>(null)
const reminderAt = ref('')
const reminderMessage = ref('')

// Edição inline de coluna e card.
const editingColumn = ref(false)
const columnTitleDraft = ref(props.column.title)
const editingCardId = ref<number | null>(null)
const cardDraftTitle = ref('')
const cardDraftDescription = ref('')

async function addCard() {
  const title = newCardTitle.value.trim()
  if (!title) return
  const card = await api.createCard(props.column.id, title, props.column.cards.length)
  props.column.cards.push(card)
  newCardTitle.value = ''
}

async function removeCard(cardId: number) {
  await api.deleteCard(cardId)
  props.column.cards = props.column.cards.filter((card) => card.id !== cardId)
}

async function removeColumn() {
  await api.deleteColumn(props.column.id)
  emit('changed')
}

// --- DnD ---
// Cada coluna é um draggable do mesmo grupo ("cards"), permitindo mover cards
// entre colunas e reordenar dentro da coluna. Após qualquer mudança, persiste a
// ordem desta coluna; o reorder da coluna de origem (no remove) e da de destino
// (no add) cobrem o move cross-column. Em erro, recarrega pra reverter.
async function persistOrder() {
  try {
    await api.reorderCards(props.column.id, props.column.cards.map((card) => card.id))
  } catch (e) {
    emit('changed')
  }
}

// --- edição de coluna ---
function startEditColumn() {
  columnTitleDraft.value = props.column.title
  editingColumn.value = true
}

async function saveColumn() {
  const title = columnTitleDraft.value.trim()
  editingColumn.value = false
  if (!title || title === props.column.title) return
  try {
    await api.updateColumn(props.column.id, title, props.column.position)
    props.column.title = title
  } catch (e) {
    emit('changed')
  }
}

// --- edição de card ---
function startEditCard(card: Card) {
  editingCardId.value = card.id
  cardDraftTitle.value = card.title
  cardDraftDescription.value = card.description
}

async function saveCard(card: Card) {
  const title = cardDraftTitle.value.trim()
  editingCardId.value = null
  if (!title || (title === card.title && cardDraftDescription.value === card.description)) return
  try {
    await api.updateCard({
      id: card.id,
      column_id: card.column_id,
      title,
      description: cardDraftDescription.value,
      position: card.position,
    })
    card.title = title
    card.description = cardDraftDescription.value
  } catch (e) {
    emit('changed')
  }
}

// --- lembretes ---
function toggleReminder(cardId: number) {
  openReminderFor.value = openReminderFor.value === cardId ? null : cardId
  reminderAt.value = ''
  reminderMessage.value = ''
}

async function addReminder(cardId: number) {
  if (!reminderAt.value) return
  await api.createReminder(
    cardId,
    new Date(reminderAt.value).toISOString(),
    reminderMessage.value.trim(),
  )
  reminderAt.value = ''
  reminderMessage.value = ''
  openReminderFor.value = null
  emit('changed')
}

function formatReminder(iso: string): string {
  return new Date(iso).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
}

// Path do ícone do WhatsApp (reutilizado no chip e no botão).
const WA_PATH =
  'M12 2a10 10 0 0 0-8.6 15.1L2 22l5-1.3A10 10 0 1 0 12 2zm0 18a8 8 0 0 1-4.1-1.1l-.3-.2-3 .8.8-2.9-.2-.3A8 8 0 1 1 12 20zm4.4-5.9c-.2-.1-1.4-.7-1.6-.8s-.4-.1-.5.1-.6.8-.8 1-.3.1-.5 0a6.5 6.5 0 0 1-3.2-2.8c-.2-.4.2-.4.6-1.2a.4.4 0 0 0 0-.4l-.7-1.7c-.2-.5-.4-.4-.5-.4h-.5a.9.9 0 0 0-.7.3 2.8 2.8 0 0 0-.9 2.1 5 5 0 0 0 1 2.6 11 11 0 0 0 4.3 3.8c1.6.6 1.9.5 2.3.5a2.4 2.4 0 0 0 1.6-1.1 2 2 0 0 0 .1-1.1c0-.1-.2-.2-.4-.3z'
</script>

<template>
  <section class="column">
    <div class="column-header">
      <h3 v-if="!editingColumn" title="clique duplo para renomear" @dblclick="startEditColumn">
        {{ column.title }}
      </h3>
      <input v-else v-model="columnTitleDraft" @keyup.enter="saveColumn" @blur="saveColumn" />
      <button class="col-delete" title="excluir coluna" @click="removeColumn">×</button>
    </div>

    <VueDraggable
      v-model="column.cards"
      :group="{ name: 'cards', pull: true, put: true }"
      item-key="id"
      class="cards"
      :animation="150"
      chosen-class="card-dragging"
      @update="persistOrder"
      @add="persistOrder"
      @remove="persistOrder"
    >
      <div v-for="card in column.cards" :key="card.id" class="card">
        <div v-if="editingCardId === card.id" class="card-edit">
          <input v-model="cardDraftTitle" placeholder="título" @keyup.enter="saveCard(card)" />
          <textarea v-model="cardDraftDescription" rows="2" placeholder="descrição" />
          <div class="form-actions">
            <button class="btn-accent" @click="saveCard(card)">salvar</button>
            <button class="btn-ghost" @click="editingCardId = null">cancelar</button>
          </div>
        </div>

        <template v-else>
          <div class="card-head">
            <span class="card-title" @click="startEditCard(card)">{{ card.title }}</span>
            <button class="card-delete" @click="removeCard(card.id)">×</button>
          </div>
          <p v-if="card.description" class="card-desc" @click="startEditCard(card)">
            {{ card.description }}
          </p>

          <div v-if="card.reminders?.length" class="reminders">
            <span
              v-for="reminder in card.reminders"
              :key="reminder.id"
              class="reminder-chip"
              :class="{ sent: reminder.sent_at }"
              :title="reminder.sent_at ? 'WhatsApp enviado' : 'WhatsApp agendado'"
            >
              <svg width="11" height="11" viewBox="0 0 24 24" fill="currentColor">
                <path :d="WA_PATH" />
              </svg>
              {{ formatReminder(reminder.reminder_at) }}
            </span>
          </div>

          <div class="card-actions">
            <button class="edit-btn" @click="startEditCard(card)">editar</button>
            <button class="wa-btn" @click="toggleReminder(card.id)">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor">
                <path :d="WA_PATH" />
              </svg>
              WhatsApp
            </button>
          </div>

          <div v-if="openReminderFor === card.id" class="reminder-form">
            <span class="reminder-label">Enviar pelo WhatsApp em:</span>
            <input type="datetime-local" v-model="reminderAt" />
            <textarea v-model="reminderMessage" placeholder="Mensagem (opcional)" rows="2" />
            <div class="form-actions">
              <button class="btn-wa" @click="addReminder(card.id)">agendar envio</button>
              <button class="btn-ghost" @click="toggleReminder(card.id)">cancelar</button>
            </div>
          </div>
        </template>
      </div>
    </VueDraggable>

    <div class="add-form">
      <input v-model="newCardTitle" placeholder="novo card" @keyup.enter="addCard" />
      <button title="adicionar card" @click="addCard">+</button>
    </div>
  </section>
</template>
