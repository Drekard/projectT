# Задача: Адаптивный layout для режима чата

## Цель
Сделать интерфейс компактным в режиме чата, аналогично Telegram: шапка и левая панель меняют содержимое, правая панель скрыта по умолчанию.

## Два режима интерфейса

### Normal Mode (текущий)
- Шапка: `☰ [] ProjectT [+] | Хлебные крошки | 🔍 Поиск | ⚙️ Фильтр`
- Sidebar: иконки + подписи навигации + Favorites
- Workspace: контент выбранной вкладки

### Chat Mode (новый)
- Шапка: `Имя чата | 🔍 Поиск по чату | 📎 | ℹ️ Профиль`
- Sidebar: список чатов (как в `internal/ui/workspace/chats/left`)
- Workspace: центральная область чата
- Right panel: скрыта, открывается кнопкой `ℹ️`

---

## Подзадачи

### 1. Сворачиваемая боковая панель (Normal Mode)
**Файлы:** `internal/ui/sidebar/sidebar.go`, `internal/ui/header/header.go`

- Добавить кнопку ☰ (три полоски) в шапку **перед иконкой приложения** как первый элемент
- По клику переключать состояние `sidebarCollapsed bool`
- В свёрнутом состоянии sidebar показывает только иконки (~50px ширина)
- В развёрнутом — иконки + подписи + Favorites (180px)
- В Chat Mode кнопка ☰ **не отображается**

### 2. Адаптивная шапка
**Файлы:** `internal/ui/header/header.go`, `internal/ui/layout/main_layout.go`

- Добавить параметр режима в `CreateHeader`
- **Normal Mode:** текущая шапка с `☰`, `[+]`, хлебными крошками, фильтром
- **Chat Mode:** 
  - `☰` отсутствует
  - `[+]` исчезает
  - Хлебные крошки → имя чата (без аватара)
  - Фильтр исчезает
  - Добавить кнопку `ℹ️` (открыть правую панель)
  - Добавить кнопку `` — заимствует функционал `profileMoreButton` из `profile_panel.go` (открытие remote профиля собеседника)
  - Поиск остаётся (поиск по сообщениям чата)

### 3. Левая панель в режиме чата
**Файлы:** `internal/ui/sidebar/sidebar.go`, `internal/ui/workspace/chats/left/header.go`, `internal/ui/workspace/chats/left/chat_list.go`

- При переключении на вкладку Chats sidebar автоматически показывает список чатов
- Скопировать логику отображения из `internal/ui/workspace/chats/left` в sidebar
- Кнопка `[←]` находится в `internal/ui/workspace/chats/left/header.go` **первой кнопкой** (возврат к Normal Mode)
- Удалить `internal/ui/workspace/chats/left` после переноса логики

### 4. Правая панель (скрыта по умолчанию)
**Файлы:** `internal/ui/workspace/chats/chats.go`, `internal/ui/workspace/chats/right/`

- По умолчанию `rightPanel` скрыта
- Кнопка `ℹ️` в шапке чата toggles видимость правой панели
- Анимация slide-in справа
- Правая панель содержит: профиль собеседника + витрина элементов

### 5. Удаление left panel из chats
**Файлы:** `internal/ui/workspace/chats/chats.go`

- Убрать `leftPanel` из `createViewContent()`
- Центральная область чата теперь занимает всё пространство между sidebar и right panel
- Layout: `container.NewBorder(nil, nil, sidebar, rightPanel, chatArea)`

---

## Состояния и переключение

```
Normal Mode ←[клик на Chats]→ Chat Mode
     ↑                              |
     |________[← в left/header.go]__|
```

- `sidebarCollapsed bool` — свёрнута ли боковая панель
- `isChatMode bool` — активен ли режим чата
- `rightPanelVisible bool` — видима ли правая панель

---

## Файлы для изменения

| Файл | Изменения |
|---|---|
| `internal/ui/header/header.go` | Адаптивная шапка, кнопка ☰ перед иконкой |
| `internal/ui/sidebar/sidebar.go` | Сворачивание, список чатов в Chat Mode |
| `internal/ui/layout/main_layout.go` | Передача состояний между компонентами |
| `internal/ui/workspace/chats/chats.go` | Удалить left panel, right panel по требованию |
| `internal/ui/workspace/chats/left/` | Перенести логику в sidebar, удалить |
| `internal/ui/workspace/workspace.go` | Callback переключения режима |
