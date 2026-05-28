# 📋 План реализации групповых чатов и каналов

**Дата:** 28 мая 2026
**Зависимости:** Стабильное P2P соединение, PubSub (GossipSub) инициализирован

---

## 🎯 Концепция

Три типа коммуникации с разными моделями нагрузки:

| Тип | Участники | Модель | Доставка |
|-----|-----------|--------|----------|
| **Маленький чат** | 2-10 | Full mesh, все хранят всё | Распределённая |
| **Группа** | 10-50 | Mesh + хранители | Частично распределённая |
| **Канал** | 50+ | Broadcast | Админ публикует, подписчики читают |

---

## 🗄️ База данных — новые таблицы

### 1. `group_chats`

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | INTEGER PK | Автоинкремент |
| `group_uuid` | TEXT UNIQUE | UUID для P2P идентификации |
| `name` | TEXT | Название |
| `description` | TEXT | Описание |
| `creator_peer_id` | TEXT | PeerID создателя |
| `avatar_hash` | TEXT | Хеш аватара (CAS) |
| `chat_type` | TEXT | `"group"` или `"channel"` |
| `max_invite_depth` | INTEGER | Глубина цепочки приглашений |
| `created_at` | INTEGER | Unix timestamp |
| `updated_at` | INTEGER | Unix timestamp |

### 2. `group_members`

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | INTEGER PK | Автоинкремент |
| `group_uuid` | TEXT FK → group_chats | |
| `peer_id` | TEXT | PeerID участника |
| `role` | TEXT | `"creator"`, `"admin"`, `"moderator"`, `"member"`, `"subscriber"` |
| `invited_by` | TEXT | PeerID пригласившего |
| `invite_depth` | INTEGER | Оставшаяся глубина инвайтов |
| `joined_at` | INTEGER | Unix timestamp |
| `left_at` | INTEGER | NULL если активен |

### 3. `group_messages`

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | INTEGER PK | Автоинкремент |
| `group_uuid` | TEXT FK → group_chats | |
| `message_uuid` | TEXT UNIQUE | UUID сообщения (дедупликация) |
| `from_peer_id` | TEXT | PeerID отправителя |
| `content` | TEXT | Текст или UUID элемента |
| `content_type` | TEXT | `"text"` или `"element"` |
| `metadata` | TEXT | JSON метаданные |
| `timestamp` | INTEGER | Unix nanoseconds |
| `lamport_clock` | INTEGER | Lamport timestamp для порядка |
| `signature` | BLOB | Ed25519 подпись отправителя |

### 4. `group_membership_proofs`

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | INTEGER PK | Автоинкремент |
| `group_uuid` | TEXT | |
| `peer_id` | TEXT | PeerID участника |
| `role` | TEXT | Роль |
| `granted_by` | TEXT | PeerID админа, выдавшего роль |
| `timestamp` | INTEGER | Unix timestamp |
| `admin_signature` | BLOB | Ed25519 подпись админа |

### 5. `group_invitations`

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | INTEGER PK | Автоинкремент |
| `group_uuid` | TEXT FK → group_chats | |
| `invite_token` | TEXT UNIQUE | Токен приглашения |
| `invited_by` | TEXT | PeerID пригласившего |
| `target_peer_id` | TEXT | NULL если ссылка ещё не использована |
| `depth` | INTEGER | Глубина цепочки |
| `status` | TEXT | `"pending"`, `"accepted"`, `"rejected"`, `"expired"` |
| `created_at` | INTEGER | Unix timestamp |

### 6. `group_blocks`

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | INTEGER PK | Автоинкремент |
| `group_uuid` | TEXT | |
| `peer_id` | TEXT | Заблокированный |
| `blocked_by` | TEXT | Кто заблокировал |
| `reason` | TEXT | Причина |
| `created_at` | INTEGER | Unix timestamp |

---

## 🔌 P2P протоколы

### `/projectt/groupchat/1.0.0` — сообщения (PubSub)

- Каждый групповой чат = отдельный GossipSub topic: `groupchat-{group_uuid}`
- Сообщения публикуются один раз, распространяются через mesh
- Структура:
  ```go
  type GroupMessage struct {
      MessageUUID string
      GroupUUID   string
      FromPeerID  string
      Content     string
      ContentType string // "text" | "element"
      Metadata    string
      Timestamp   int64
      LamportClock uint64
      Signature   []byte
  }
  ```

