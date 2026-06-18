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
  if (!reminderAt.value || !reminderMessage.value.trim()) return
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
</script>

<template>
  <section class="column">
    <header>
      <h3 v-if="!editingColumn" @dblclick="startEditColumn">{{ column.title }}</h3>
      <input
        v-else
        v-model="columnTitleDraft"
        @keyup.enter="saveColumn"
        @blur="saveColumn"
      />
      <button class="danger" @click="removeColumn">×</button>
    </header>

    <VueDraggable
      v-model="column.cards"
      :group="{ name: 'cards', pull: true, put: true }"
      item-key="id"
      class="cards"
      :animation="150"
      @update="persistOrder"
      @add="persistOrder"
      @remove="persistOrder"
    >
      <div v-for="card in column.cards" :key="card.id" class="card">
        <template v-if="editingCardId === card.id">
          <input v-model="cardDraftTitle" @keyup.enter="saveCard(card)" />
          <textarea v-model="cardDraftDescription" rows="2" placeholder="descrição"></textarea>
          <div class="card-actions">
            <button @click="saveCard(card)">salvar</button>
            <button @click="editingCardId = null">cancelar</button>
          </div>
        </template>
        <template v-else>
          <div class="title">
            <span class="card-title" @click="startEditCard(card)">{{ card.title }}</span>
            <button class="danger" @click="removeCard(card.id)">×</button>
          </div>
          <p v-if="card.description" class="desc" @click="startEditCard(card)">{{ card.description }}</p>
          <div v-if="card.reminders.length" class="reminders">
            <span
              v-for="reminder in card.reminders"
              :key="reminder.id"
              class="reminder-chip"
              :class="{ sent: reminder.sent_at }"
            >
              {{ formatReminder(reminder.reminder_at) }}
            </span>
          </div>
          <div class="card-actions">
            <button @click="startEditCard(card)">editar</button>
            <button @click="toggleReminder(card.id)">lembrete</button>
          </div>
          <div v-if="openReminderFor === card.id" class="reminder-form">
            <input type="datetime-local" v-model="reminderAt" />
            <textarea v-model="reminderMessage" placeholder="mensagem do lembrete" rows="2" />
            <button @click="addReminder(card.id)">agendar</button>
          </div>
        </template>
      </div>
    </VueDraggable>

    <div class="add-form">
      <input v-model="newCardTitle" placeholder="novo card" @keyup.enter="addCard" />
      <button @click="addCard">+</button>
    </div>
  </section>
</template>
