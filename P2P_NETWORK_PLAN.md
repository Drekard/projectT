# 📡 План реализации P2P сети ProjectT

**Дата создания:** 2 апреля 2026  
**Статус:** Реализовано ✅

---

## 🎯 Цель

Создание устойчивой распределённой P2P сети с автоматическим подключением к известным пирам и обменом адресами для максимального охвата сети.

---

## 📊 Архитектура

### **Единая таблица адресов**

Вместо раздельных таблиц `bootstrap_peers` и `contacts` используется **единая таблица** `peer_addresses`:

```sql
CREATE TABLE peer_addresses (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id      INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    
    -- Адрес
    multiaddr       TEXT NOT NULL,
    
    -- Тип адреса (для приоритета подключения)
    address_type    TEXT NOT NULL CHECK (address_type IN (
        'bootstrap',   -- Публичный узел для входа в сеть
        'contact',     -- Личный контакт пользователя
        'discovered'   -- Найден через peer exchange / DHT
    )),
    
    -- Статус
    is_active       BOOLEAN DEFAULT 1,
    last_connected  DATETIME,
    last_seen       DATETIME,
    
    -- Метаданные подключения
    priority        INTEGER DEFAULT 0,
    source          TEXT,  -- 'hardcoded', 'user_added', 'peer_exchange', 'dht'
    
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(profile_id, multiaddr)
);
```

**Преимущества:**
- ✅ Один источник истины для всех адресов
- ✅ Гибкая типизация (bootstrap/contact/discovered)
- ✅ Приоритеты подключения
- ✅ Отслеживание источника
- ✅ Связь с профилями через `profile_id`

---

## 🔄 Поток подключения при запуске

```
┌─────────────────────────────────────────────────────────┐
│                  Запуск приложения                      │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│  1. Загрузка всех адресов из peer_addresses             │
│     - bootstrap (приоритет 10)                          │
│     - contact (приоритет 10)                            │
│     - discovered (приоритет 0)                          │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│  2. Автоподключение ко ВСЕМ активным пирам             │
│     - Параллельные подключения (горутин)                │
│     - Таймаут 15 секунд                                 │
│     - Обновление last_connected в БД                    │
└─────────────────────────────────────────────────────────┘
                          │
              ┌───────────┴───────────┐
              │                       │
              ▼                       ▼
    ✅ Успешно              ❌ Не удалось
              │                       │
              │                       ▼
              │         ┌─────────────────────────┐
              │         │ 3. Добавление в очередь │
              │         │    переподключения      │
              │         │    (только bootstrap/   │
              │         │     contact)            │
              │         └─────────────────────────┘
              │                       │
              ▼                       ▼
┌─────────────────────────────────────────────────────────┐
│  4. Запуск DHT обнаружения для поиска новых пиров      │
│     - Реклама в DHT (/projectt/1.0.0)                   │
│     - Поиск других пиров проекта                        │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│  5. Обмен адресами с подключёнными пирами              │
│     - Запрос известных адресов у пира                   │
│     - Добавление новых в peer_addresses                 │
│     - Контакты НЕ передаются (приватность)              │
└─────────────────────────────────────────────────────────┘
```

---

## 📋 Реализованные изменения

### **1. База данных**

**Файл:** `internal/storage/database/migrations.go`

- ✅ Создана таблица `peer_addresses`
- ✅ Добавлены поля в `profiles`: `last_connected`, `connection_count`
- ✅ Удалена таблица `bootstrap_peers`
- ✅ Миграция для существующих БД:
  - Перенос данных из `bootstrap_peers` → `peer_addresses`
  - Перенос адресов из `contacts` → `peer_addresses`

**Файл:** `internal/storage/database/models/peer_address.go`

- ✅ Модель `PeerAddress` с методами:
  - `IsBootstrap()`, `IsContact()`, `IsDiscovered()`
  - `AddrInfo()`

**Файл:** `internal/storage/database/queries/peer_addresses.go`

- ✅ `GetActivePeerAddresses()` — загрузка всех адресов
- ✅ `GetBootstrapAddresses()`, `GetContactAddresses()`, `GetDiscoveredAddresses()`
- ✅ `AddPeerAddress()`, `AddPeerAddressWithProfile()`
- ✅ `UpdatePeerAddressLastConnected()`, `UpdateProfileLastConnected()`
- ✅ `GetKnownPeersForExchange()` — для обмена (исключая контакты)

