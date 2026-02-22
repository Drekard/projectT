# 📡 План реализации P2P функциональности

**ProjectT** — локальный Pinterest-подобный менеджер файлов  
**Цель** — добавить P2P связь и чаты пользователей без центрального сервера

---

## 🎯 Желаемый конечный результат

После реализации пользователь сможет:

1. **Уникальная идентификация**
   - Каждый узел имеет уникальный `PeerID` и криптографические ключи
   - Может экспортировать свой адрес для шаринга с другими пользователями
   - Импортировать адреса контактов для подключения

2. **Обнаружение узлов**
   - Автоматическое нахождение пиров в локальной сети (mDNS)
   - Подключение к глобальной сети через DHT
   - Сохранение bootstrap-узлов в БД для быстрого старта

3. **NAT Traversal**
   - Автоматическое преодоление NAT/firewall
   - Поддержка соединения при обрыве
   - Работа за большинством домашних роутеров

4. **Чат 1-на-1**
   - Отправка текстовых сообщений
   - Отправка файлов/изображений
   - Шифрование трафика (Noise/TLS)
   - История переписки в БД

5. **Управление контактами**
   - Добавление/удаление контактов
   - Сохранение статуса онлайн/оффлайн (но как реализовать без пингов каждую секунду?)
   - Поиск по контактам

---

## 📊 Новые таблицы базы данных

### 1. `p2p_profile` — локальный профиль P2P

```sql
CREATE TABLE p2p_profile (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    peer_id         TEXT UNIQUE NOT NULL,
    private_key     BLOB NOT NULL,
    public_key      BLOB NOT NULL,
    username        TEXT NOT NULL,
    status          TEXT DEFAULT 'online',
    listen_addrs    TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**Назначение:** Хранение идентичности узла. Всегда одна запись (id=1).

---

### 2. `contacts` — адресная книга

```sql
CREATE TABLE contacts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    peer_id         TEXT UNIQUE NOT NULL,
    username        TEXT NOT NULL,
    public_key      BLOB,
    multiaddr       TEXT,
    status          TEXT DEFAULT 'offline',
    last_seen       DATETIME,
    notes           TEXT,
    is_blocked      BOOLEAN DEFAULT 0,
    added_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**Назначение:** Сохранённые пользователи для связи.

---

### 3. `chat_messages` — история сообщений

```sql
CREATE TABLE chat_messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    contact_id      INTEGER NOT NULL,
    from_peer_id    TEXT NOT NULL,
    content         TEXT NOT NULL,
    content_type    TEXT DEFAULT 'text',
    metadata        TEXT,
    is_read         BOOLEAN DEFAULT 0,
    sent_at         DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (contact_id) REFERENCES contacts (id) ON DELETE CASCADE
);
```

**Назначение:** Постоянное хранение переписки.

---

### 4. `bootstrap_peers` — узлы для входа в сеть

