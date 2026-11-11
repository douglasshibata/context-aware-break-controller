# 📘 Tutorial de Operação: Context-Aware Workspace Well-Being & Break Controller

Guia passo a passo para configuração, execução do daemon de segundo plano, utilização do CLI de testes e resolução de problemas no projeto **Context-Aware Workspace Well-Being & Break Controller** (`context-aware-break-controller`).

---

## 🛠️ 1. Visão Geral

O **Context-Aware Break Controller** é um daemon local inteligente projetado para substituir cronômetros estáticos por intervenções de saúde ergonômica disparadas exclusivamente em momentos de **transição cognitiva natural**.

- **Linguagem:** Go 1.24+
- **Persistência:** SQLite local (`modernc.org/sqlite`, sem CGO)
- **Privacidade:** 100% Edge/Local (sem envio de dados para nuvem ou gravação de microfone/teclado)

---

## 📋 2. Pré-requisitos

- **Go 1.24+** instalado no ambiente (`go version`).
- **Docker & Docker Compose** (opcional, para execução isolada em container).

---

## 🚀 3. Execução do Daemon em Segundo Plano

### Passo 1: Navegar até a pasta do módulo
```bash
cd context-aware-break-controller
```

### Passo 2: Inicializar o Daemon
```bash
go run ./cmd/daemon
```

O daemon iniciará o monitoramento automático de periféricos, modo de apresentação e cadência de input a cada 10 segundos, atualizando o modelo de fadiga a cada 30 segundos.

---

## 📊 4. Comandos do CLI (`break-ctl`)

O binário CLI permite auditar o histórico diário local e simular cenários de transição cognitiva:

### 1. Consultar Resumo Diário de Bem-Estar
```bash
go run ./cmd/cli stats
```
*Exemplo de saída:*
```text
📊 Local Well-Being Summary for Today (2026-11-07)
--------------------------------------------------
⏱️  Active Focus Minutes : 45 min
📞 Meetings Completed   : 2
🍵 Breaks Taken         : 3
⚡ Avg Fatigue Score    : 38.50 / 100
🚫 Rapid Dismissals (<2s): 0
```

### 2. Simular Fim de Reunião com Decompressão
```bash
go run ./cmd/cli simulate --scenario meeting-end
```

### 3. Simular Supressão em Modo Apresentação / Fullscreen (DND)
```bash
go run ./cmd/cli simulate --scenario dnd-suppression
```

### 4. Simular Mitigação de Notificações Rápidas (Notification Blindness)
```bash
go run ./cmd/cli simulate --scenario rapid-dismissal
```

---

## 🐳 5. Execução em Container Docker

Para validar e executar a imagem compilada via Docker Compose:

```bash
# Validar sintaxe Compose V2
docker compose config

# Construir e inicializar o container
docker compose up -d
```

---

## 🧪 6. Execução de Testes Automatizados

Para rodar todos os testes unitários e de integração:

```bash
cd context-aware-break-controller
go test -count=1 -v ./...
```
