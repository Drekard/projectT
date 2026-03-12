# 🧹 Рефакторинг: Удаление обратной совместимости

**Дата:** 12 марта 2026  
**Статус:** ✅ **ВЫПОЛНЕН**

---

## 📋 Что было сделано

### 1. Удалены старые таблицы из миграций

**Файл:** `internal/storage/database/migrations.go`

**Удалено:**
- ❌ Создание таблицы `profile` (строки 136-189)
- ❌ Функция `createP2PTables()` (создавала `p2p_profile`)
- ❌ Функция `migrateExistingData()` (перенос данных из старых таблиц)
- ❌ Функция `clearOldBootstrapPeers()`

**Добавлено:**
- ✅ Создание `contacts`, `chat_messages`, `bootstrap_peers` в `createNewProfileTables()`

**Итог:**
- Новые установки БД создают **только новую схему**
- Старые таблицы (`profile`, `p2p_profile`) больше не создаются

---

### 2. Удалён файл обратной совместимости

**Файл:** `internal/storage/database/queries/profile.go` — **УДАЛЁН**

**Удалённые функции-обёртки:**
- ❌ `GetProfile()` → заменена на `GetLocalProfile()`
- ❌ `CreateProfile()` → заменена на `CreateRemoteProfile()`
- ❌ `UpdateProfile()` → заменена на `UpdateLocalProfile()`
- ❌ `UpdateProfileField()` → заменена на `UpdateLocalProfileField()`
- ❌ `GetProfileByUsername()` → заменена на прямую загрузку профиля
- ❌ `GetProfileByID()` → не использовалась
- ❌ `InitializeDefaultProfile()` → не использовалась

---

### 3. Обновлены зависимости

**Файл:** `internal/services/p2p/network/profile.go`

**Заменено:**
```go
// БЫЛО:
username, err := queries.GetProfileUsername()

// СТАЛО:
var username string
localProfile, err := queries.GetLocalProfile()
if err != nil {
    username = fmt.Sprintf("User_%s", peerID.String()[:8])
} else {
    username = localProfile.Username
}
```

---

## 🎯 Целевая схема БД

### Таблицы которые создаются ТЕПЕРЬ:

```
✅ items              — элементы (folder/element)
✅ tags               — теги
✅ item_tags          — связь элементов с тегами
✅ favorites          — избранное
✅ pinned_items       — закреплённые элементы
✅ profiles           — профили (local/remote)
✅ profile_keys       — криптографические ключи
✅ remote_items       — кэшированные чужие элементы
✅ item_files         — файлы элементов
✅ contacts           — контакты (адресная книга)
✅ chat_messages      — история сообщений
✅ bootstrap_peers    — узлы для подключения
```

### Таблицы которые БОЛЬШЕ не создаются:

```
❌ profile            — устарела (заменена на profiles)
❌ p2p_profile        — устарела (заменена на profiles + profile_keys)
❌ files              — устарела (заменена на item_files)
```

---

## 📊 Влияние на существующие БД

### Если БД уже существует:

1. **Старые таблицы остаются** — они не удаляются автоматически
2. **Миграции пропускаются** — новые таблицы создаются через `CREATE TABLE IF NOT EXISTS`
3. **Данные сохраняются** — старые данные не затрагиваются

### Для очистки старой БД:

```bash
# 1. Выполните миграцию данных (если ещё не сделали)
go run scripts/migrate_to_profiles.go

# 2. Удалите старые таблицы
go run scripts/cleanup_old_tables.go

# 3. Оптимизируйте БД
go run scripts/vacuum_db.go
```

---

## ✅ Проверка компиляции

```bash
go build ./internal/... ./cmd/...
# ✅ Успешно
```

---

## 📁 Изменённые файлы

| Файл | Изменения |
|------|-----------|
| `internal/storage/database/migrations.go` | Удалено 200+ строк |
| `internal/storage/database/queries/profile.go` | **УДАЛЁН** (80 строк) |
| `internal/services/p2p/network/profile.go` | Обновлена загрузка username |

---

## 🚀 Преимущества

1. **Чище код** — нет устаревших обёрток
2. **Проще миграции** — не нужно переносить данные
3. **Меньше таблиц** — проще поддержка
4. **Ясная схема** — `profiles` + `profile_keys` вместо `profile` + `p2p_profile`

---

## ⚠️ Риски

| Риск | Вероятность | Влияние |
|------|-------------|---------|
| Падение старых БД | Низкая | Критичное |
| Потеря данных | Низкая | Критичное |
| Ошибки компиляции | ✅ Нет | — |

**Митигация:**
- ✅ Резервная копия БД перед обновлением
- ✅ Тестирование на dev-окружении
- ✅ Скрипты миграции доступны

---

## 📝 Следующие шаги

1. ✅ Протестировать приложение
2. ✅ Проверить запуск на чистой БД
3. ⏳ Обновить документацию проекта
4. ⏳ Сделать релиз

---

**Итого:** Приложение теперь работает **только с новой схемой** (`profiles` + `profile_keys`). 
Старые таблицы (`profile`, `p2p_profile`) больше не создаются и не используются.