```sql
CREATE TABLE bootstrap_peers (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    multiaddr       TEXT UNIQUE NOT NULL,
    peer_id         TEXT,
    is_active       BOOLEAN DEFAULT 1,
    last_connected  DATETIME,
    added_at        DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**Назначение:** Предопределённые узлы для начального подключения.

---

## 🏗 Новая структура проекта

```
projectT/
├── cmd/
│   └── main.go                          # ⚠️ ИЗМЕНИТЬ
│
├── internal/
│   ├── app/
│   │   └── app.go                       # ⚠️ ИЗМЕНИТЬ
│   │
│   ├── ui/
│   │   ├── ... (существующие файлы)
│   │   │
│   │   └── workspace/
│   │       └── chats/                   # 🆕 СОЗДАТЬ
│   │           ├── bootstrap_panel.go   # Панель управления bootstrap-узлами
│   │           ├── contacts_window.go   # Окно со списком контактов
│   │           ├── chat_window.go       # Окно чата
│   │           ├── peer_info_panel.go   # Информация о текущем пире
│   │           └── widgets/             # 🆕 СОЗДАТЬ
│   │               ├── contact_item.go      # Виджет строки контакта
│   │               ├── message_bubble.go    # Виджет сообщения в чате
│   │               └── status_indicator.go  # Индикатор статуса
│   │
│   ├── services/
│   │   ├── ... (существующие файлы)
│   │   │
│   │   ├── p2p/                         # 🆕 СОЗДАТЬ
│   │   │   ├── network.go               # Ядро libp2p: создание хоста, ключи, PeerID 
│   │   │   ├── discovery.go             # Обнаружение: mDNS, DHT, bootstrap
│   │   │   ├── chat.go                  # Чат: отправка/получение сообщений, потоки
│   │   │   ├── connections.go           # Мониторинг соединений: переподключение, keepalive
│   │   │   └── config.go                # Конфигурация P2P: порты, протоколы, таймауты
│   │   │
│   │   └── contacts/                    # 🆕 СОЗДАТЬ
│   │       └── contacts_service.go      # Бизнес-логика: добавление, поиск, статусы, управление контактами
│   │
│   └── storage/
│       ├── database/
│       │   ├── connection.go            # ✅ Без изменений
│       │   ├── migrations.go            # ⚠️ ИЗМЕНИТЬ
│       │   │
│       │   ├── models/
│       │   │   ├── ... (существующие)
│       │   │   ├── peer.go              # 🆕 СОЗДАТЬ - идентичность узла
│       │   │   ├── contact.go           # 🆕 СОЗДАТЬ - контакт в адресной книге
│       │   │   ├── message.go           # 🆕 СОЗДАТЬ - сообщение чата
│       │   │   └── bootstrap_peer.go    # 🆕 СОЗДАТЬ - узел для подключения
│       │   │
│       │   └── queries/
│       │       ├── ... (существующие)
│       │       ├── peers.go             # 🆕 СОЗДАТЬ - CRUD для `P2PProfile`
│       │       ├── contacts.go          # 🆕 СОЗДАТЬ - CRUD для `Contact`
│       │       ├── messages.go          # 🆕 СОЗДАТЬ - CRUD для `ChatMessage`
│       │       └── bootstrap_peers.go   # 🆕 СОЗДАТЬ - CRUD для `BootstrapPeer` 
│       │
│       └── files/
│           └── ... (существующее)       # ✅ Без изменений
│
├── go.mod                               # ⚠️ ИЗМЕНИТЬ
├── go.sum                               # ⚠️ ИЗМЕНИТЬ (автоматически)
│
└── P2P_IMPLEMENTATION_PLAN.md           # 🆕 ЭТОТ ФАЙЛ
```

---

## 📦 Новые библиотеки (go.mod)

```go
require (
    // === P2P ядро ===
    github.com/libp2p/go-libp2p v0.32.0
    github.com/libp2p/go-libp2p-kad-dht v0.25.0
    github.com/libp2p/go-libp2p-pubsub v0.10.0
    github.com/libp2p/go-libp2p-discovery v0.5.0
    github.com/libp2p/go-libp2p-mdns v0.6.0
    
    // === Адреса и мультиформаты ===
    github.com/multiformats/go-multiaddr v0.12.0
    github.com/multiformats/go-multibase v0.2.0
    github.com/multiformats/go-multihash v0.2.3
    
    // === Криптография ===
    github.com/libp2p/go-libp2p/core v0.30.0
    github.com/libp2p/go-noise v0.1.0
    
    // === Протоколы ===
    google.golang.org/protobuf v1.31.0  // для protobuf сообщений
    
    // === Утилиты ===
    github.com/hashicorp/go-multierror v1.1.1
    golang.org/x/crypto v0.17.0
)
```

---

## ⚠️ Существующие файлы для изменения

### 1. `cmd/main.go`

**Что изменить:**
- Инициализация P2P сети при старте
- Graceful shutdown P2P при выходе
- Передача P2P сети в приложение

**Примерная логика:**
```go
func main() {
    database.InitDB()
    database.RunMigrations()
    filesystem.EnsureStorageStructure()
    
    // Инициализация P2P
    p2pNetwork := p2p.NewP2PNetwork()
    if err := p2pNetwork.Start(); err != nil {
        log.Printf("Warning: P2P network failed to start: %v", err)
    }
    defer p2pNetwork.Stop()
    
    myApp := app.NewApp(p2pNetwork)
    myApp.Run()
}
```

---

### 2. `internal/app/app.go`

**Что изменить:**
- Принимать `*p2p.P2PNetwork` в конструкторе
- Сохранять ссылку на P2P сеть в структуре `App`
- Передать P2P сеть в UI для доступа из интерфейса

---

### 3. `internal/storage/database/migrations.go`

**Что изменить:**
- Добавить функцию `createP2PTables()`
- Вызвать её в `RunMigrations()`
- Добавить 4 новые таблицы (см. выше)

**Примерная структура:**
```go
func RunMigrations() {
    // ... существующие миграции ...
    
    createP2PTables()
    seedBootstrapPeers()
}

