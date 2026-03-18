# 🏗️ Архитектура ProjectT

**Версия:** 1.0  
**Дата:** Март 2026  
**Автор:** Егор Редоран

---

## 📋 Содержание

1. [Обзор архитектуры](#1-обзор-архитектуры)
2. [Структура проекта](#2-структура-проекта)
3. [Слой 1: UI Layer (Fyne)](#3-слой-1-ui-layer-fyne)
4. [Слой 2: Business Logic Layer](#4-слой-2-business-logic-layer)
5. [Слой 3: Data Access Layer](#5-слой-3-data-access-layer)
6. [P2P Подсистема](#6-p2p-подсистема)
7. [Модель данных](#7-модель-данных)
8. [Технологический стек](#8-технологический-стек)
9. [Паттерны проектирования](#9-паттерны-проектирования)
10. [Поток данных](#10-поток-данных)
11. [Безопасность](#11-безопасность)
12. [Производительность](#12-производительность)

---

## 1. Обзор архитектуры

ProjectT построен по **трёхслойной архитектуре** с дополнительным P2P-слоем для распределённого взаимодействия:

```
┌─────────────────────────────────────────────────────────────────┐
│                     PRESENTATION LAYER                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  UI Components (Fyne GUI Framework)                     │   │
│  │  • Sidebar (теги, избранное, настройки)                 │   │
│  │  │  Workspace (сетка карточек, редакторы)               │   │
│  │  │  Header (фильтры, поиск, сортировка)                 │   │
│  │  │  Chat Panel (P2P чат, профили)                       │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     BUSINESS LOGIC LAYER                        │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐  │
│  │  Items  │ │  Tags   │ │Favorites│ │ Pinned  │ │  Chat   │  │
│  │ Service │ │ Service │ │ Service │ │ Service │ │ Service │  │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │           P2P Network Services (libp2p)                 │   │
│  │  • Discovery  • Chat  • Transfer  • Profile  • Sync    │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     DATA ACCESS LAYER                           │
│  ┌──────────────────────────┐  ┌────────────────────────────┐  │
│  │   SQLite Database        │  │   File System Storage      │  │
│  │   • queries/             │  │   • storage/files/         │  │
│  │   • models/              │  │   • Хеш-идентификация      │  │
│  │   • migrations/          │  │   • SHA-256                │  │
│  └──────────────────────────┘  └────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

**Ключевые принципы:**
- **Separation of Concerns** — разделение ответственности между слоями
- **Dependency Injection** — внедрение зависимостей через интерфейсы
- **Event-Driven Architecture** — событийная модель для UI обновлений
- **Repository Pattern** — абстракция доступа к данным

---

## 2. Структура проекта

```
projectT/
├── cmd/                          # Точки входа приложения
│   └── main.go                   # Главный исполняемый файл
│
├── internal/                     # Внутренняя логика (private API)
│   ├── app/                      # Инициализация приложения
│   │   └── app.go                # Главный класс приложения
│   │
│   ├── config/                   # Конфигурация
│   │   └── config.go             # Загрузка YAML конфигов
│   │
│   ├── services/                 # Бизнес-логика
│   │   ├── chat_service.go       # Сервис чата (event dispatcher)
│   │   ├── content_blocks_service.go  # Обработка контента карточек
│   │   ├── favorites/            # Избранное
│   │   ├── pinned/               # Закреплённые элементы
│   │   ├── tags_service.go       # Управление тегами
│   │   └── p2p/                  # P2P подсистема
│   │       ├── chat/             # P2P чат
│   │       ├── network/          # Сетевая инфраструктура
│   │       ├── profile/          # Обмен профилями
│   │       ├── itemsync/         # Синхронизация элементов
│   │       └── transfer/         # Передача файлов
│   │
│   ├── storage/                  # Хранение данных
│   │   └── database/
│   │       ├── migrations.go     # Миграции БД
│   │       ├── models/           # Data models
│   │       └── queries/          # SQL запросы
│   │
│   └── ui/                       # Пользовательский интерфейс
│       ├── workspace/            # Рабочая область
│       │   ├── chats/            # P2P чаты
│       │   └── elements/         # Сетка элементов
│       ├── sidebar/              # Боковая панель
│       ├── header/               # Верхняя панель
│       └── cards/                # Карточки элементов
│
├── storage/                      # Пользовательские данные
│   ├── files/                    # Файловое хранилище
│   └── projectT.db               # SQLite база данных
│
├── assets/                       # Ресурсы приложения
│   ├── icons/                    # Иконки
│   └── screenshots/              # Скриншоты для документации
│
├── scripts/                      # Вспомогательные скрипты
├── .github/                      # GitHub Actions CI/CD
├── .vscode/                      # Настройки IDE
│
├── go.mod                        # Go модуль зависимости
├── go.sum                        # Хеш-суммы зависимостей
├── config.yaml                   # Конфигурация приложения
├── config.example.yaml           # Пример конфигурации
├── Makefile                      # Make команды
├── make.ps1                      # PowerShell make команды
└── README.md                     # Докуфикация
```

---

## 3. Слой 1: UI Layer (Fyne)

### 3.1. Компоненты UI

**Технология:** [Fyne Toolkit v2](https://fyne.io/)

```
internal/ui/
├── workspace/
│   ├── chats/
│   │   ├── chats.go              # Главный UI чатов
│   │   ├── p2p_panel.go          # Панель управления P2P
│   │   ├── left_panel.go         # Список контактов
│   │   ├── center_panel.go       # Область чата
│   │   ├── right_panel.go        # Профиль контакта
│   │   └── center/
│   │       ├── chat_panel.go     # Компонент панели чата
│   │       └── message_bubble.go # Пузырьки сообщений
│   │
│   └── elements/
│       ├── elements.go           # Сетка элементов
│       └── editor/
│           └── editor.go         # Редактор карточек
│
├── sidebar/
│   ├── sidebar.go                # Боковая панель
│   ├── tags.go                   # Список тегов
│   ├── favorites.go              # Избранное
│   └── transfer_progress.go      # Прогресс P2P передач
│
├── header/
│   ├── header.go                 # Верхняя панель
│   └── filter_window.go          # Окно фильтров
│
└── cards/
    ├── concrete/
    │   ├── folder_card.go        # Карточка папки
    │   ├── element_card.go       # Карточка элемента
    │   └── composite_card.go     # Композитная карточка
    └── hover_preview/
        └── hover_preview.go      # Предпросмотр при наведении
```

### 3.2. Архитектура UI

**Паттерн:** Model-View-Controller (MVC) с адаптацией под Fyne

```
┌─────────────────────────────────────────────────────────┐
│  View (Fyne Widgets)                                    │
│  • widget.Label, widget.Button, widget.Entry            │
│  • container.VBox, container.HBox, container.Border     │
│  • canvas.Image, canvas.Rectangle                       │
└─────────────────────────────────────────────────────────┘
                          ▲
                          │ Refresh()
                          │
┌─────────────────────────────────────────────────────────┐
│  Controller (UI Structs)                                │
│  • UI { content, window, p2pUI, ... }                   │
│  • Обработчики событий: SetOnSendMessage, SetOnContact  │
└─────────────────────────────────────────────────────────┘
                          ▲
                          │ Вызов методов
                          │
┌─────────────────────────────────────────────────────────┐
│  Model (Business Services)                              │
│  • services.ChatService                                 │
│  • services.TagsService                                 │
│  • p2p.network.P2PNetwork                               │
└─────────────────────────────────────────────────────────┘
```

### 3.3. Событийная модель UI

**Механизм:** Event Bus через каналы Go

```go
// internal/services/chat_service.go
type ChatService struct {
    messageChannel chan *ChatMessageEvent
    subscribers    []chan *ChatMessageEvent
}

// Подписка UI на события
func (ui *UI) SubscribeToMessages() {
    chatSvc := services.GetChatService()
    if chatSvc != nil {
        ui.messageChannel = chatSvc.Subscribe()
        go ui.handleMessageEvents()
    }
}

// Обработка событий
func (ui *UI) handleMessageEvents() {
    for event := range ui.messageChannel {
        if ui.currentContact.ID == event.ContactID {
            ui.chatPanel.AddMessage(event.Message, event.IsOutgoing)
        }
    }
}
```

**Преимущества:**
- ✅ Декуплирование компонентов UI
- ✅ Автоматическое обновление при изменении данных
- ✅ Потокобезопасность через каналы
- ✅ Минимизация race conditions

---

## 4. Слой 2: Business Logic Layer

### 4.1. Сервисы предметной области

| Сервис | Файл | Ответственность |
|--------|------|-----------------|
| **TagsService** | `services/tags_service.go` | Управление тегами (создание, редактирование, связывание) |
| **ContentBlocksService** | `services/content_blocks_service.go` | Обработка контента карточек (текст, файлы, ссылки) |
| **FavoritesService** | `services/favorites/service.go` | Избранные элементы |
| **PinnedService** | `services/pinned/service.go` | Закреплённые элементы |
| **ChatService** | `services/chat_service.go` | Event dispatcher для чатов |
| **SortSettingsService** | `services/sort_settings_service.go` | Настройки сортировки и фильтрации |

### 4.2. P2P Сервисы

| Сервис | Файл | Протокол | Ответственность |
|--------|------|----------|-----------------|
| **P2PNetwork** | `p2p/network/network.go` | - | Оркестрация всех P2P компонентов |
| **DiscoveryService** | `p2p/discovery.go` | DHT, mDNS | Обнаружение пиров |
| **ConnectionService** | `p2p/connections.go` | Ping (кастомный) | Мониторинг подключений, keep-alive |
| **ChatService** | `p2p/chat/chat.go` | `/projectt/chat/1.0.0` | Обмен сообщениями |
| **TransferService** | `p2p/transfer/transfer_service.go` | `/projectt/transfer/1.0.0` | Передача файлов |
| **ProfileExchange** | `p2p/profile/profile_exchange.go` | `/projectt/profile/1.0.0` | Обмен профилями |
| **ItemSyncService** | `p2p/itemsync/item_sync.go` | `/projectt/itemsync/1.0.0` | Синхронизация элементов |
| **HelperService** | `p2p/network/helper.go` | `/projectt/helper/1.0.0` | Режим помощника (хранение адресов) |

### 4.3. Взаимодействие сервисов

```
┌──────────────────────────────────────────────────────────────┐
│  Application (cmd/main.go)                                   │
│  └── app.App                                                 │
│      └── InitServices()                                      │
└──────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────────────┐
│  P2P Network (оркестратор)                                   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  P2PNetwork {                                        │   │
│  │    host            host.Host                         │   │
│  │    dht             *dht.IpfsDHT                      │   │
│  │    chat            *chat.Service                     │   │
│  │    transfer        *transfer.Service                 │   │
│  │    profileExchange *profile.ExchangeService          │   │
│  │    itemSync        *itemsync.Service                 │   │
│  │    discovery       *DiscoveryService                 │   │
│  │    connections     *ConnectionService                │   │
│  │  }                                                     │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────────────┐
│  Domain Services                                             │
│  • TagsService                                               │
│  • ContentBlocksService                                      │
│  • FavoritesService                                          │
│  • PinnedService                                             │
└──────────────────────────────────────────────────────────────┘
```

---

## 5. Слой 3: Data Access Layer

### 5.1. База данных (SQLite)

**ORM:** Чистый SQL без ORM (database/sql)

**Структура БД:**

```sql
-- Основные таблицы
items               -- Карточки элементов
tags                -- Теги
item_tags           -- Связь элементов и тегов (M:M)
content_blocks      -- Контент карточек

-- Избранное и закреплённые
favorites           -- Избранные элементы
pinned_elements     -- Закреплённые элементы

-- P2P и чаты
chat_messages       -- Сообщения чата
contacts            -- Контакты/P2P пиры
profiles            -- Профили пользователей
bootstrap_peers     -- Bootstrap узлы P2P

-- Локальные настройки
local_profiles      -- Локальный профиль пользователя
sort_settings       -- Настройки сортировки
```

### 5.2. Паттерн Repository

**Реализация:**

```go
// internal/storage/database/queries/items.go
type ItemRepository interface {
    Create(item *models.Item) error
    GetByID(id int) (*models.Item, error)
    GetAll() ([]*models.Item, error)
    Update(item *models.Item) error
    Delete(id int) error
    GetByTag(tagID int) ([]*models.Item, error)
}

// Реализация
func CreateItem(item *models.Item) error {
    query := `INSERT INTO items (...) VALUES (...)`
    result, err := db.Exec(query, ...)
    // ...
}
```

**Преимущества:**
- ✅ Единая точка доступа к БД
- ✅ Легко тестировать (мокирование)
- ✅ Инкапсуляция SQL запросов
- ✅ Централизованная обработка ошибок

### 5.3. Файловое хранилище

**Стратегия:** Content-Addressable Storage (CAS)

```
storage/files/
├── a1b2c3d4e5f6...      # SHA-256 хэш содержимого
├── f6e5d4c3b2a1...
└── ...
```

**Алгоритм сохранения:**

```go
func SaveFile(content []byte, originalName string) (string, error) {
    // 1. Вычисляем SHA-256 хэш
    hash := sha256.Sum256(content)
    hashStr := hex.EncodeToString(hash[:])
    
    // 2. Проверяем существование (дедупликация)
    filePath := filepath.Join("storage/files", hashStr)
    if fileExists(filePath) {
        return hashStr, nil  // Файл уже есть
    }
    
    // 3. Сохраняем файл
    os.WriteFile(filePath, content, 0644)
    
    return hashStr, nil
}
```

**Преимущества:**
- ✅ Автоматическая дедупликация
- ✅ Целостность данных (проверка хэша)
- ✅ Быстрый поиск по хэшу
- ✅ Невозможность коллизий имён

---

## 6. P2P Подсистема

### 6.1. Архитектура libp2p

**Технология:** [libp2p](https://libp2p.io/)

```
┌─────────────────────────────────────────────────────────────┐
│  Application Layer                                          │
│  • Chat UI  • File Transfer UI  • Profile UI               │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  Protocol Layer (Custom Protocols)                          │
│  /projectt/chat/1.0.0       - Обмен сообщениями            │
│  /projectt/transfer/1.0.0   - Передача файлов              │
│  /projectt/profile/1.0.0    - Обмен профилями              │
│  /projectt/itemsync/1.0.0   - Синхронизация элементов      │
│  /projectt/helper/1.0.0     - Режим помощника              │
│  /projectt/ping/1.0.0       - Keep-alive                   │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  libp2p Core                                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   Host      │  │  Stream     │  │   Peer      │         │
│  │  (Node)     │  │  (Conn)     │  │  (Store)    │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  Transport & Discovery                                      │
│  • TCP / QUIC              - Транспорт                      │
│  • DHT (Kademlia)          - Обнаружение пиров             │
│  • mDNS (zeroconf)         - Локальная сеть                │
│  • Relay                   - Обход NAT                     │
│  • STUN                    - Определение внешнего адреса   │
└─────────────────────────────────────────────────────────────┘
```

### 6.2. Жизненный цикл P2P подключения

```
1. Инициализация хоста
   └─> libp2p.New(opts...)
       ├─> Генерация ключей Ed25519
       ├─> Создание PeerID
       └─> Запуск слушателя на порту

2. Обнаружение пиров
   ├─> DHT.Advertise()  - Реклама себя в сети
   ├─> DHT.FindPeers()  - Поиск других пиров
   ├─> mDNS (локально)  - Обнаружение в LAN
   └─> Bootstrap        - Подключение к известным узлам

3. Подключение
   └─> host.Connect(ctx, peerInfo)
       ├─> Handshake (Noise/TLS)
       ├─> Multiplexing (mplex/yamux)
       └─> Установка stream'а

4. Обмен данными
   ├─> host.NewStream(peerID, protocolID)
   ├─> stream.Write(data)
   ├─> stream.Read(response)
   └─> stream.Close()

5. Мониторинг
   ├─> Keep-alive ping каждые 30 сек
   ├─> При 3 неудачах - пометка offline
   └─> Автопереподключение (до 5 попыток)
```

### 6.3. Формат сообщений чата

```go
type Message struct {
    ID          int64       `json:"id"`
    FromPeerID  string      `json:"from_peer_id"`
    Content     string      `json:"content"`
    ContentType string      `json:"content_type"`
    Metadata    string      `json:"metadata"`
    Timestamp   int64       `json:"timestamp"`
    MessageType MessageType `json:"message_type"`
    Signature   []byte      `json:"signature"`      // Ed25519 подпись
    Encrypted   bool        `json:"encrypted"`      // Флаг шифрования
    Nonce       []byte      `json:"nonce"`          // Nonce для шифрования
}
```

**Процесс отправки:**

```
1. Создание сообщения
   └─> Message{FromPeerID, Content, Timestamp, ...}

2. Подпись
   └─> data = fmt.Sprintf("%s:%s:%d", FromPeerID, Content, Timestamp)
   └─> signature = privKey.Sign(data)

3. Шифрование (опционально)
   └─> encrypted = XOR(content, encryptionKey, nonce)

4. Сериализация
   └─> json.Marshal(msg)

5. Отправка
   └─> stream.Write(data)

6. Получение ACK
   └─> stream.Read(ack)  // 0x01 = успешно
```

---

## 7. Модель данных

### 7.1. Основные модели

```go
// internal/storage/database/models/item.go
type Item struct {
    ID          int       `json:"id"`
    ElementUUID string    `json:"element_uuid"`  // Уникальный UUID
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Type        string    `json:"type"`          // element, folder, link
    ContentMeta string    `json:"content_meta"`  // JSON контент-блоков
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// internal/storage/database/models/tag.go
type Tag struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Color     string    `json:"color"`
    CreatedAt time.Time `json:"created_at"`
}

// internal/storage/database/models/chat_message.go
type ChatMessage struct {
    ID          int       `json:"id"`
    ContactID   int       `json:"contact_id"`
    FromPeerID  string    `json:"from_peer_id"`
    Content     string    `json:"content"`
    ContentType string    `json:"content_type"`
    Metadata    string    `json:"metadata"`
    SentAt      time.Time `json:"sent_at"`
    IsRead      bool      `json:"is_read"`
}
```

### 7.2. Связи между таблицами

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│   items     │       │  item_tags  │       │    tags     │
├─────────────┤       ├─────────────┤       ├─────────────┤
│ id (PK)     │◄──────│ item_id     │       │ id (PK)     │
│ element_uuid│       │ tag_id      │──────►│ name        │
│ title       │       └─────────────┘       │ color       │
│ description │                              └─────────────┘
│ type        │
│ content_meta│       ┌─────────────┐
└─────────────┘       │  favorites  │
                      ├─────────────┤
┌─────────────┐       │ item_id     │
│  contacts   │       │ user_id     │
├─────────────┤       └─────────────┘
│ id (PK)     │
│ peer_id     │       ┌─────────────┐
│ username    │       │   pinned    │
│ multiaddr   │       ├─────────────┤
└─────────────┘       │ item_id     │
                      │ position    │
┌─────────────┐       └─────────────┘
│chat_messages│
├─────────────┤
│ id (PK)     │
│ contact_id  │──────► contacts.id
│ from_peer_id│
│ content     │
└─────────────┘
```

---

## 8. Технологический стек

### 8.1. Основные технологии

| Категория | Технология | Версия | Назначение |
|-----------|------------|--------|------------|
| **Язык программирования** | Go | 1.25.0 | Основной язык разработки |
| **UI фреймворк** | Fyne | v2.x | Кроссплатформенный GUI |
| **База данных** | SQLite | 3.x | Локальное хранение метаданных |
| **P2P библиотека** | libp2p | v0.30+ | Сетевое взаимодействие |
| **DHT** | go-libp2p-kad-dht | v0.20+ | Распределённая хеш-таблица |
| **PubSub** | go-libp2p-pubsub | v0.9+ | Широкосещательная рассылка |

### 8.2. Зависимости (go.mod)

```go
require (
    // UI
    fyne.io/fyne/v2 v2.x.x
    
    // P2P
    github.com/libp2p/go-libp2p v0.30.x
    github.com/libp2p/go-libp2p-kad-dht v0.20.x
    github.com/libp2p/go-libp2p-pubsub v0.9.x
    github.com/multiformats/go-multiaddr v0.9.x
    
    // База данных
    modernc.org/sqlite v1.x.x
    
    // Утилиты
    gopkg.in/yaml.v3 v3.x.x        // YAML конфиги
    github.com/google/uuid v1.x.x  // UUID генерация
    golang.org/x/crypto v0.x.x     // Криптография
)
```

### 8.3. Инструменты разработки

| Инструмент | Назначение |
|------------|------------|
| **Go Modules** | Управление зависимостями |
| **Makefile** | Автоматизация сборки |
| **pre-commit** | Хуки для форматирования кода |
| **GitHub Actions** | CI/CD пайплайны |
| **VS Code** | Основная IDE |
| **go test** | Тестирование |
| **go fmt** | Форматирование кода |
| **go vet** | Статический анализ |

---

## 9. Паттерны проектирования

### 9.1. Использованные паттерны

| Паттерн | Где используется | Преимущества |
|---------|------------------|--------------|
| **Singleton** | `services.GetChatService()` | Глобальный доступ к сервисам |
| **Repository** | `internal/storage/database/queries/` | Абстракция доступа к данным |
| **Observer** | `ChatService.Subscribe()` | Событийное обновление UI |
| **Factory** | `p2p.NewChatService()` | Создание P2P сервисов |
| **Strategy** | Различные типы карточек | Гибкое отображение контента |
| **Decorator** | HoverPreview для карточек | Расширение функциональности |
| **Command** | Обработчики кнопок UI | Инкапсуляция действий |

### 9.2. Пример: Observer Pattern

```go
// Subject
type ChatService struct {
    subscribers []chan *ChatMessageEvent
}

// Метод подписки
func (cs *ChatService) Subscribe() <-chan *ChatMessageEvent {
    ch := make(chan *ChatMessageEvent, 10)
    cs.subscribers = append(cs.subscribers, ch)
    return ch
}

// Метод уведомления
func (cs *ChatService) NotifyNewMessage(...) {
    for _, sub := range cs.subscribers {
        select {
        case sub <- event:
            // Успешно
        default:
            // Канал переполнен
        }
    }
}

// Observer
func (ui *UI) SubscribeToMessages() {
    ch := chatSvc.Subscribe()
    go func() {
        for event := range ch {
            ui.handleMessage(event)
        }
    }()
}
```

---

## 10. Поток данных

### 10.1. Создание элемента

```
1. Пользователь создаёт карточку
   └─> UI: editor.go → onSave()

2. Валидация данных
   └─> services.ContentBlocksService.Validate()

3. Сохранение контента
   ├─> Файлы → storage/files/ (SHA-256 хэш)
   └─> Метаданные → SQLite (items table)

4. Обновление UI
   └─> elements.Refresh() → grid.Reload()
```

### 10.2. Отправка P2P сообщения

```
1. Пользователь вводит сообщение
   └─> UI: center_panel.go → sendMessage()

2. Проверка подключения
   └─> p2p.ChatService.SendMessage(ctx, peerID, content)

3. Если пир онлайн:
   ├─> Создание Message с подписью
   ├─> Шифрование (опционально)
   ├─> Отправка через stream.Write()
   ├─> Получение ACK
   └─> Сохранение в БД (исходящее)

4. Если пир оффлайн:
   └─> Добавление в очередь (queueMessage)

5. Обновление UI
   └─> chatPanel.AddMessage(message, isOutgoing)
```

### 10.3. Получение P2P сообщения

```
1. Получение stream'а
   └─> p2p.ChatService.HandleChatStream(stream)

2. Чтение и десериализация
   └─> json.Unmarshal(data, &Message)

3. Проверка подписи
   └─> pubKey.Verify(data, signature)

4. Расшифровка (если зашифровано)
   └─> XOR decrypt with encryptionKey

5. Сохранение в БД
   └─> queries.CreateChatMessage(message)

6. Уведомление UI
   └─> services.ChatService.NotifyNewMessage()
       └─> eventChannel ← ChatMessageEvent
           └─> UI.handleMessageEvents()
               └─> chatPanel.AddMessage(message, isIncoming)
```

---

## 11. Безопасность

### 11.1. Криптография

| Компонент | Алгоритм | Назначение |
|-----------|----------|------------|
| **Ключи доступа** | Ed25519 (32 байта) | Идентификация пира |
| **Подпись сообщений** | Ed25519 Sign/Verify | Верификация отправителя |
| **Шифрование сообщений** | XOR + nonce | Конфиденциальность |
| **Хеширование файлов** | SHA-256 | Целостность и дедупликация |
| **Защита ключей** | Мастер-пароль (AES-256) | Шифрование приватного ключа |

### 11.2. Модель угроз

| Угроза | Мера защиты |
|--------|-------------|
| **Подмена пира** | Ed25519 подпись профилей |
| **Перехват сообщений** | Шифрование с симметричным ключом |
| **Повторная отправка** | Timestamp + nonce в подписи |
| **DoS атака** | Лимит подключений, таймауты |
| **Утечка ключей** | Шифрование мастер-паролем |
| **Нежелательные пиры** | Чёрный список контактов |

### 11.3. Хранение ключей

```
┌─────────────────────────────────────────────────────────┐
│  Ключи хранятся в БД (local_profiles)                   │
│                                                          │
│  • public_key  - открытый ключ (не зашифрован)          │
│  • private_key - закрытый ключ (зашифрован AES-256)     │
│                                                          │
│  Расшифровка:                                           │
│  1. Пользователь вводит мастер-пароль                   │
│  2. Из пароля выводится ключ (PBKDF2)                   │
│  3. Ключом расшифровывается private_key                 │
│  4. Ключ хранится в памяти до закрытия приложения       │
└─────────────────────────────────────────────────────────┘
```

---

## 12. Производительность

### 12.1. Оптимизации

| Область | Оптимизация | Эффект |
|---------|-------------|--------|
| **БД** | Индексы на foreign keys | Быстрый JOIN запросов |
| **БД** | Пакетная вставка сообщений | Уменьшение I/O операций |
| **Файлы** | Дедупликация по хэшу | Экономия места на диске |
| **UI** | Ленивая загрузка карточек | Быстрый старт приложения |
| **UI** | Кэширование превью | Мгновенный hover-эффект |
| **P2P** | Keep-alive ping | Быстрое обнаружение оффлайна |
| **P2P** | Очередь сообщений | Гарантия доставки |

### 12.2. Метрики производительности

| Метрика | Значение | Примечание |
|---------|----------|------------|
| **Старт приложения** | < 2 сек | Зависит от размера БД |
| **Загрузка сетки (100 элементов)** | < 500 мс | С кэшированием |
| **Отправка сообщения** | < 100 мс | Локально |
| **P2P подключение** | 1-5 сек | Зависит от сети |
| **Передача файла (1 MB)** | 2-10 сек | Зависит от канала |
| **Поиск по тегам** | < 100 мс | С индексами |

### 12.3. Профилирование

**Инструменты:**
- `go test -bench=` - Бенчмарки
- `go tool pprof` - Профилирование CPU/Memory
- `go test -race` - Detection race conditions

**Пример бенчмарка:**

```go
func BenchmarkSaveMessage(b *testing.B) {
    for i := 0; i < b.N; i++ {
        msg := &ChatMessage{
            ContactID: 1,
            Content:   "Test message",
            SentAt:    time.Now(),
        }
        queries.CreateChatMessage(msg)
    }
}
```

---

## 📎 Приложения

### A. Диаграмма последовательности: P2P Чат

```
Пользователь A          P2P Network          Пользователь B
    │                       │                       │
    │── Ввод сообщения ───> │                       │
    │                       │                       │
    │                       │── Create Message ───> │
    │                       │   Sign & Encrypt      │
    │                       │                       │
    │                       │── stream.Write() ───> │
    │                       │                       │── Получение stream ──>
    │                       │                       │── Verify Signature ──>
    │                       │                       │── Decrypt ──>
    │                       │                       │── Save to DB ──>
    │                       │                       │
    │                       │<── stream.Write(ACK) ─│
    │                       │                       │
    │<── Обновление UI ──── │                       │
    │                       │                       │── Обновление UI ──>
```

### B. Список используемых Go пакетов

```
internal/
├── app/              # Инициализация приложения
├── config/           # Конфигурация
├── services/         # Бизнес-логика
│   ├── favorites/    # Избранное
│   ├── pinned/       # Закреплённые
│   └── p2p/          # P2P сервисы
│       ├── chat/     # Чат
│       ├── network/  # Сеть
│       ├── profile/  # Профили
│       ├── itemsync/ # Синхронизация
│       └── transfer/ # Передача файлов
├── storage/
│   └── database/
│       ├── migrations/  # Миграции
│       ├── models/      # Модели данных
│       └── queries/     # SQL запросы
└── ui/
    ├── workspace/    # Рабочая область
    ├── sidebar/      # Боковая панель
    ├── header/       # Верхняя панель
    └── cards/        # Карточки
```

---

<div align="center">

**ProjectT — Архитектура дипломного проекта**

*Документация актуальна на момент защиты*

Made with ❤️ by Egor Redoran

</div>
