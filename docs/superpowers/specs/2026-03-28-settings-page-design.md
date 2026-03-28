# Settings Page -- Design Spec

## Problem

DevHub имеет захардкоженные значения (projects dir, shell, terminal font/theme, max sessions) которые можно менять только в `~/.devhub.yaml` вручную. Нет UI для настроек. Проект привязан к конкретной системе.

## Requirements

- Страница Settings (`/settings`) в sidebar
- Настройки General: projects directory, default project, server port
- Настройки Terminal: shell (bash/zsh/fish), font size, font family, scrollback, cursor blink, max sessions
- Настройки Theme: terminal color theme presets (Dracula, Nord, One Dark, Monokai, GitHub Dark, Solarized)
- Серверные настройки в `~/.devhub.yaml` с hot reload
- UI-настройки в `localStorage` с мгновенным применением
- Авто-определение доступных shell'ов на системе

## Architecture

### Storage -- два уровня

**Backend (`~/.devhub.yaml`)** -- для серверных настроек:
```yaml
port: 9000
projects_dir: ~/project
default_project: cfa
terminal:
  max_sessions: 10
  shell: /bin/bash
```

**Frontend (`localStorage`)** -- для UI-настроек:
```json
{
  "terminal.fontSize": 14,
  "terminal.fontFamily": "JetBrains Mono, SF Mono, Fira Code, monospace",
  "terminal.scrollback": 10000,
  "terminal.cursorBlink": true,
  "terminal.theme": "github-dark"
}
```

### API

```
GET  /api/settings        -- текущие серверные настройки (из yaml)
PUT  /api/settings        -- обновить настройки (пишет в yaml, hot reload)
GET  /api/settings/shells -- доступные shell'ы на системе (сканирует /etc/shells)
```

### UI Structure

```
Settings
├── General
│   ├── Projects Directory    [/home/washka/project]
│   ├── Default Project       [cfa ▼]
│   └── Server Port           [9000] (requires restart)
│
├── Terminal
│   ├── Shell                 [bash ▼] (auto-detected)
│   ├── Font Size             [14] px
│   ├── Font Family           [JetBrains Mono ▼]
│   ├── Scrollback            [10000] lines
│   ├── Cursor Blink          [✓]
│   └── Max Sessions          [10]
│
└── Theme
    ├── Terminal Theme        [GitHub Dark ▼]
    └── (live preview terminal)
```

### Terminal Theme Presets

Каждая тема -- 16 ANSI цветов + background/foreground/cursor/selection:

- **GitHub Dark** (текущая) -- `#0d1117` bg, `#c9d1d9` fg
- **Dracula** -- `#282a36` bg, `#f8f8f2` fg
- **One Dark** -- `#282c34` bg, `#abb2bf` fg
- **Nord** -- `#2e3440` bg, `#d8dee9` fg
- **Monokai** -- `#272822` bg, `#f8f8f2` fg
- **Solarized Dark** -- `#002b36` bg, `#839496` fg
- **Tokyo Night** -- `#1a1b26` bg, `#c0caf5` fg

## Settings that stay hardcoded (NOT configurable)

- HTTP timeouts (15s read, 120s idle) -- too technical
- CORS origins -- localhost only
- Resize debounce (50ms) -- optimal
- TERM env var (xterm-256color) -- covers 99.9%
- Glow effects, transitions -- overconfiguration

## File Summary

| File | Type | ~Lines |
|------|------|--------|
| `internal/api/settings_handlers.go` | New | 150 |
| `internal/config/config.go` | Modified | +30 |
| `internal/server/server.go` | Modified | +10 |
| `frontend/src/stores/settings.ts` | New | 100 |
| `frontend/src/views/SettingsView.vue` | New | 300 |
| `frontend/src/data/terminal-themes.ts` | New | 120 |
| `frontend/src/components/WebTerminal.vue` | Modified | +20 |
| `frontend/src/router/index.ts` | Modified | +5 |
| `frontend/src/components/AppSidebar.vue` | Modified | +4 |
| **Total** | | **~750** |

## Phased Implementation

1. **Phase 1 (MVP):** Shell selection + Projects dir + Font size/family
2. **Phase 2:** Terminal theme presets with live preview
3. **Phase 3:** Full theme customization + advanced settings

## Verification

1. `/settings` -- страница открывается
2. Сменить projects dir → проекты пересканированы
3. Сменить shell → новый терминал использует выбранный shell
4. Сменить font size → терминал обновляется мгновенно
5. Сменить theme → терминал перекрашивается мгновенно
6. Перезагрузить страницу → настройки сохранились