func createP2PTables() {
    // CREATE TABLE p2p_profile
    // CREATE TABLE contacts
    // CREATE TABLE chat_messages
    // CREATE TABLE bootstrap_peers
}

func seedBootstrapPeers() {
    // Добавить предопределённые bootstrap-узлы
}
```

---

### 4. `go.mod`

**Что изменить:**
- Добавить новые зависимости (см. раздел "Новые библиотеки")
- Запустить `go mod tidy`

---

## 🔄 Логика взаимодействия компонентов

```
┌─────────────────────────────────────────────────────────────┐
│                        UI (Fyne)                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Contacts    │  │    Chat     │  │   Peer Info         │  │
│  │ Window      │  │   Window    │  │   Panel             │  │
│  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────┘  │
└─────────┼────────────────┼─────────────────────┼────────────┘
          │                │                     │
          ▼                ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                     Services Layer                          │
│  ┌──────────────────┐  ┌─────────────────────────────────┐  │
│  │ ContactsService  │  │         P2P Services            │  │
│  │ (contacts/)      │  │  ┌──────────┐  ┌─────────────┐  │  │
│  │                  │  │  │ Network  │  │  Discovery  │  │  │
│  │ + AddContact()   │  │  │  Chat    │  │  Connection │  │  │
│  │ + GetContacts()  │  │  └──────────┘  └─────────────┘  │  │
│  │ + DeleteContact()│  └─────────────────────────────────┘  │
│  └──────────────────┘                                       │
└──────────┬────────────────────────┬─────────────────────────┘
           │                        │
           ▼                        ▼
