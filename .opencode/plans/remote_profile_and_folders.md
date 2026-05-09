# План: Просмотр удалённого профиля, навигация по элементам и отправка папок через чат

## Что уже сделано (сессия 1)

| Компонент | Описание |
|-----------|----------|
| **Миграция БД** | `parent_id` → `parent_uuid` в `migrations.go`, авто-миграция данных, индекс `idx_items_parent_uuid` |
| **Модели** | `Item.ParentUUID`, `PinnedItem.ItemElementUUID` |
| **Queries** | `GetItemsByParentUUID`, `GetItemsByParentUUIDs`, `GetElementUUIDByID`, `GetElementUUIDsByIDs`, `GetSavedItemsByParentUUID`, `GetPreviewItemsByParentUUID`, `GetRemoteItemsByElementUUIDs`, `GetPinnedItemUUIDs` |
| **Batch Transfer** | `SendBatch`, `SendFolder`, `SendPinnedItems`, `SendSelection`, `BatchProgress`, `BatchItemProgress`, handler `handleBatchTransferRequest` |
| **ItemSync batch** | `RequestBatchByUUIDs`, `RequestFolder`, `ItemResponse` с `ParentUUID` |
| **Core accessors** | Batch методы в `P2PNetwork` |
| **UI API** | Batch методы в `UIP2P` |
| **UI Batch Widget** | `BatchProgressWidget` — общий прогресс + список элементов |
| **Скрипт миграции** | `tools/migrate/main.go` — успешно мигрировал 33 записи |

---

## Фаза 1: Просмотр удалённого профиля в workspace

### 1.1. Новый тип контента "remote_profile"

**Файл:** `internal/ui/workspace/workspace.go`

- Добавить `ContentTypeRemoteProfile = "remote_profile"` в enum `ContentType`
- Добавить в `Workspace` поля:
  ```go
  remoteProfilePeerID  string
  remoteProfileUI      *profile.RemoteProfileUI
  ```
- В `UpdateContent()` добавить обработку `"remote_profile"` — создание/кэширование view

### 1.2. RemoteProfileUI — адаптер профиля для удалённого пользователя

**Новый файл:** `internal/ui/workspace/profile/remote_profile.go`

Создать `RemoteProfileUI` — read-only версию `profile.UI`:
- Загружает данные из `queries.GetRemoteProfile(peerID)`
- Отображает: аватар, имя, title, характеристики (из `ContentChar`)
- Вместо `GridManager` для pinned items — использует **batch запрос** через `ItemSync`
- Кнопки редактирования скрыты (read-only)
- Кнопка "Открыть элементы" → навигация к saved элементам этого пира

**Структура:**
```go
type RemoteProfileUI struct {
    content                  fyne.CanvasObject
    profileAvatar            *canvas.Image
    profileName              *widget.Label
    profileTitle             *widget.Label
    characteristicsContainer *fyne.Container
    gridManager              *saved.GridManager // для pinned items
    peerID                   string
    p2pUI                    *network.UIP2P
    workspace                *Workspace // ссылка для навигации
}
```

### 1.3. Загрузка pinned items удалённого профиля

**Flow:**
1. При открытии remote профиля → читаем `PinnedUUIDs` из `profiles` таблицы
2. Если элементы уже есть в локальной БД (`owner_type='remote', source_peer_id=peerID`) → отображаем из БД
3. Если элементов нет или их меньше чем в `PinnedUUIDs` → вызываем `ItemSync.RequestBatchByUUIDs(peerID, pinnedUUIDs)`
4. После загрузки → `gridManager.LoadItemsWithoutCreateElement(items)`

**Метод:** `RemoteProfileUI.LoadPinnedItems()`

### 1.4. Открытие remote профиля из UI — ИСТОЧНИКИ ВЫЗОВА

**Поправка:** Переход на расширенную версию профиля удалённого пользователя происходит через:

#### А) Кнопка "⋯" (троеточие) в правой панели чата
**Файл:** `internal/ui/workspace/chats/right/profile_panel.go`

- Добавить кнопку "⋯" (или "View Full Profile") рядом с именем/аватаром в `createProfileArea()`
- При клике → `workspace.OpenRemoteProfile(peerID)`
- Отображается только для remote контактов (не для `IsLocalChat()`)