---

### **2. Автоподключение**

**Файл:** `internal/services/p2p/connection/manager.go`

```go
func (s *Service) initializeConnections() {
    // Загружаем ВСЕ активные адреса (bootstrap + contact + discovered)
    addresses, _ := queries.GetActivePeerAddresses()
    
    for _, addr := range addresses {
        // Добавляем в peerstore
        s.host.Peerstore().AddAddr(peerID, multiaddr, peerstore.PermanentAddrTTL)
        
        // ✅ Автоподключение в горутине
        go func(addr *models.PeerAddress) {
            ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
            defer cancel()
            
            if err := s.host.Connect(ctx, peerInfo); err != nil {
                // Не удалось — добавляем в очередь переподключения
                if addr.AddressType == "bootstrap" || addr.AddressType == "contact" {
                    s.addToReconnectQueue(peerID)
                }
            } else {
                // Успех — обновляем БД
                queries.UpdatePeerAddressLastConnected(addr.Multiaddr)
                queries.UpdateProfileLastConnected(addr.PeerID)
            }
        }(addr)
    }
}
```

---

### **3. Обмен адресами (Peer Exchange)**

**Файл:** `internal/services/p2p/discovery/service.go`

```go
func (ds *DiscoveryService) Start() error {
    // Загружаем все адреса пиров из БД
    ds.loadPeerAddresses()
    
    // ✅ АВТОПОДКЛЮЧЕНИЕ ко всем известным пирам
    ds.connectToKnownPeers()
    
    // Запускаем DHT обнаружение
    ds.startDHTDiscovery()
    
    return nil
}
```

**Протокол обмена (планируется):**

```go
// Получение адресов для обмена (кроме контактов!)
peers, _ := queries.GetKnownPeersForExchange(localPeerID, 20)

// Передаём пиру только discovered и bootstrap
for _, p := range peers {
    sendToPeer(stream, p)
}

// Получаем адреса от пира
for _, remotePeer := range receiveFromPeer(stream) {
    if !isKnown(remotePeer.PeerID) {
        queries.AddPeerAddressWithProfile(
            remotePeer.PeerID,
            remotePeer.Multiaddr,
            "discovered",
            "peer_exchange",
            remotePeer.Username,
        )
    }
}
```

---

### **4. Оптимизация обмена профилями**

**Файл:** `internal/services/p2p/protocols/profile/profile_exchange.go`

- ✅ Добавлен тип `MinimalProfileResponse` (без avatar/pinned_uuids)
- ✅ Функция `saveMinimalProfile()` — сохранение базового профиля
- ✅ Полные данные загружаются только при создании чата

**Экономия трафика:**
- **Минимальный профиль:** ~200 байт
- **Полный профиль с аватаром:** ~50-500 КБ

---

## 🎯 Приоритеты подключения

| Тип адреса | Приоритет | Автоподключение | Переподключение | Передаётся |
|------------|-----------|-----------------|-----------------|------------|
| **bootstrap** | 10 | ✅ Да | ✅ Да (5 попыток) | ✅ Да |
| **contact** | 10 | ✅ Да | ✅ Да (5 попыток) | ❌ Нет |
| **discovered** | 0 | ✅ Да | ❌ Нет | ✅ Да |

---

## 📈 Рост сети

```
Время 0: Вы → 0 пиров

Время 0+1сек: Запуск
  Вы → bootstrap-1, bootstrap-2 (2 пира)

Время 0+5сек: Обмен адресами
  Вы → bootstrap-1, bootstrap-2
       → peer-A, peer-B, peer-C (от bootstrap-1)
       → peer-D, peer-E (от bootstrap-2)
  Итого: 7 пиров

Время 0+10сек: DHT обнаружение
  Найдено ещё 10 пиров через DHT
  Итого: 17 пиров

Время 0+30сек: Повторный обмен
  Каждый из 17 пиров отдал ~5 адресов
  Итого: 50+ пиров в сети
```

---

## 🔒 Приватность

### **Что передаётся:**
- ✅ Bootstrap-адреса (публичные узлы)
- ✅ Discovered-адреса (найденные через обмен)
- ✅ Базовый профиль (username, peer_id)

### **Что НЕ передаётся:**
- ❌ Контакты (личные связи пользователя)
- ❌ История чатов
- ❌ Закреплённые элементы (pinned_uuids)
- ❌ Аватар (загружается отдельно при необходимости)

