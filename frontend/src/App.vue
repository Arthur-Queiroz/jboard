<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Board, type BoardSummary } from './api'
import ColumnView from './components/ColumnView.vue'

const boards = ref<BoardSummary[]>([])
const selected = ref<Board | null>(null)
const newBoardTitle = ref('')
const newColumnTitle = ref('')
const editingBoardTitle = ref('')
const error = ref('')
const loading = ref(false)

async function loadBoards() {
  loading.value = true
  try {
    boards.value = await api.listBoards()
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

async function selectBoard(id: number) {
  loading.value = true
  try {
    selected.value = await api.getBoard(id)
    editingBoardTitle.value = selected.value.title
    error.value = ''
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

async function addBoard() {
  const title = newBoardTitle.value.trim()
  if (!title) return
  try {
    const created = await api.createBoard(title)
    newBoardTitle.value = ''
    await loadBoards()
    await selectBoard(created.id)
  } catch (e) {
    error.value = String(e)
  }
}

async function renameBoard() {
  if (!selected.value) return
  const title = editingBoardTitle.value.trim()
  if (!title || title === selected.value.title) return
  try {
    await api.updateBoard(selected.value.id, title)
    selected.value.title = title
    await loadBoards()
  } catch (e) {
    error.value = String(e)
  }
}

async function removeBoard(id: number) {
  try {
    await api.deleteBoard(id)
    if (selected.value?.id === id) selected.value = null
    await loadBoards()
  } catch (e) {
    error.value = String(e)
  }
}

async function addColumn() {
  if (!selected.value) return
  const title = newColumnTitle.value.trim()
  if (!title) return
  try {
    const column = await api.createColumn(selected.value.id, title, selected.value.columns.length)
    selected.value.columns.push({ ...column, cards: [] })
    newColumnTitle.value = ''
  } catch (e) {
    error.value = String(e)
  }
}

// Recarrega o board selecionado após uma mutação de coluna/card que precise
// sincronizar com o servidor (ex.: erro de reorder que precisa reverter).
async function reloadSelected() {
  if (selected.value) await selectBoard(selected.value.id)
}

onMounted(loadBoards)
</script>

<template>
  <div class="app">
    <div class="topbar">
      <h1>jboard</h1>
      <div class="boards">
        <button
          v-for="board in boards"
          :key="board.id"
          class="board-pill"
          :class="{ active: selected?.id === board.id }"
          @click="selectBoard(board.id)"
        >
          {{ board.title }}
        </button>
      </div>
      <input v-model="newBoardTitle" placeholder="novo quadro" @keyup.enter="addBoard" />
      <button @click="addBoard">+</button>
    </div>

    <div v-if="selected" class="subbar">
      <input v-model="editingBoardTitle" @keyup.enter="renameBoard" @blur="renameBoard" />
      <button @click="removeBoard(selected.id)" class="danger">excluir quadro</button>
    </div>

    <div v-if="error" class="error">{{ error }}</div>

    <div class="content">
      <div v-if="loading && !selected" class="empty">carregando…</div>
      <div v-else-if="!selected" class="empty">Selecione ou crie um quadro.</div>
      <div v-else class="board">
        <ColumnView
          v-for="column in selected.columns"
          :key="column.id"
          :column="column"
          @changed="reloadSelected"
        />
        <div class="column add-column">
          <div class="add-form">
            <input v-model="newColumnTitle" placeholder="nova coluna" @keyup.enter="addColumn" />
            <button @click="addColumn">+</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