#### Б) Отдельная кнопка в списке контактов
**Файл:** `internal/ui/workspace/contacts/contacts.go`

- В `createContactItem()` добавить кнопку "👤" / "Profile" рядом с кнопками Chat/Connect/Delete
- При клике → `workspace.OpenRemoteProfile(contact.PeerID)`
- Требуется обновить `UIProvider` интерфейс: добавить `OpenRemoteProfile(peerID string)`

#### В) Отдельная кнопка в списке пиров (P2P вкладка)
**Файл:** `internal/ui/workspace/p2p/connections.go`

- В `createConnectedPeerItem()` — добавить кнопку "Profile" рядом с Chat
- В `createProfileItem()` — добавить кнопку "Profile" рядом с Chat
- При клике → `workspace.OpenRemoteProfile(peerID)`

**Метод:** `Workspace.OpenRemoteProfile(peerID string)`
```go
func (ws *Workspace) OpenRemoteProfile(peerID string) {
    ws.remoteProfilePeerID = peerID
    ws.UpdateContent("remote_profile")
}
```

---

## Фаза 2: Навигация по элементам удалённого пользователя

### 2.1. Remote Navigation Manager

**Новый файл:** `internal/ui/workspace/navigation/remote_navigation.go`

Создать `RemoteNavigationManager` — аналог `NavigationManager` но для удалённых элементов:
```go
type RemoteNavigationManager struct {
    currentParentUUID  string
    folderStack        []string // stack of parent_uuids
    peerID             string
    peerName           string
    onBreadcrumbUpdate func([]*RemoteBreadcrumbItem)
}
```

### 2.2. Remote Breadcrumbs — ПОПРАВКА

**Файл:** `internal/ui/header/breadcrumbs.go`

**Поведение:**
- Надпись **"Сохраненное"** отображается ТОЛЬКО при просмотре папок/профиля локального пользователя
- При просмотре профиля/папок/элементов **удалённого** пользователя — надпись "Сохраненное" заменяется на **имя удалённого профиля** во ВСЕХ случаях
- Первый элемент breadcrumbs — имя удалённого пользователя (кликабельная ссылка → `workspace.OpenRemoteProfile(peerID)`)
- Остальные элементы — путь по папкам

**Визуал для локального профиля:**
```
[Сохраненное] > [Документы] > [Отчёты]
```

**Визуал для удалённого профиля (профиль, папки, элементы):**
```
[Иван Иванов] > [Документы] > [Отчёты]
```

**Методы:**
- `UpdateBreadcrumbs(path []*models.Item)` — для локального (первый элемент = "Сохраненное")
- `UpdateRemoteBreadcrumbs(peerName string, peerID string, path []*models.Item)` — для удалённого (первый элемент = имя пира)
- Клик на имя пира → `workspace.OpenRemoteProfile(peerID)`

### 2.3. Переключение между локальной и удалённой навигацией

**Файл:** `internal/ui/layout/main_layout.go`

- При навигации в remote элементы → заменить callback breadcrumbs на `UpdateRemoteBreadcrumbs`
- При возврате к локальным элементам → восстановить обычный `UpdateBreadcrumbs`
- Добавить метод `MainLayout.SwitchToRemoteMode(peerID, peerName string)`

### 2.4. Загрузка элементов по parent_uuid

**Flow:**
1. Пользователь кликает на remote профиль → открывается профиль с pinned items
2. Пользователь кликает "Элементы" → `RemoteNavigationManager` устанавливает `currentParentUUID = ""` (корень)
3. Вызывается `queries.GetSavedItemsByParentUUID("")` для remote элементов (`owner_type='remote', source_peer_id=peerID`)
4. GridManager отображает элементы
5. Клик на папку → `GetSavedItemsByParentUUID(folder.ElementUUID)` → навигация глубже

---

## Фаза 3: Отправка папок через чат

### 3.1. Снятие фильтра с кнопки "Send" для папок

**Файл:** `internal/ui/cards/hover_preview/menu_manager.go`

