# 🔄 План миграции на новую схему БД

**Цель:** Перейти со старых таблиц (`profile`, `p2p_profile`, `files`, `items_backup`) на новые (`profiles`, `profile_keys`, `item_files`).

**Статус:** ✅ **ВЫПОЛНЕНА** (12 марта 2026)

---

## 📊 Текущее состояние

### Старые таблицы (подлежат удалению)
| Таблица | Статус | Используется в |
|---------|--------|----------------|
| `profile` | ⚠️ Устарела | Резервная копия |
| `p2p_profile` | ⚠️ Устарела | Резервная копия |
| `files` | ❌ Устарела | Не используется |
| `items_backup` | ❌ Устарела | Не используется |

### Новые таблицы (целевые)
| Таблица | Статус | Назначение |
|---------|--------|------------|
| `profiles` | ✅ Активно | Профили пользователей (local/remote) |
| `profile_keys` | ✅ Активно | Криптографические ключи профилей |
| `item_files` | ✅ Активно | Файлы элементов |

---

## 🎯 Целевая схема

### Таблица `profiles` (универсальная)
```sql
CREATE TABLE profiles (
    id              INTEGER PRIMARY KEY,
    owner_type      TEXT NOT NULL CHECK (owner_type IN ('local', 'remote')),
    peer_id         TEXT UNIQUE NOT NULL,
    username        TEXT NOT NULL,
    status          TEXT DEFAULT 'online',
    avatar_path     TEXT,
    background_path TEXT DEFAULT '',
    content_char    TEXT,
    demo_elements   TEXT,
    cached_at       DATETIME,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Таблица `profile_keys` (ключи)
```sql
CREATE TABLE profile_keys (
    profile_id       INTEGER PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
    private_key      BLOB,
    public_key       BLOB NOT NULL,
    signature        BLOB,
    is_key_encrypted BOOLEAN DEFAULT 0
);
```

### Связь
- `profiles` ↔ `profile_keys`: 1:1
- Данные мигрированы из `profile` + `p2p_profile`

---

## ✅ Выполненные этапы

### Этап 1: Обновление queries/profile.go
**Задача:** Заменить вызовы старых функций на новые

**Выполнено:**
- ✅ `queries.GetProfile()` → `queries.GetLocalProfile()`
- ✅ `queries.CreateProfile()` → `queries.CreateRemoteProfile()`
- ✅ `queries.UpdateProfile()` → `queries.UpdateLocalProfile()`
- ✅ `queries.UpdateProfileField()` → `queries.UpdateLocalProfileField()`

**Обновлённые файлы:**
- ✅ `internal/ui/workspace/workspace.go`
- ✅ `internal/ui/workspace/profile/profile.go`
- ✅ `internal/ui/workspace/profile/methods.go`
- ✅ `internal/ui/workspace/profile/handlers.go`
- ✅ `internal/services/background/service.go`
- ✅ `internal/ui/workspace/profile/profile_test.go`

---

### Этап 2: Обновление queries/peers.go
**Задача:** Заменить `GetP2PProfile()` на загрузку из `profiles` + `profile_keys`

**Выполнено:**
- ✅ Обновлён `GetP2PProfile()` — загрузка из двух таблиц
- ✅ Обновлён `CreateP2PProfile()` — создание в двух таблицах
- ✅ Обновлён `UpdateP2PProfile()` — обновление в двух таблицах
- ✅ Обновлён `UpdateP2PProfileField()` — маршрутизация полей

**Обновлённые файлы:**
- ✅ `internal/storage/database/queries/peers.go`
- ✅ `internal/storage/database/queries/profile_keys.go` (создан)

---

### Этап 3: Миграция данных
**Скрипт:** `scripts/migrate_to_profiles.go`

**Выполнено:** ✅
1. ✅ Перенесён `profile` → `profiles` (owner_type='local')
2. ✅ Перенесены ключи из `p2p_profile` → `profile_keys`
3. ✅ Связь `profile_keys.profile_id` с `profiles.id`
4. ✅ Проверка целостности данных

**Результат:**
```
profiles (local): 1 запись
profile_keys: 1 запись
✅ Все локальные профили имеют ключи
```

---

### Этап 4: Удаление старых таблиц
**Скрипт:** `scripts/cleanup_old_tables.go` (обновлён)

**Готов к удалению:**
- ✅ `profile`
- ✅ `p2p_profile`
- ✅ `files`
- ✅ `items_backup`

---

### Этап 5: Обновление моделей
**Статус:**
- ✅ `internal/storage/database/models/peer.go` — оставлен для обратной совместимости
- ✅ `internal/storage/database/models/profile_key.go` — используется активно

---

## 📁 Созданные файлы

1. ✅ `internal/storage/database/queries/profile_keys.go` — CRUD для ключей
2. ✅ `scripts/migrate_to_profiles.go` — скрипт миграции данных
3. ✅ `scripts/cleanup_old_tables.go` — скрипт удаления старых таблиц
4. ✅ `scripts/check_and_view_db.go` — утилита проверки и просмотра БД

---

## 📋 Чек-лист

- [x] Обновить `queries/profile.go` — заменить вызовы на `GetLocalProfile()`
- [x] Обновить `queries/peers.go` — загрузка из `profiles` + `profile_keys`
- [x] Создать `queries/profile_keys.go` — CRUD для ключей
- [x] Обновить `services/background/service.go`
- [x] Обновить `ui/workspace/profile/*.go`
- [x] Обновить `ui/workspace/workspace.go`
- [x] Создать скрипт миграции данных
- [x] Протестировать на тестовой БД
- [x] Выполнить миграцию
- [ ] Удалить старые таблицы (ожидает подтверждения)
- [x] Запустить `go mod tidy`
- [x] Проверить компиляцию
- [ ] Протестировать приложение

---

## 🚀 Использование

### 1. Проверка текущего состояния
```bash
go run scripts/check_and_view_db.go
```

### 2. Миграция данных (если ещё не выполнена)
```bash
go run scripts/migrate_to_profiles.go
```

### 3. Удаление старых таблиц
```bash
go run scripts/cleanup_old_tables.go
```

### 4. Оптимизация БД (рекомендуется)
```bash
go run scripts/vacuum_db.go
```

---

## ⚠️ Риски

| Риск | Влияние | Митигация |
|------|---------|-----------|
| Потеря ключей | Критичное | ✅ Резервная копия БД перед миграцией |
| Некорректная связь profile_id | Среднее | ✅ Проверка целостности после миграции |
| Падение производительности | Низкое | ✅ Кэширование профиля при загрузке |

---

**Дата создания:** 12 марта 2026  
**Статус:** ✅ **ВЫПОЛНЕНА** (ожидает удаления старых таблиц)