### `/projectt/groupchat-invite/1.0.0` — приглашения (point-to-point)

- Отправка инвайта конкретному пиру через stream
- Запрос:
  ```go
  type InviteRequest struct {
      GroupUUID   string
      GroupName   string
      ChatType    string
      InviteToken string
      InviterPeerID string
      Depth       int
      InviterSig  []byte
  }
  ```
- Ответ: `Accept` / `Reject`

### `/projectt/groupchat-sync/1.0.0` — синхронизация истории (point-to-point)

- Запрос у любого онлайн-участника:
  ```go
  type SyncRequest struct {
      GroupUUID          string
      LastKnownMessageUUID string
      LastLamportClock   uint64
      Count              int
  }
  ```
- Ответ: поток сообщений после `LastKnownMessageUUID`
- Любой участник может ответить — не только админ

### `/projectt/groupchat-admin/1.0.0` — управление (point-to-point)

- Только для admin/moderator
- Операции: `KickMember`, `ChangeRole`, `UpdateGroupMeta`, `BanMember`
- Каждое действие создаёт новый `GroupMembershipProof` с подписью админа

### `/projectt/profile-request/1.0.0` — запрос профиля (broadcast)

- Запрос профиля оффлайн пользователя у подключенных пиров:
  ```go
  type ProfileRequest struct {
      TargetPeerID string
      RequesterID  string
  }
  ```
- Ответ:
  ```go
  type ProfileResponse struct {
      Profile    models.Profile
      HasProfile bool
      Signature  []byte // подпись пира, отдающего профиль
  }
  ```
- Ответ добровольный, пир может проигнорировать

### `/projectt/tag-search/1.0.0` — поиск элементов по тегу (broadcast)

- Расширение ItemSync:
  ```go
  type TagSearchRequest struct {
      Tags       []string
      MatchMode  string // "any" | "all"
      MaxResults int
      RequesterID string
  }
  ```
- Ответ: `[]ItemSummary` (только метаданные)
- Пир может отказаться ответить или вернуть пустой список

---

## 🏗️ Архитектура

### Новые сервисы

#### `PubSubManager`
- Управление GossipSub topics
- Методы: `JoinGroup(groupUUID)`, `LeaveGroup(groupUUID)`, `PublishMessage(groupUUID, msg)`, `Subscribe(groupUUID, handler)`
- Подписка при входе, отписка при выходе/бане

#### `GroupChatService`
- Бизнес-логика групповых чатов
- Методы: `CreateGroup()`, `InviteMember()`, `SendMessage()`, `SyncHistory()`, `KickMember()`, `BanMember()`
- Event subscription для UI

#### `MembershipVerifier`
- Верификация `GroupMembershipProof`
- Проверка подписи админа перед принятием действий от участника
- Кеш валидных proofs

### Изменения в существующих компонентах

| Компонент | Изменение |
|-----------|-----------|
| `core/network.go` | Инжект PubSubManager |
| `services/chat_service.go` | Различение типов: direct / group / channel |
| `controllers/chat_controller.go` | Поддержка групп в UI-логике |
| `storage/database/` | +4 таблицы, миграции, новые queries |
| `models/` | +GroupChat, GroupMember, GroupMessage, GroupInvitation, GroupMembershipProof |
| UI sidebar | Разделение: Direct Messages / Groups / Channels |

---

## 🔐 Безопасность и права

### Подписанный реестр участников

Проблема: пользователь меняет себе `role` в локальной SQLite на `admin`.

Решение: `GroupMembershipProof` с подписью админа:
```go
type GroupMembershipProof struct {
    GroupUUID   string
    PeerID      string
    Role        string
    GrantedBy   string
    Timestamp   int64
    AdminSig    []byte
}
```

- Каждое действие участника проверяется: есть ли валидный proof с подписью админа
- Нет подписи = действие игнорируется
- Смена роли = новый proof, старый инвалидируется

### Каналы

- Только creator/admin может публиковать сообщения
- Subscribers только читают
- Сообщения от subscribers не проходят верификацию

### Чёрные списки