**Поправка:** В текущем коде кнопка "Send" отображается для всех типов элементов (строка ~246: `// Добавляем кнопку отправки для всех типов элементов`). Фильтра по типу элемента нет — кнопка "Send" уже работает для папок. Однако `sendItemToChat()` отправляет элемент через `SendElementMessage`, который работает только для одиночных элементов.

**Что нужно сделать:**
- Адаптировать `sendItemToChat()` для распознавания типа элемента
- Если `item.Type == "folder"` → использовать `SendFolder(peerID, item.ElementUUID)` вместо `SendElementMessage`
- Сообщение сохраняется с `content_type="folder_batch"` и metadata (folder_uuid, folder_title, item_count)

### 3.2. Новый тип сообщения "folder_batch"

**Файл:** `internal/services/p2p/protocols/chat/chat.go`

Добавить `ContentType = "folder_batch"`:
```go
type FolderBatchMetadata struct {
    FolderUUID   string `json:"folder_uuid"`
    FolderTitle  string `json:"folder_title"`
    ItemCount    int    `json:"item_count"`
    OwnerPeerID  string `json:"owner_peer_id"`
    OwnerName    string `json:"owner_name"`
    BatchID      string `json:"batch_id"`
}
```

### 3.3. Адаптация chat_panel.go для отправки/получения папок

**Файл:** `internal/ui/workspace/chats/center/chat_panel.go`

**Поправка:** `chat_panel.go` нужно приспособить для:

#### Отправка:
- Кнопка "Send" в контекстном меню папки (из `menu_manager.go`) → вызывает `sendItemToChat()` → распознаёт тип folder → вызывает `SendFolder()` через batch transfer
- Сообщение сохраняется в БД с `content_type="folder_batch"`

#### Получение/отображение:
- В `NewMessageBubble` добавить обработку `ContentType == "folder_batch"`:
  - Извлекает metadata (folder_uuid, folder_title, item_count)
  - Создаёт карточку папки с иконкой 📁, названием, количеством элементов
  - При клике → загружает содержимое папки от пира через `ItemSync.RequestFolder()`

**Новый метод:** `MessageBubble.createBubbleForFolderBatch()`

Карточка папки в сообщении:
```
┌─────────────────────────┐
│ 📁 Документы            │
│ 12 элементов            │
│ [Открыть]               │
└─────────────────────────┘
```

### 3.4. Отправка папки из чата

**Файл:** `internal/ui/workspace/chats/chats.go`

Добавить метод `sendFolderToChat(peerID, parentUUID string)`:
1. Вызывает `p2pUI.SendFolder(peerID, parentUUID)` → batch transfer
2. После успешной отправки → сохраняет сообщение в БД с `content_type="folder_batch"` и metadata
3. UI обновляет список сообщений

---

## Фаза 4: Загрузка содержимого папки из чата

### 4.1. Обработка клика на папку в чате

**Flow:**
1. Пользователь кликает "Открыть" на карточке папки в чате
2. Вызывается `ItemSync.RequestFolder(peerID, folderUUID)`
3. Загруженные элементы сохраняются в БД с `owner_type='remote', source_peer_id=peerID, parent_uuid=folderUUID`
4. Workspace переключается на remote saved view
5. Breadcrumbs: `[Имя владельца] > [Название папки]`

### 4.2. Переключение workspace на remote saved view

**Файл:** `internal/ui/workspace/workspace.go`

Добавить метод `OpenRemoteFolder(peerID, folderUUID, folderTitle string)`:
```go
func (ws *Workspace) OpenRemoteFolder(peerID, folderUUID, folderTitle string) {
    ws.remoteProfilePeerID = peerID
    ws.remoteFolderUUID = folderUUID
    ws.remoteFolderTitle = folderTitle
    ws.UpdateContent("remote_saved")
}
```

### 4.3. Remote Saved View

**Новый тип контента:** `"remote_saved"`

Аналогичен `"saved"` но:
- Загружает элементы через `GetSavedItemsByParentUUID(parentUUID)` с фильтром `source_peer_id=peerID`
- Breadcrumbs показывают имя владельца вместо "Сохраненное"
- Клик на имя владельца → `OpenRemoteProfile(peerID)`
- Клик на папку → навигация глубже по `parent_uuid`

