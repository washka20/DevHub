# Console Improvements — Design Spec

**Date:** 2026-04-01
**Scope:** 6 features для Console: Activity Indicator, Bell Notification, Context Menu, CWD Tracking, Bottom Terminal Panel, Ctrl+` shortcut.

---

## 1. Activity Indicator

**Что:** Пульсирующая синяя точка на табе, когда в неактивном терминале идёт вывод.

**Поведение:**
- Активный таб: зелёная точка (без изменений)
- Неактивный таб получает вывод: точка становится синей с CSS-анимацией `pulse` (1.5s ease-in-out infinite)
- Пользователь переключается на таб → точка возвращается к зелёной/серой
- Idle неактивный таб: серая точка (без изменений)

**Реализация:**
- `TerminalPane` получает поле `hasActivity: boolean` (в store)
- WebTerminal: в `ws.onmessage` при получении binary data, если таб неактивный → `pane.hasActivity = true`
- При `setActiveTab` → сбрасывать `hasActivity = false` для всех panes активного таба
- TerminalTabBar: класс `.tab-dot.activity` с CSS animation

**Файлы:** `stores/terminal.ts`, `components/WebTerminal.vue`, `components/TerminalTabBar.vue`

---

## 2. Bell Notification

**Что:** Когда `\a` (BEL, 0x07) приходит в фоновый таб — таб мигает оранжевым + browser Notification.

**Поведение:**
- xterm.js имеет событие `term.onBell`
- Если таб неактивный: точка на табе мигает оранжевым на 3 секунды
- Параллельно: `Notification.requestPermission()` при первом использовании Console, затем `new Notification('Terminal bell', { body: 'Tab: <label>' })`
- Если таб активный: bell игнорируется (пользователь и так видит терминал)

**Реализация:**
- `TerminalPane` получает поле `hasBell: boolean`
- WebTerminal: `term.onBell(() => { if (tabIsInactive) pane.hasBell = true })`
- Через 3 секунды `pane.hasBell = false`
- При `setActiveTab` → сбрасывать `hasBell = false`
- Browser Notification: запрашивать permission в `ConsoleView.onActivated`, показывать через Notification API
- TerminalTabBar: класс `.tab-dot.bell` с оранжевой анимацией

**Файлы:** `components/WebTerminal.vue`, `stores/terminal.ts`, `components/TerminalTabBar.vue`, `views/ConsoleView.vue`

---

## 3. Context Menu

**Что:** Правый клик по табу → контекстное меню с действиями.

**Пункты меню:**
| Пункт | Действие | Hotkey hint |
|-------|----------|-------------|
| Rename | Inline-редактирование label таба | F2 |
| — separator — | | |
| Split Horizontal | splitPane('horizontal') | |
| Split Vertical | splitPane('vertical') | |
| — separator — | | |
| Close | closeTab(tabId) | |
| Close Others | closeTab для всех кроме текущего | |
| Close All | closeTab для всех | |

**Поведение:**
- `@contextmenu.prevent` на каждом `.tab`
- Меню позиционируется absolute у курсора (с корректировкой чтобы не уходило за край)
- Клик вне меню или Escape → закрытие
- Rename: заменяет span.tab-label на input, Enter/blur → сохранение, Escape → отмена

**Реализация:**
- Новый компонент `TabContextMenu.vue` (или inline в TerminalTabBar)
- State: `contextMenu: { visible: boolean, x: number, y: number, tabId: string } | null`
- Store: `renameTab(tabId, label)`, `closeOtherTabs(tabId)`, `closeAllTabs()`

**Файлы:** `components/TerminalTabBar.vue`, `stores/terminal.ts`

---

## 4. CWD Tracking

**Что:** Реалтайм отображение текущей рабочей директории терминала.

**Два метода (оба, с fallback):**

### 4a. OSC 7 parsing (primary)
- Шелл отправляет `\e]7;file://hostname/path\a` при каждом cd
- bash: включено по умолчанию (PROMPT_COMMAND)
- zsh: включено по умолчанию (chpwd hook)
- Фронт парсит OSC-последовательности из потока xterm
- xterm.js: можно перехватить через `term.parser.registerOscHandler(7, data => ...)`

