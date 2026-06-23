# jboard desktop — build nativo no Windows

Como gerar e instalar o app **Windows** do jboard (Tauri), apontando pro backend
de **produção** (`https://jboard.devarthur.com.br`). O app abre uma janela nativa,
pede **login com senha** (a mesma do site) e funciona normalmente.

> **Por que não no WSL?** O Tauri no WSL gera um binário **Linux** (webkit2gtk).
> Pra um `.exe`/instalador **Windows** de verdade (WebView2 + toolchain MSVC), o
> build precisa rodar no **Windows nativo**. Faça tudo no PowerShell do Windows.

---

## 1. Pré-requisitos (instalar uma vez)

| Item | Como |
|------|------|
| **Git** | https://git-scm.com/download/win |
| **Node.js LTS** | https://nodejs.org (inclui o npm) |
| **Rust (MSVC)** | https://rustup.rs → roda `rustup-init.exe`; aceite o padrão (`stable-x86_64-pc-windows-msvc`) |
| **Visual Studio Build Tools** | https://visualstudio.microsoft.com/visual-cpp-build-tools/ → instale o workload **"Desktop development with C++"** (traz o linker MSVC) |
| **WebView2 Runtime** | Já vem no **Windows 11**. No Windows 10, instale o *Evergreen Runtime*: https://developer.microsoft.com/microsoft-edge/webview2/ |

Confira no PowerShell (abra um terminal **novo** depois de instalar):

```powershell
git --version
node -v
rustc --version
cargo --version
```

---

## 2. Clonar o projeto

```powershell
git clone https://github.com/Arthur-Queiroz/jboard C:\prog\jboard
cd C:\prog\jboard\frontend
npm install
```

> Clone do GitHub em vez de copiar do WSL — evita arrastar `node_modules`/`target`
> com binários de Linux.

---

## 3. Buildar o instalador (apontando pra produção)

O app empacotado fala direto com a API de produção. Defina a URL **antes** de
buildar e rode o build (o Tauri compila o Vue e o binário Rust):

```powershell
$env:VITE_JBOARD_API_BASE = "https://jboard.devarthur.com.br/api"
npm run tauri:build
```

- **Não precisa** de token: o app pede **login com senha** ao abrir (igual ao site).
- O primeiro build baixa/compila as dependências Rust — pode levar alguns minutos.

Alternativa sem o CLI do npm (usando o cargo direto):

```powershell
cargo install tauri-cli --version "^2"   # uma vez
cd ..\desktop
cargo tauri build
```

---

## 4. Instalar e usar

O instalador sai em:

```
C:\prog\jboard\desktop\src-tauri\target\release\bundle\
```

- **`nsis\jboard_0.1.0_x64-setup.exe`** — instalador (recomendado), ou
- **`msi\jboard_0.1.0_x64_en-US.msi`** — pacote MSI.

Rode o instalador → o jboard aparece no menu Iniciar. Ao abrir:

1. surge a **tela de login** → digite sua **senha** (a mesma de `jboard.devarthur.com.br`);
2. pronto — o quadro carrega da produção. A sessão fica salva no app (token de
   sessão guardado localmente); pra sair, use o botão **sair** no canto superior.

> O app tem **tray icon** e **autostart** (inicia com o Windows). Fechar a janela
> manda pra bandeja; clicar no ícone mostra/esconde; "Sair" encerra de fato.

---

## 5. Atualizar o app depois

A UI fica embutida no binário, então atualizar = rebuildar:

```powershell
cd C:\prog\jboard
git pull
cd frontend
npm install
$env:VITE_JBOARD_API_BASE = "https://jboard.devarthur.com.br/api"
npm run tauri:build
```

Rode o novo instalador por cima.

---

## 6. (Opcional) Rodar em modo dev

`npm run tauri:dev` abre a janela com hot-reload, mas carrega de
`http://localhost:5173` e faz proxy `/api → localhost:8080` — ou seja, espera um
**backend local**. Apontar o dev pra produção exigiria liberar a origem
`http://localhost:5173` no CORS do backend (`JBOARD_CORS_ORIGINS`), o que **não**
está ligado por padrão. Para uso normal, prefira o **build** do passo 3.

---

## Troubleshooting

- **`link.exe`/erro de linker** → faltou o workload **"Desktop development with C++"** do Visual Studio Build Tools.
- **Tela branca / app não carrega** → WebView2 Runtime ausente (Windows 10): instale o Evergreen.
- **`cargo`/`rustc` não encontrado** → abra um PowerShell novo após instalar o rustup (PATH).
- **Login dá "senha incorreta"** → confirme que está usando a senha de produção (`JBOARD_AUTH_PASSWORD` do servidor). Sem internet, o app não alcança a produção.