---

## Фаза 5: Интеграция и связывание

### 5.1. Sidebar: кнопка Saved возвращает к локальной сортировке

**Файл:** `internal/ui/sidebar/navigation.go`

**Поправка:** Клик по `savedButton` должен возвращать на изначальную версию сортировки элементов локального профиля из ЛЮБОГО состояния (включая remote profile, remote saved и т.д.):
- Сбрасывает `remoteProfilePeerID`, `remoteFolderUUID` в workspace
- Восстанавливает обычные breadcrumbs ("Сохраненное")
- Вызывает `workspace.UpdateContent("saved")` с `parentID = 0`
- Переключает навигацию обратно на локальный режим

### 5.2. ChatController — новые методы

**Файл:** `internal/controllers/chat_controller.go`

- `SendFolderToChat(contactID, parentUUID string) error`
- `LoadRemoteFolder(peerID, folderUUID string) ([]*models.Item, error)`
- `LoadRemoteProfileItems(peerID string) ([]*models.Item, error)`

### 5.3. Связь breadcrumbs с навигацией

**Файл:** `internal/ui/layout/main_layout.go`

- При клике на имя владельца в breadcrumbs → `workspace.OpenRemoteProfile(peerID)`
- При клике на папку в breadcrumbs → `workspace.NavigateToRemoteFolder(parentUUID)`

### 5.4. Обновление правой панели чата

**Файл:** `internal/ui/workspace/chats/right/profile_panel.go`

- Добавить кнопку "⋯" / "View Full Profile" → `workspace.OpenRemoteProfile(peerID)`
- Кнопка "Открыть элементы" → `workspace.OpenRemoteSaved(peerID)`

---

## Сводка файлов для изменения/создания

| Файл | Действие | Описание |
|------|----------|----------|
| `internal/ui/workspace/workspace.go` | Изменить | +remote_profile, +remote_saved типы, +методы навигации |
| `internal/ui/workspace/profile/remote_profile.go` | **Создать** | Read-only профиль удалённого пользователя |
| `internal/ui/workspace/navigation/remote_navigation.go` | **Создать** | Навигация по удалённым элементам |
| `internal/ui/header/breadcrumbs.go` | Изменить | +UpdateRemoteBreadcrumbs, имя владельца вместо "Сохраненное" для remote |
| `internal/ui/layout/main_layout.go` | Изменить | Переключение между локальной/удалённой навигацией |
| `internal/ui/sidebar/navigation.go` | Изменить | Saved button → сброс remote режима, возврат к локальной сортировке |
| `internal/ui/workspace/chats/chats.go` | Изменить | +sendFolderToChat |
| `internal/ui/workspace/chats/center/chat_panel.go` | Изменить | +createBubbleForFolderBatch, адаптация для folder send/receive |
| `internal/ui/workspace/chats/dialogs/folder_send_dialog.go` | **Создать** | Диалог выбора папки для отправки (если нужен) |
| `internal/ui/workspace/chats/right/profile_panel.go` | Изменить | +кнопка "⋯" для открытия полного профиля |
| `internal/ui/workspace/contacts/contacts.go` | Изменить | +кнопка "Profile" в contact item |
| `internal/ui/workspace/p2p/connections.go` | Изменить | +кнопка "Profile" в connected peer item и profile item |
| `internal/controllers/chat_controller.go` | Изменить | +методы для папок и remote элементов |
| `internal/services/p2p/protocols/chat/chat.go` | Изменить | +ContentType "folder_batch" |
| `internal/ui/cards/hover_preview/menu_manager.go` | Изменить | Адаптировать sendItemToChat для folder → SendFolder |

---

## Зависимости между фазами

```
Фаза 1 (Remote Profile + точки входа)
    ↓
Фаза 2 (Remote Navigation + Breadcrumbs с именем владельца)
    ↓
Фаза 3 (Send Folder через chat — адаптация menu_manager + chat_panel)
    ↓
Фаза 4 (Load Folder from Chat)
    ↓
Фаза 5 (Integration — Sidebar Saved reset, ChatController, breadcrumbs)
```

Каждая фаза зависит от предыдущей. Фаза 5 — финальная интеграция всех компонентов.