┌─────────────────────┐  ┌─────────────────────────────────────┐
│   SQLite Database   │  │      libp2p Network Stack           │
│  ┌───────────────┐  │  │  ┌───────────────────────────────┐  │
│  │ p2p_profile   │  │  │  │ DHT (kad-dht)                 │  │
│  │ contacts      │  │  │  │ mDNS (локальная сеть)         │  │
│  │ chat_messages │  │  │  │ NAT Traversal (hole punch)    │  │
│  │ bootstrap     │  │  │  │ Encrypted Streams (Noise)     │  │
│  └───────────────┘  │  │  └───────────────────────────────┘  │
└─────────────────────┘  └─────────────────────────────────────┘
```

---

## 📋 Чек-лист реализации

### Этап 1: Подготовка (1-2 недели)
- [ ] Обновить `go.mod` новыми зависимостями
- [ ] Добавить миграции БД в `migrations.go`
- [ ] Создать модели (`models/*.go`)
- [ ] Создать запросы к БД (`queries/*.go`)
- [ ] Тесты на CRUD операции

### Этап 2: P2P ядро (2-3 недели)
- [ ] `internal/services/p2p/network.go` — создание хоста
- [ ] Генерация/загрузка ключей
- [ ] Получение PeerID
- [ ] Экспорт/импорт адресов
- [ ] `internal/services/p2p/config.go` — конфигурация

### Этап 3: Обнаружение (2-3 недели)
- [ ] `internal/services/p2p/discovery.go`
- [ ] mDNS для локальной сети
- [ ] DHT для глобальной сети
- [ ] Bootstrap-узлы (чтение/запись в БД)
- [ ] Тесты на обнаружение пиров

### Этап 4: NAT Traversal (2-3 недели)
- [ ] `internal/services/p2p/connections.go`
- [ ] NAT Port Mapping (UPnP/NAT-PMP)
- [ ] Hole punching
- [ ] AutoRelay
- [ ] Мониторинг соединений
- [ ] Автоматическое переподключение

### Этап 5: Чат (2-3 недели)
- [ ] `internal/services/p2p/chat.go`
- [ ] Протокол `/projectt/chat/1.0.0`
- [ ] Отправка сообщений
- [ ] Получение сообщений
- [ ] Сохранение в БД
- [ ] Шифрование (Noise/TLS)
- [ ] Очередь сообщений (если оффлайн)

### Этап 6: Контакты (1-2 недели)
- [ ] `internal/services/contacts/contacts_service.go`
- [ ] Добавление контакта по PeerID
- [ ] Поиск контактов
- [ ] Статусы (онлайн/оффлайн)
- [ ] Блокировка
- [ ] Тесты

### Этап 7: UI (3-4 недели)
- [ ] `internal/ui/workspace/chats/contacts_window.go`
- [ ] `internal/ui/workspace/chats/chat_window.go`
- [ ] `internal/ui/workspace/chats/bootstrap_panel.go`
- [ ] `internal/ui/workspace/chats/peer_info_panel.go`
- [ ] Виджеты (`widgets/`)
- [ ] Интеграция с основным окном

### Этап 8: Интеграция и тесты (2-3 недели)
- [ ] Обновить `cmd/main.go`
- [ ] Обновить `internal/app/app.go`
- [ ] Graceful shutdown
- [ ] End-to-end тесты (2 узла)
- [ ] Тесты на разрыв соединения
- [ ] Документация

---

## 🎯 Критерии приёмки

### Функциональные:
- [ ] Узел генерирует уникальный PeerID при первом запуске
- [ ] PeerID сохраняется в БД и восстанавливается
- [ ] Можно экспортировать адрес в формате `peerid@multiaddr`
- [ ] Можно импортировать адрес и добавить контакт
- [ ] Контакты отображаются в UI со статусом
- [ ] Можно отправить сообщение контакту
- [ ] Сообщения сохраняются в истории
- [ ] Автоматическое обнаружение в локальной сети
- [ ] Подключение через DHT (глобальная сеть)
- [ ] Работа за NAT (более 80% успешных подключений)

### Нефункциональные:
- [ ] Время запуска P2P < 5 секунд
- [ ] Потребление памяти < 100MB в простое
- [ ] Сообщения доставляются < 2 секунд (в локальной сети)
- [ ] Корректная очистка ресурсов при выходе
- [ ] Нет утечек горутин

---

## ⚠️ Риски и проблемы

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| NAT traversal не работает | Высокая | Критичное | Использовать relay-узлы fallback |
| DHT слишком медленный | Средняя | Среднее | Кэшировать пиров, использовать mDNS |
| Утечки памяти в libp2p | Средняя | Среднее | Профилирование, тесты нагрузки |
| Сложность отладки | Высокая | Среднее | Логирование, тесты с моками |
| Конфликты портов | Низкая | Низкое | Динамический выбор портов |

---

## 📚 Полезные ресурсы

### Документация:
- [libp2p Docs](https://docs.libp2p.io/)
- [go-libp2p GitHub](https://github.com/libp2p/go-libp2p)
- [libp2p Examples](https://github.com/libp2p/go-libp2p-examples)

### Статьи:
- [How NAT Traversal Works](https://docs.libp2p.io/concepts/nat/)
- [DHT in libp2p](https://docs.libp2p.io/concepts/dht/)
- [Noise Protocol](https://noiseprotocol.org/)

### Примеры кода:
- [go-libp2p-examples/chat](https://github.com/libp2p/go-libp2p-examples/tree/master/chat)
- [go-libp2p-examples/echo](https://github.com/libp2p/go-libp2p-examples/tree/master/echo)

---

## 📝 Примечания

1. **Не начинать с упрощённой версии** — целевая функциональность должна быть полной
2. **Сначала бэкенд, потом UI** — убедиться что P2P работает до интеграции с GUI
3. **Тестировать на 2+ машинах** — локальные тесты не выявят проблем с сетью
4. **Логирование обязательно** — для отладки P2P нужно детальное логирование
5. **Graceful shutdown** — libp2p требует корректного закрытия соединений

---

**Автор:** Qwen Code  
**Дата создания:** 19 февраля 2026  
**Статус:** План на согласовании
