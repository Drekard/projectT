# Миграция таблицы contacts

## Обзор изменений

Таблица `contacts` была нормализована для устранения дублирования данных с таблицей `profiles`.

### Что было удалено из `contacts`:
- `username` → теперь хранится в `profiles.username`
- `public_key` → теперь хранится в `profile_keys.public_key`
- `status` → теперь хранится в `profiles.title`
- `avatar_path` → теперь хранится в `profiles.avatar_path`

### Что осталось в `contacts`:
- `peer_id` — FOREIGN KEY → `profiles.peer_id`
- `multiaddr` — адрес для P2P подключения
- `notes` — заметки пользователя
- `is_blocked` — локальные настройки блокировки
- `is_favorite` — флаг "в избранном"
- `last_seen` — локальная история активности
- `added_at` — когда добавлен в контакты
- `updated_at`

## Как выполнить миграцию

### Вариант 1: Автоматическая миграция (рекомендуется)

```bash
cd c:\Users\egors\Desktop\projectT
go run scripts/migrate_contacts_table.go
```

Скрипт:
1. Проверит наличие старых колонок
2. Создаст профили для всех контактов (если их нет)
3. Перенесёт данные в новую таблицу
4. Удалит старую таблицу
5. Переименует новую таблицу в `contacts`

### Вариант 2: Ручная миграция через SQL

```sql
-- 1. Создаём профили для контактов
INSERT OR IGNORE INTO profiles (owner_type, peer_id, username, title, avatar_path, created_at, updated_at)
SELECT 
    'remote' as owner_type,
    peer_id,
    COALESCE(username, substr(peer_id, 1, 8)) as username,
    COALESCE(status, '') as title,
    COALESCE(avatar_path, '') as avatar_path,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM contacts
WHERE peer_id NOT IN (SELECT peer_id FROM profiles);

-- 2. Создаём новую таблицу
CREATE TABLE contacts_new (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    peer_id     TEXT UNIQUE NOT NULL REFERENCES profiles(peer_id),
    multiaddr   TEXT,
    notes       TEXT,
    is_blocked  BOOLEAN DEFAULT 0,
    is_favorite BOOLEAN DEFAULT 1,
    last_seen   DATETIME,
    added_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 3. Копируем данные
INSERT INTO contacts_new (id, peer_id, multiaddr, notes, is_blocked, last_seen, added_at, updated_at)
SELECT id, peer_id, multiaddr, notes, is_blocked, last_seen, added_at, updated_at
FROM contacts;

-- 4. Меняем таблицу местами
DROP TABLE contacts;
ALTER TABLE contacts_new RENAME TO contacts;

-- 5. Создаём индексы
CREATE INDEX IF NOT EXISTS idx_contacts_peer_id ON contacts(peer_id);
```

## Проверка результата

```sql
-- Проверяем схему
PRAGMA table_info(contacts);

-- Проверяем данные
SELECT c.id, c.peer_id, p.username, p.title, c.multiaddr, c.notes
FROM contacts c
LEFT JOIN profiles p ON c.peer_id = p.peer_id;
```

## Изменения в коде

### Обновлённые файлы:
- `internal/storage/database/models/contact.go` — новая модель
- `internal/storage/database/migrations.go` — новая схема в миграциях
- `internal/storage/database/queries/contacts.go` — запросы с JOIN на profiles
- `internal/storage/database/queries/profiles.go` — функция `EnsureProfileForContact`
- `internal/services/contacts/contacts_service.go` — создание профиля при добавлении контакта
- `internal/services/p2p/network/events.go` — обновление `last_seen` вместо `status`
- `internal/services/p2p/connections.go` — обновление `last_seen` вместо `status`
- `internal/services/p2p/discovery.go` — обновление `last_seen` вместо `status`
- `internal/ui/workspace/chats/right_panel.go` — использование `contact.Title` вместо `contact.Status`

### Статус онлайн/офлайн

Теперь определяется **динамически** через P2P подключение, а не хранится в БД:

```go
// В contacts_service.go
func (s *ContactService) enrichContactWithStatus(contact *models.Contact) *ContactWithStatus {
    result := &ContactWithStatus{
        Contact:  contact,
        IsOnline: contact.IsOnline, // Заполняется из P2P
    }
    
    if s.p2pNetwork != nil {
        status := s.p2pNetwork.GetConnectionStatus(peerID)
        result.IsOnline = status == p2p.StatusConnected
    }
    
    return result
}
```

## Откат миграции

Если нужно вернуть старую схему (не рекомендуется):

```sql
-- Создаём старую таблицу
CREATE TABLE contacts_old (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    peer_id     TEXT UNIQUE NOT NULL,
    username    TEXT NOT NULL,
    public_key  BLOB,
    multiaddr   TEXT,
    status      TEXT DEFAULT 'offline',
    last_seen   DATETIME,
    notes       TEXT,
    is_blocked  BOOLEAN DEFAULT 0,
    avatar_path TEXT,
    added_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Копируем данные обратно
INSERT INTO contacts_old (id, peer_id, username, public_key, multiaddr, status, last_seen, notes, is_blocked, avatar_path, added_at, updated_at)
SELECT 
    c.id, c.peer_id, 
    p.username, p.title as status, -- Используем title как status
    c.multiaddr, p.title, c.last_seen, c.notes, c.is_blocked, p.avatar_path,
    c.added_at, c.updated_at
FROM contacts c
LEFT JOIN profiles p ON c.peer_id = p.peer_id;

-- Меняем таблицу
DROP TABLE contacts;
ALTER TABLE contacts_old RENAME TO contacts;
```
