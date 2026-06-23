<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, UnauthorizedError, type Board, type BoardSummary } from './api'
import ColumnView from './components/ColumnView.vue'

const boards = ref<BoardSummary[]>([])
const selected = ref<Board | null>(null)
const newBoardTitle = ref('')
const newColumnTitle = ref('')
const editingBoardTitle = ref('')
const error = ref('')
const loading = ref(false)

// Tema dark/light persistido em localStorage; aplicado via data-theme no <html>.
const theme = ref<'dark' | 'light'>(
  (localStorage.getItem('jboard-theme') as 'dark' | 'light') || 'dark',
)
function applyTheme() {
  document.documentElement.setAttribute('data-theme', theme.value)
}
function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  localStorage.setItem('jboard-theme', theme.value)
  applyTheme()
}

// Login (web): quando a API responde 401, mostra a tela de senha.
const needLogin = ref(false)
const password = ref('')
const loginError = ref('')
const loggingIn = ref(false)

async function doLogin() {
  if (!password.value) return
  loggingIn.value = true
  loginError.value = ''
  try {
    await api.login(password.value)
    needLogin.value = false
    password.value = ''
    await loadBoards()
  } catch (e) {
    loginError.value = e instanceof UnauthorizedError ? 'Senha incorreta.' : String(e)
  } finally {
    loggingIn.value = false
  }
}

async function logout() {
  await api.logout()
  needLogin.value = true
  selected.value = null
  boards.value = []
}

async function loadBoards() {
  loading.value = true
  try {
    boards.value = await api.listBoards()
  } catch (e) {
    if (e instanceof UnauthorizedError) {
      needLogin.value = true
    } else {
      error.value = String(e)
    }
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

onMounted(() => {
  applyTheme()
  loadBoards()
})
</script>

<template>
  <!-- Login (web): a SPA não tem token; autentica por senha → cookie de sessão. -->
  <div v-if="needLogin" class="login">
    <form class="login-card" @submit.prevent="doLogin">
      <div class="login-brand">
        <div class="brand-mark">j</div>
        <span class="brand-name">board</span>
      </div>
      <input
        v-model="password"
        class="field"
        type="password"
        placeholder="senha"
        autocomplete="current-password"
      />
      <button type="submit" class="btn-accent" :disabled="loggingIn">
        {{ loggingIn ? 'entrando…' : 'entrar' }}
      </button>
      <p v-if="loginError" class="error">{{ loginError }}</p>
    </form>
  </div>

  <div v-else class="app">
    <div class="topbar">
      <div class="brand">
        <div class="brand-mark">j</div>
        <span class="brand-name">board</span>
      </div>

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

      <div class="topbar-actions">
        <input
          v-model="newBoardTitle"
          class="field field-board"
          placeholder="novo quadro"
          @keyup.enter="addBoard"
        />
        <button class="primary-btn" title="Criar quadro" @click="addBoard">+</button>

        <button class="icon-btn" title="Alternar tema" @click="toggleTheme">
          <svg
            v-if="theme === 'dark'"
            width="16" height="16" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
          >
            <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" />
          </svg>
          <svg
            v-else
            width="16" height="16" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
          >
            <circle cx="12" cy="12" r="4" />
            <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M6.3 17.7l-1.4 1.4M19.1 4.9l-1.4 1.4" />
          </svg>
        </button>

        <button class="icon-btn" title="Sair" @click="logout">
          <svg
            width="16" height="16" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
          >
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
            <polyline points="16 17 21 12 16 7" />
            <line x1="21" y1="12" x2="9" y2="12" />
          </svg>
        </button>
      </div>
    </div>

    <div v-if="selected" class="subbar">
      <input
        v-model="editingBoardTitle"
        class="field"
        @keyup.enter="renameBoard"
        @blur="renameBoard"
      />
      <button class="danger-btn" @click="removeBoard(selected.id)">excluir quadro</button>
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
        <div class="add-column">
          <div class="add-form">
            <input v-model="newColumnTitle" placeholder="nova coluna" @keyup.enter="addColumn" />
            <button @click="addColumn">+</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
