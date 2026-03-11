# 🗺️ Карта UI и функционала ProjectT

## 📱 Общая архитектура

```
UI Layer (Fyne)
├── left_panel.go      — навигация (контакты)
├── center_panel.go    — P2P настройки (7 секций)
└── profile_area.go    — профиль контакта
        ▼
Service Layer (P2P Network)
├── network/p2p.go     — P2P хост (libp2p)
├── network/ui_api.go  — API для UI (обёртки)
├── chat.go            — сообщения (подпись, отправка)
├── discovery.go       — mDNS + DHT обнаружение
└── address_tools.go   — адреса, firewall, NAT
        ▼
Storage Layer (SQLite)
├── contacts.go        — CRUD контактов
├── messages.go        — CRUD сообщений
└── bootstrap.go       — CRUD bootstrap пиров
```

---

## 🎛️ Панель управления P2P (center_panel.go) — 7 секций

### 1️⃣ Ваш адрес
- **Копировать** → `copyMyAddress()` → `CopyPeerAddress()` → буфер обмена
- **Проверить порт** → `checkPortAccessibility()` → `CheckFirewall(8080)` → показывает команды PowerShell/CMD

### 2️⃣ Добавить контакт
- **Поле адреса** → ввод `projectt:QmPeerID@/ip4/.../tcp/.../p2p/...`
- **➕ Добавить** → `addContactByAddress()` → `ImportPeerAddress()` → `CreateContact()` в БД
- **📁 Подключиться** → `connectToContact()` → `ConnectToContact()` → подключение к пиру

### 3️⃣ Состояние подключения
- **Статус** → `connectionStatusLabel` ← `GetStatus().IsRunning`
- **Пиры** → `peersCountLabel` ← `GetStatus().ConnectedPeers`
- **NAT** → `natStatusLabel` ← `GetNATStatus().Message`
- **Обновить** → `refreshConnectionStatus()`

### 4️⃣ Подключённые пиры
- **Список** → `loadConnectedPeers()` ← `GetConnectedPeers()`
- **Обновить** → `startPeerDiscovery()` + `loadConnectedPeers()`

### 5️⃣ Настройки P2P
| Настройка | Сохраняется в | Зачем |
|-----------|---------------|-------|
| Порт (8080) | `settings.ListenPort` | Порт для входящих |
| NAT Port Map | `settings.EnableNATPortMap` | UPnP в роутере |
| Relay | `settings.EnableRelay` | Обход NAT |
| DHT | `settings.EnableDHT` | Глобальный поиск пиров |
| mDNS | `settings.EnableMDNS` | Локальная сеть |
| STUN | `settings.EnableSTUN` | Узнать внешний IP |
| Helper | `settings.EnableHelperMode` | Помощь другим |

- **Сохранить** → `saveP2PSettings()` → `UpdateSettings()` → перезапуск хоста
- **Загрузить** → `loadP2PSettings()` → `GetSettings()`

### 6️⃣ Bootstrap пиры
- **Добавить** → `addBootstrapPeer()` → `AddBootstrapPeer()` → `CreateBootstrapPeer()` в БД
- **Обновить** → `loadBootstrapPeers()` ← `GetBootstrapPeers()`

**Зачем:** Точки входа в сеть при запуске

### 7️⃣ Обнаруженные пиры (mDNS/DHT)
- **Обновить** → `startPeerDiscovery()` + `loadDiscoveredPeers()` ← `GetDiscoveredPeers()`
- **Подключиться** → копирует адрес в поле ввода

---

## 🔄 Цикл подключения двух устройств

```
1. Настройка (оба) → Сохранить настройки P2P
2. Запуск (оба) → P2P хост стартует автоматически
3. Обмен адресами → B копирует адрес → отправляет A
4. Добавление (A) → Вставляет адрес B → Добавить контакт
5. Подключение (A) → Клик на контакт → подключение к B
6. Сообщение (A→B) → Ввод текста → Отправить → ChatService.SendMessage()
```

---

## 🧪 Тестирование

### Вариант 1: Два устройства в одной сети
1. Оба устройства: сохранить настройки P2P
2. Устройство B: копировать адрес → отправить A
3. Устройство A: вставить адрес → Добавить контакт
4. Проверить: "Подключённые пиры: 1"
5. Отправить сообщение

### Вариант 2: Один компьютер, два процесса
1. Запуск 1: `./projectT.exe`
2. Изменить config.yaml: `listen_port: 8081`
3. Запуск 2: `./projectT.exe --config=config2.yaml`
4. Повторить шаги из Вариант 1

---

## 📊 Таблица: UI → P2P → БД

| Кнопка | Метод UI | Метод P2P | Метод БД |
|--------|----------|-----------|----------|
| Копировать адрес | `copyMyAddress()` | `CopyPeerAddress()` | — |
| Проверить порт | `checkPortAccessibility()` | `CheckFirewall()` | — |
| Добавить контакт | `addContactByAddress()` | `ImportPeerAddress()` | `CreateContact()` |
| Подключиться | `connectToContact()` | `ConnectToContact()` | — |
| Обновить статус | `refreshConnectionStatus()` | `GetStatus()`, `GetNATStatus()` | — |
| Подключённые пиры | `loadConnectedPeers()` | `GetConnectedPeers()` | `GetContactByPeerID()` |
| Сохранить настройки | `saveP2PSettings()` | `UpdateSettings()` | — |
| Загрузить настройки | `loadP2PSettings()` | `GetSettings()` | — |
| Добавить bootstrap | `addBootstrapPeer()` | `AddBootstrapPeer()` | `CreateBootstrapPeer()` |
| Обнаруженные пиры | `loadDiscoveredPeers()` | `GetDiscoveredPeers()` | — |

---

## 🎯 Минимальный тест

1. Запуск: `./projectT.exe`
2. Проверка: "Состояние подключения" → "Статус: подключено"
3. Копировать: нажать 📋 в "Ваш адрес"
4. Добавить: вставить адрес в "Добавить контакт", имя "Тест"
5. Проверка: контакт появился слева → кликнуть → отправить сообщение