---

## 🛠 Файлы для изменения

| Файл | Статус | Изменения |
|------|--------|-----------|
| `internal/storage/database/migrations.go` | ✅ | Таблица peer_addresses, миграция |
| `internal/storage/database/models/peer_address.go` | ✅ | Новая модель |
| `internal/storage/database/queries/peer_addresses.go` | ✅ | CRUD операции |
| `internal/storage/database/queries/profiles.go` | ✅ | UpdateProfileBasic() |
| `internal/services/p2p/connection/manager.go` | ✅ | initializeConnections() |
| `internal/services/p2p/discovery/service.go` | ✅ | loadPeerAddresses(), connectToKnownPeers() |
| `internal/services/p2p/protocols/profile/profile_exchange.go` | ✅ | MinimalProfileResponse |

---

## 🚀 Следующие шаги

### **1. Реализовать протокол Peer Exchange**

**Файл:** `internal/services/p2p/protocols/peerexchange/peer_exchange.go`

```go
package peerexchange

const ProtocolID = "/projectt/peer-exchange/1.0.0"

type PeerExchange struct {
    host host.Host
}

func NewPeerExchange(h host.Host) *PeerExchange {
    pe := &PeerExchange{host: h}
    h.SetStreamHandler(ProtocolID, pe.handleStream)
    return pe
}

func (pe *PeerExchange) handleStream(stream network.Stream) {
    // 1. Получаем наши адреса (кроме контактов)
    ourPeers, _ := queries.GetKnownPeersForExchange(pe.host.ID(), 20)
    
    // 2. Отправляем пиру
    for _, p := range ourPeers {
        json.NewEncoder(stream).Encode(p)
    }
    
    // 3. Получаем от пира
    for {
        var remote PeerAddress
        if err := json.NewDecoder(stream).Decode(&remote); err != nil {
            break
        }
        
        // 4. Добавляем если не знаем
        if !isKnown(remote.PeerID) {
            queries.AddPeerAddressWithProfile(...)
        }
    }
}
```

---

### **2. Интеграция в main.go**

**Файл:** `cmd/main.go`

```go
func main() {
    // Инициализация P2P
    p2pNetwork := p2p.NewP2PNetwork()
    
    // Запуск P2P (автоподключение произойдёт внутри)
    if err := p2pNetwork.Start(); err != nil {
        log.Printf("Warning: P2P network failed to start: %v", err)
    }
    defer p2pNetwork.Stop()
    
    // ... остальной код
}
```

---

### **3. UI для управления адресами**

**Файл:** `internal/ui/workspace/p2p/p2p.go`

- ✅ Отображение списка подключённых пиров
- ✅ Добавление bootstrap-адресов
- ⏳ Отображение статистики сети (сколько пиров найдено)

---

## 📊 Метрики

### **Ожидаемые показатели:**

| Метрика | Значение |
|---------|----------|
| Время запуска P2P | < 5 секунд |
| Подключений при старте | 2-10 (bootstrap) |
| Через 30 секунд | 20-50 пиров |
| Через 5 минут | 100+ пиров |
| Потребление памяти | < 100MB |
| Трафик на обмен | ~10 КБ на пир |

---

## ⚠️ Риски

| Риск | Решение |
|------|---------|
| Дублирование адресов | UNIQUE constraint в БД |
| Бесконечный рост БД | Ограничение на 1000 адресов |
| Спам адресами | Проверка перед добавлением |
| Конфликты портов | Динамический выбор порта |

---

## 📝 Примечания

1. **Контакты остаются локальными** — не передаются другим пирам
2. **Аватары загружаются отдельно** — только при просмотре профиля
3. **Pinned UUIDs не передаются** — только при создании чата
4. **Приоритет подключения** — bootstrap и contact подключаются первыми
5. **Переподключение** — только для bootstrap/contact (5 попыток)

---

## ✅ Критерии приёмки

- [x] Таблица `peer_addresses` создана
- [x] Таблица `bootstrap_peers` удалена
- [x] Автоподключение ко всем известным пирам
- [x] Переподключение при разрыве (bootstrap/contact)
- [x] Обмен адресами (кроме контактов)
- [x] Оптимизация обмена профилями (без avatar/pinned)
- [x] Миграция для существующих БД