### 4b. /proc/{pid}/cwd (fallback, Linux)
- Бэкенд: новый эндпоинт `GET /api/terminal/sessions/{id}/cwd`
- Читает `os.Readlink(fmt.Sprintf("/proc/%d/cwd", sess.Cmd.Process.Pid))`
- Фронт: poll каждые 5 секунд если OSC 7 не работает (определяется по отсутствию OSC 7 в первые 10 секунд)

### Отображение
- Floating badge в правом верхнем углу терминала: `~/project/devhub`
- Шрифт 10px, полупрозрачный, фон bg-primary с border
- Сокращение пути: `/home/user/...` → `~/...`
- Обновляет `pane.cwd` в store → autosave в localStorage

**Файлы:**
- Backend: `internal/api/terminal_handlers.go` (новый handler), `internal/server/server.go` (route)
- Frontend: `components/WebTerminal.vue` (OSC parser + badge), `stores/terminal.ts` (pane.cwd update)

---

## 5. Bottom Terminal Panel

**Что:** Терминал-панель снизу на любой странице (кроме /console), как в VS Code.

### Режимы
- **Pinned** (default): горизонтальный Splitpanes в App.vue, drag-resizable сплиттер
- **Floating**: `position:fixed` overlay, draggable header, resizable углы

### Компоненты

**BottomTerminal.vue** — обёртка:
- Показывает TerminalTabBar (compact вариант, height:28px) + WebTerminal
- Кнопки в header: Maximize (→ /console), Float/Pin toggle, Close (Ctrl+`)
- Compact TerminalTabBar: те же табы что и в Console (shared store), но без toolbar кнопок

**App.vue изменения:**
- Wrap `<main>` в `<Splitpanes horizontal>` + `<Pane>` для bottom panel
- `v-if="terminalStore.panel.visible && route.path !== '/console'"`
- При навигации на `/console`: `panel.visible = false` (auto-hide), сохранить prev state
- При уходе с `/console`: restore panel.visible

### Panel State (уже есть в store)
```typescript
panel: {
  mode: 'pinned' | 'floating',
  visible: boolean,
  height: number,        // % для pinned mode
  floatingPos: { x, y, w, h }
}
```

### Floating Mode
- `position:fixed; z-index:1000`
- Draggable по header bar (mousedown/mousemove/mouseup)
- Resize через CSS `resize: both` или custom handles на углах
- Сохраняется в localStorage через panel state

**Файлы:** `components/BottomTerminal.vue` (новый), `App.vue`, `stores/terminal.ts`, `components/TerminalTabBar.vue` (compact prop)

---

## 6. Keyboard Shortcut: Ctrl+`

**Что:** Toggle нижней панели с любой страницы.

**Поведение:**
- `Ctrl+`` → toggle `panel.visible`
- Работает даже когда xterm имеет фокус (перехватывается ДО xterm)
- На `/console`: переключает на другую страницу? Нет — на `/console` панель скрыта, Ctrl+` ничего не делает (или навигирует обратно на предыдущую страницу)
- Если панель скрыта и нет табов: создать новый таб и показать панель

**Реализация:**
- Global `document.addEventListener('keydown')` в App.vue
- Проверка: `e.ctrlKey && e.key === '`'`
- `e.preventDefault()` + `e.stopPropagation()` чтобы не дошло до xterm
- Вызов `terminalStore.togglePanel()`

**Файлы:** `App.vue`

---

## Порядок реализации

1. Activity Indicator + Bell Notification (маленькие, фронтенд-only)
2. Context Menu (фронтенд-only)
3. CWD Tracking (бэкенд + фронтенд)
4. Bottom Terminal Panel + Ctrl+` (большая фича, App.vue + новый компонент)

---

## Иконки

Все иконки — SVG inline, 14-16px, consistent с существующим дизайном (close button в TerminalTabBar). Для bottom panel: maximize, float/pin, close — отдельные SVG.