- Админ блокирует → участник удаляется из `group_members`, добавляется в `group_blocks`
- Пользователь блокирует группу → отписка от topic, сообщения скрыты
- При инвайте → проверка `group_blocks`, автоотклонение

---

## 📡 Доставка сообщений

### GossipSub mesh

```
Отправитель → 5-6 mesh-соседей → каждый → ещё 5-6 соседей → ...
```

- Отправитель публикует **один раз** в topic
- Сообщение форвардится через mesh, не点对点 каждому
- За 2-3 хопа доходит до всех
- Дедупликация по `message_uuid`

### Store-and-forward

- Любой участник хранит историю в `group_messages`
- Новый участник запрашивает sync у **любого онлайн-участника**
- Админ не нужен для доставки или синхронизации

### Рассинхрон допустим

| Данные | Точность |
|--------|----------|
| Сообщения | Высокая (подписаны, дедупликация) |
| Список участников | Допустим рассинхрон (обновляется при sync) |
| Роли | Высокая (только с подписью админа) |
| Порядок | Относительная (Lamport clocks) |
| Счётчик непрочитанных | Локальный (не синхронизируется) |

---

## 🛡️ Анти-спам

| Механика | Описание |
|----------|----------|
| Rate limit | N сообщений в минуту на группу |
| Ban | Исключение + блок по PeerID + блок по цепочке инвайта |
| Одноразовые ссылки | Одна ссылка = один вход, с подписью и глубиной |
| Ограничение инвайтов | `max_invite_depth` задаётся создателем |
| Не начислять за сообщения | Токены только за relay, хранение, sync (на будущее) |

---

## 📊 Модели по размеру

### Маленький чат (< 10 участников)
- Full mesh, все соединены со всеми
- Каждый хранит полную историю
- Распределённая нагрузка

### Группа (10-50 участников)
- Mesh + хранители (добровольцы)
- Хранители хранят полную историю, отдают при sync
- Обычные участники хранят последние N сообщений
- Lazy sync: только дельта при подключении

### Канал (50+ подписчиков)
- Broadcast-модель
- Админ публикует → PubSub → все
- Подписчики не пишут
- История у админа + хранителей
- Sync у любого онлайн-подписчика

---

## 🔗 Цепочка приглашений

```
Создатель → A (depth=3) → B (depth=2) → C (depth=1) → D (depth=0, больше не приглашает)
```

- `invite_token` содержит `depth`
- Когда depth = 0 → инвайты больше не выдаются
- Каждый инвайт подписан выдавшим → отслеживаемость цепочки
- Бан по цепочке: если D спамит → бан до A включительно

---

## 📝 Чек-лист реализации

### Фаза 1: База (2-3 недели)
- [ ] Миграции БД: 4 новые таблицы
- [ ] Модели: GroupChat, GroupMember, GroupMessage, GroupInvitation, GroupMembershipProof
- [ ] Queries: CRUD для всех таблиц
- [ ] PubSubManager: join/leave/publish/subscribe

### Фаза 2: Протоколы (2-3 недели)
- [ ] `/projectt/groupchat/1.0.0` — PubSub сообщения
- [ ] `/projectt/groupchat-invite/1.0.0` — приглашения
- [ ] `/projectt/groupchat-sync/1.0.0` — синхронизация
- [ ] `/projectt/groupchat-admin/1.0.0` — управление
- [ ] MembershipVerifier

### Фаза 3: Бизнес-логика (1-2 недели)
- [ ] GroupChatService
- [ ] Создание группы, инвайты, цепочки
- [ ] Отправка/приём сообщений, верификация
- [ ] Синхронизация истории
- [ ] Бан/кик, чёрные списки

### Фаза 4: UI (1-2 недели)
- [ ] Список групп/каналов в sidebar
- [ ] Создание группы, генерация инвайт-ссылок
- [ ] Групповой чат: имена отправителей, аватары
- [ ] Правая панель: информация о группе, участники
- [ ] Управление участниками (для админа)
- [ ] Разделение типов: direct / group / channel

### Фаза 5: Дополнительные протоколы (1-2 недели)
- [ ] `/projectt/profile-request/1.0.0`
- [ ] `/projectt/tag-search/1.0.0`
- [ ] Rate limiting
- [ ] Lazy sync оптимизация

---

**Общее время:** 7-12 недель
