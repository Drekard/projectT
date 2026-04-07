# 🏗️ ProjectT Architecture

**Version:** 1.1 (updated)
**Date:** March 2026
**Author:** Egor Redoran

---

## 📋 Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Project Structure](#2-project-structure)
3. [Layer 1: UI Layer (Fyne)](#3-layer-1-ui-layer-fyne)
4. [Layer 2: Business Logic Layer](#4-layer-2-business-logic-layer)
5. [Layer 3: Data Access Layer](#5-layer-3-data-access-layer)
6. [P2P Subsystem](#6-p2p-subsystem)
7. [Data Model](#7-data-model)
8. [Technology Stack](#8-technology-stack)
9. [Design Patterns](#9-design-patterns)
10. [Data Flow](#10-data-flow)
11. [Security](#11-security)
12. [Performance](#12-performance)

---

## 1. Architecture Overview

ProjectT is built on a **three-layer architecture** with an additional P2P layer for distributed interaction:

```
┌─────────────────────────────────────────────────────────────────┐
│                     PRESENTATION LAYER                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  UI Components (Fyne GUI Framework)                     │   │
│  │  • Sidebar (tags, favorites, settings)                  │   │
│  │  │  Workspace (card grid, editors)                      │   │
│  │  │  Header (filters, search, sorting)                   │   │
│  │  │  Chat Panel (P2P chat, profiles)                     │   │
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
│  │   • models/              │  │   • Hash identification    │  │
│  │   • migrations/          │  │   • SHA-256                │  │
│  └──────────────────────────┘  └────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

**Key Principles:**
- **Separation of Concerns** — separation of responsibilities between layers
- **Dependency Injection** — dependency injection via interfaces
- **Event-Driven Architecture** — event model for UI updates
- **Repository Pattern** — data access abstraction

---

## 2. Project Structure

```
projectT/
├── cmd/                          # Application entry points
│   ├── main.go                   # Main executable file
│   ├── clear_db/                 # DB cleanup utility
│   │   └── main.go
│   └── db_viewer/                # DB viewer utility
│       └── main.go
│
├── internal/                     # Internal logic (private API)
│   ├── app/                      # Application initialization
│   │   └── app.go                # Main application class
│   │
│   ├── config/                   # Configuration
│   │   └── config.go             # YAML config loading
│   │
│   ├── services/                 # Business logic
│   │   ├── items_service.go      # Card items CRUD
│   │   ├── tags_service.go       # Tag management
│   │   ├── content_blocks_service.go  # Card content processing
│   │   ├── chat_service.go       # Chat service (event dispatcher)
│   │   ├── favorites/            # Favorites
│   │   │   └── service.go
│   │   ├── pinned/               # Pinned items
│   │   │   └── service.go
│   │   ├── contacts/             # Contact management
│   │   │   └── contacts_service.go
│   │   ├── crypto/               # Cryptography
│   │   │   └── crypto.go
│   │   └── p2p/                  # P2P subsystem
│   │       ├── core/             # P2P network core
│   │       │   ├── p2p.go
│   │       │   ├── host.go
│   │       │   ├── network.go
│   │       │   └── keys.go
│   │       ├── discovery/        # Peer discovery
│   │       │   └── service.go
│   │       ├── connection/       # Connection management
│   │       │   ├── manager.go
│   │       │   └── ping.go
│   │       ├── autodial/         # Auto-dial
│   │       │   ├── manager.go
│   │       │   ├── dialer.go
│   │       │   └── queue.go
│   │       ├── peerexchange/     # Peer address exchange
│   │       │   └── exchange.go
│   │       ├── helper/           # Helper mode
│   │       │   └── service.go
│   │       └── protocols/        # P2P protocols
│   │           ├── chat/         # P2P chat
│   │           │   └── chat.go
│   │           ├── transfer/     # File transfer
│   │           │   └── transfer_service.go
│   │           ├── profile/      # Profile exchange
│   │           │   └── profile_exchange.go
│   │           ├── itemsync/     # Item synchronization
│   │           │   └── item_sync.go
│   │           └── avatar/       # Avatar exchange
│   │               └── avatar_service.go
│   │
│   ├── controllers/              # Controllers
│   │   ├── chat_controller.go    # Chat controller
│   │   └── contact_controller.go # Contact controller
│   │
│   ├── storage/                  # Data storage
│   │   ├── database/             # Database
│   │   │   ├── connection.go     # SQLite connection
│   │   │   ├── migrations.go     # DB schema migrations
│   │   │   ├── models/           # Data models
│   │   │   └── queries/          # SQL queries
│   │   └── filesystem/           # File storage
│   │       └── (content-addressable storage)
│   │
│   └── ui/                       # User interface
│       ├── ui.go                 # Main UI class
│       ├── workspace/            # Workspace
│       │   ├── chats/            # P2P chats
│       │   └── elements/         # Elements grid
│       ├── sidebar/              # Sidebar
│       ├── header/               # Header bar
│       ├── cards/                # Item cards
│       └── theme/                # Custom theme
│
├── storage/                      # User data
│   ├── files/                    # File storage (SHA-256)
│   └── projectT.db               # SQLite database
│
├── assets/                       # Application resources
│   ├── icons/                    # Icons
│   └── screenshots/              # Documentation screenshots
│
├── .github/                      # GitHub Actions CI/CD
├── .vscode/                      # IDE settings
│
├── go.mod                        # Go module dependencies
├── go.sum                        # Dependency hash sums (~130 dependencies)
├── config.yaml                   # Application configuration
├── config.example.yaml           # Example configuration
├── Makefile                      # Make commands
├── make.ps1                      # PowerShell make commands
└── README.md                     # Documentation
```

---

## 3. Layer 1: UI Layer (Fyne)

### 3.1. UI Components

**Technology:** [Fyne Toolkit v2](https://fyne.io/)

```
internal/ui/
├── workspace/
│   ├── chats/
│   │   ├── chats.go              # Main chat UI
│   │   ├── p2p_panel.go          # P2P management panel
│   │   ├── left_panel.go         # Contact list
│   │   ├── center_panel.go       # Chat area
│   │   ├── right_panel.go        # Contact profile
│   │   └── center/
│   │       ├── chat_panel.go     # Chat panel component
│   │       └── message_bubble.go # Message bubbles
│   │
│   └── elements/
│       ├── elements.go           # Elements grid
│       └── editor/
│           └── editor.go         # Card editor
│
├── sidebar/
│   ├── sidebar.go                # Sidebar
│   ├── tags.go                   # Tag list
│   ├── favorites.go              # Favorites
│   └── transfer_progress.go      # P2P transfer progress
│
├── header/
│   ├── header.go                 # Header bar
│   └── filter_window.go          # Filter window
│
└── cards/
    ├── concrete/
    │   ├── folder_card.go        # Folder card
    │   ├── element_card.go       # Element card
    │   └── composite_card.go     # Composite card
    └── hover_preview/
        └── hover_preview.go      # Hover preview
```

### 3.2. UI Architecture

**Pattern:** Model-View-Controller (MVC) adapted for Fyne

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
│  • Event handlers: SetOnSendMessage, SetOnContact       │
└─────────────────────────────────────────────────────────┘
                          ▲
                          │ Method calls
                          │
┌─────────────────────────────────────────────────────────┐
│  Model (Business Services)                              │
│  • services.ChatService                                 │
│  • services.TagsService                                 │
│  • p2p.network.P2PNetwork                               │
└─────────────────────────────────────────────────────────┘
```

### 3.3. UI Event Model

**Mechanism:** Event Bus via Go channels

```go
// internal/services/chat_service.go
type ChatService struct {
    messageChannel chan *ChatMessageEvent
    subscribers    []chan *ChatMessageEvent
}

// UI subscribes to events
func (ui *UI) SubscribeToMessages() {
    chatSvc := services.GetChatService()
    if chatSvc != nil {
        ui.messageChannel = chatSvc.Subscribe()
        go ui.handleMessageEvents()
    }
}

// Event handling
func (ui *UI) handleMessageEvents() {
    for event := range ui.messageChannel {
        if ui.currentContact.ID == event.ContactID {
            ui.chatPanel.AddMessage(event.Message, event.IsOutgoing)
        }
    }
}
```

**Advantages:**
- ✅ Decoupling of UI components
- ✅ Automatic updates on data changes
- ✅ Thread safety via channels
- ✅ Minimized race conditions

---

## 4. Layer 2: Business Logic Layer

### 4.1. Domain Services

| Service | File | Responsibility |
|--------|------|----------------|
| **TagsService** | `services/tags_service.go` | Tag management (creation, editing, linking) |
| **ContentBlocksService** | `services/content_blocks_service.go` | Card content processing (text, files, links) |
| **FavoritesService** | `services/favorites/service.go` | Favorite items |
| **PinnedService** | `services/pinned/service.go` | Pinned items |
| **ChatService** | `services/chat_service.go` | Event dispatcher for chats |
| **SortSettingsService** | `services/sort_settings_service.go` | Sorting and filtering settings |

### 4.2. P2P Services

| Service | File | Protocol | Responsibility |
|--------|------|----------|----------------|
| **P2PCore** | `p2p/core/p2p.go` | - | P2P network core (host, keys, network) |
| **DiscoveryService** | `p2p/discovery/service.go` | DHT, mDNS | Peer discovery |
| **ConnectionManager** | `p2p/connection/manager.go` | Ping (custom) | Connection monitoring, keep-alive |
| **AutoDial** | `p2p/autodial/manager.go` | - | Auto-dial to known peers |
| **PeerExchange** | `p2p/peerexchange/exchange.go` | - | Peer address exchange |
| **ChatService** | `p2p/protocols/chat/chat.go` | `/projectt/chat/1.0.0` | Message exchange |
| **TransferService** | `p2p/protocols/transfer/transfer_service.go` | `/projectt/transfer/1.0.0` | File transfer |
| **ProfileExchange** | `p2p/protocols/profile/profile_exchange.go` | `/projectt/profile/1.0.0` | Profile exchange |
| **ItemSyncService** | `p2p/protocols/itemsync/item_sync.go` | `/projectt/itemsync/1.0.0` | Item synchronization |
| **AvatarService** | `p2p/protocols/avatar/avatar_service.go` | `/projectt/avatar/1.0.0` | Avatar exchange |
| **HelperService** | `p2p/helper/service.go` | `/projectt/helper/1.0.0` | Helper mode (address storage) |

### 4.3. Service Interaction

```
┌──────────────────────────────────────────────────────────────┐
│  Application (cmd/main.go)                                   │
│  └── app.App                                                 │
│      └── InitServices()                                      │
└──────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────────────┐
│  P2P Network (orchestrator)                                  │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  P2PCore {                                           │   │
│  │    host            host.Host                         │   │
│  │    dht             *dht.IpfsDHT                      │   │
│  │    discovery       *discovery.Service                │   │
│  │    connection      *connection.Manager               │   │
│  │    autodial        *autodial.Manager                 │   │
│  │    peerExchange    *peerexchange.Exchange            │   │
│  │    chat            *chat.Service                     │   │
│  │    transfer        *transfer.Service                 │   │
│  │    profileExchange *profile.ExchangeService          │   │
│  │    itemSync        *itemsync.Service                 │   │
│  │    avatar          *avatar.Service                   │   │
│  │    helper          *helper.Service                   │   │
│  │  }                                                     │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────────────┐
│  Domain Services                                             │
│  • ItemsService                                              │
│  • TagsService                                               │
│  • ContentBlocksService                                      │
│  • FavoritesService                                          │
│  • PinnedService                                             │
│  • ContactsService                                           │
│  • ChatService (event dispatcher)                            │
└──────────────────────────────────────────────────────────────┘
```

---

## 5. Layer 3: Data Access Layer

### 5.1. Database (SQLite)

**ORM:** Pure SQL without ORM (database/sql)

**DB Structure:**

```sql
-- Main tables
items               -- Element cards
tags                -- Tags
item_tags           -- Element-tag relationship (M:M)
content_blocks      -- Card content

-- Favorites and pinned
favorites           -- Favorite items
pinned_elements     -- Pinned items

-- P2P and chats
chat_messages       -- Chat messages
contacts            -- Contacts/P2P peers
profiles            -- User profiles
bootstrap_peers     -- P2P Bootstrap nodes

-- Local settings
local_profiles      -- User's local profile
sort_settings       -- Sort settings
```

### 5.2. Repository Pattern

**Implementation:**

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

// Implementation
func CreateItem(item *models.Item) error {
    query := `INSERT INTO items (...) VALUES (...)`
    result, err := db.Exec(query, ...)
    // ...
}
```

**Advantages:**
- ✅ Single point of access to the database
- ✅ Easy to test (mocking)
- ✅ Encapsulation of SQL queries
- ✅ Centralized error handling

### 5.3. File Storage

**Strategy:** Content-Addressable Storage (CAS)

```
storage/files/
├── a1b2c3d4e5f6...      # SHA-256 content hash
├── f6e5d4c3b2a1...
└── ...
```

**Save algorithm:**

```go
func SaveFile(content []byte, originalName string) (string, error) {
    // 1. Compute SHA-256 hash
    hash := sha256.Sum256(content)
    hashStr := hex.EncodeToString(hash[:])

    // 2. Check existence (deduplication)
    filePath := filepath.Join("storage/files", hashStr)
    if fileExists(filePath) {
        return hashStr, nil  // File already exists
    }

    // 3. Save the file
    os.WriteFile(filePath, content, 0644)

    return hashStr, nil
}
```

**Advantages:**
- ✅ Automatic deduplication
- ✅ Data integrity (hash verification)
- ✅ Fast lookup by hash
- ✅ No name collision possible

---

## 6. P2P Subsystem

### 6.1. libp2p Architecture

**Technology:** [libp2p](https://libp2p.io/)

```
┌─────────────────────────────────────────────────────────────┐
│  Application Layer                                          │
│  • Chat UI  • File Transfer UI  • Profile UI               │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  Protocol Layer (Custom Protocols)                          │
│  /projectt/chat/1.0.0       - Message exchange              │
│  /projectt/transfer/1.0.0   - File transfer                 │
│  /projectt/profile/1.0.0    - Profile exchange              │
│  /projectt/itemsync/1.0.0   - Item synchronization          │
│  /projectt/helper/1.0.0     - Helper mode                   │
│  /projectt/ping/1.0.0       - Keep-alive                    │
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
│  • TCP / QUIC              - Transport                      │
│  • DHT (Kademlia)          - Peer discovery                 │
│  • mDNS (zeroconf)         - Local network                  │
│  • Relay                   - NAT traversal                  │
│  • STUN                    - External address discovery     │
└─────────────────────────────────────────────────────────────┘
```

### 6.2. P2P Connection Lifecycle

```
1. Host initialization
   └─> libp2p.New(opts...)
       ├─> Ed25519 key generation
       ├─> PeerID creation
       └─> Start listener on port

2. Peer discovery
   ├─> DHT.Advertise()  - Advertise itself on the network
   ├─> DHT.FindPeers()  - Search for other peers
   ├─> mDNS (local)     - LAN discovery
   └─> Bootstrap        - Connect to known nodes

3. Connection
   └─> host.Connect(ctx, peerInfo)
       ├─> Handshake (Noise/TLS)
       ├─> Multiplexing (mplex/yamux)
       └─> Establish stream

4. Data exchange
   ├─> host.NewStream(peerID, protocolID)
   ├─> stream.Write(data)
   ├─> stream.Read(response)
   └─> stream.Close()

5. Monitoring
   ├─> Keep-alive ping every 30 sec
   ├─> After 3 failures - mark as offline
   └─> Auto-reconnect (up to 5 attempts)
```

### 6.3. Chat Message Format

```go
type Message struct {
    ID          int64       `json:"id"`
    FromPeerID  string      `json:"from_peer_id"`
    Content     string      `json:"content"`
    ContentType string      `json:"content_type"`
    Metadata    string      `json:"metadata"`
    Timestamp   int64       `json:"timestamp"`
    MessageType MessageType `json:"message_type"`
    Signature   []byte      `json:"signature"`      // Ed25519 signature
    Encrypted   bool        `json:"encrypted"`      // Encryption flag
    Nonce       []byte      `json:"nonce"`          // Nonce for encryption
}
```

**Send process:**

```
1. Create message
   └─> Message{FromPeerID, Content, Timestamp, ...}

2. Sign
   └─> data = fmt.Sprintf("%s:%s:%d", FromPeerID, Content, Timestamp)
   └─> signature = privKey.Sign(data)

3. Encrypt (optional)
   └─> encrypted = XOR(content, encryptionKey, nonce)

4. Serialize
   └─> json.Marshal(msg)

5. Send
   └─> stream.Write(data)

6. Receive ACK
   └─> stream.Read(ack)  // 0x01 = success
```

---

## 7. Data Model

### 7.1. Core Models

```go
// internal/storage/database/models/item.go
type Item struct {
    ID          int       `json:"id"`
    ElementUUID string    `json:"element_uuid"`  // Unique UUID
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Type        string    `json:"type"`          // element, folder, link
    ContentMeta string    `json:"content_meta"`  // JSON of content blocks
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

### 7.2. Table Relationships

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

## 8. Technology Stack

### 8.1. Core Technologies

| Category | Technology | Version | Purpose |
|-----------|------------|--------|------------|
| **Programming Language** | Go | 1.26 | Main development language |
| **UI Framework** | Fyne | v2.4.4 | Cross-platform GUI |
| **Database** | SQLite | 3.x | Local metadata storage |
| **SQLite Driver** | mattn/go-sqlite3 | v1.14.32 | CGO SQLite driver |
| **P2P Library** | libp2p | v0.32.0 | Network communication |
| **DHT** | go-libp2p-kad-dht | v0.25.0 | Distributed Hash Table |
| **PubSub** | go-libp2p-pubsub | v0.10.0 | Broadcast messaging |

### 8.2. Dependencies (go.mod)

```go
require (
    // UI
    fyne.io/fyne/v2 v2.4.4

    // Database
    github.com/mattn/go-sqlite3 v1.14.32

    // P2P
    github.com/libp2p/go-libp2p v0.32.0
    github.com/libp2p/go-libp2p-kad-dht v0.25.0
    github.com/libp2p/go-libp2p-pubsub v0.10.0
    github.com/multiformats/go-multiaddr v0.12.0

    // Cryptography
    golang.org/x/crypto v0.48.0

    // Utilities
    gopkg.in/yaml.v3 v3.0.1             // YAML configs
    github.com/stretchr/testify v1.10.0  // Testing
    golang.org/x/text v0.34.0            // Localization
)
```

**Total dependencies:** ~130 (direct + transitive)

### 8.3. Development Tools

| Tool | Purpose |
|------------|------------|
| **Go Modules** | Dependency management |
| **Makefile / make.ps1** | Build automation (cross-platform) |
| **pre-commit** | Code formatting hooks |
| **GitHub Actions** | CI/CD pipelines |
| **VS Code** | Main IDE |
| **go test** | Testing |
| **go fmt** | Code formatting |
| **go vet** | Static analysis |

### 8.4. Compiled Artifacts

| File | Purpose |
|------|------------|
| `projectT.exe` | Main application executable |
| `clear_chats.exe` | Chat cleanup utility |
| `autodial.test.exe` | P2P auto-dial subsystem tests |
| `peerexchange.test.exe` | P2P peer exchange subsystem tests |

---

## 9. Design Patterns

### 9.1. Applied Patterns

| Pattern | Where Used | Advantages |
|---------|------------------|--------------|
| **Singleton** | `services.GetChatService()` | Global access to services |
| **Repository** | `internal/storage/database/queries/` | Data access abstraction |
| **Observer** | `ChatService.Subscribe()` | Event-driven UI updates |
| **Factory** | `p2p.NewChatService()` | P2P service creation |
| **Strategy** | Various card types | Flexible content rendering |
| **Decorator** | HoverPreview for cards | Extended functionality |
| **Command** | UI button handlers | Action encapsulation |

### 9.2. Example: Observer Pattern

```go
// Subject
type ChatService struct {
    subscribers []chan *ChatMessageEvent
}

// Subscribe method
func (cs *ChatService) Subscribe() <-chan *ChatMessageEvent {
    ch := make(chan *ChatMessageEvent, 10)
    cs.subscribers = append(cs.subscribers, ch)
    return ch
}

// Notification method
func (cs *ChatService) NotifyNewMessage(...) {
    for _, sub := range cs.subscribers {
        select {
        case sub <- event:
            // Success
        default:
            // Channel overflow
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

## 10. Data Flow

### 10.1. Creating an Item

```
1. User creates a card
   └─> UI: editor.go → onSave()

2. Data validation
   └─> services.ContentBlocksService.Validate()

3. Save content
   ├─> Files → storage/files/ (SHA-256 hash)
   └─> Metadata → SQLite (items table)

4. Update UI
   └─> elements.Refresh() → grid.Reload()
```

### 10.2. Sending a P2P Message

```
1. User types a message
   └─> UI: center_panel.go → sendMessage()

2. Check connection
   └─> p2p.ChatService.SendMessage(ctx, peerID, content)

3. If peer is online:
   ├─> Create Message with signature
   ├─> Encrypt (optional)
   ├─> Send via stream.Write()
   ├─> Receive ACK
   └─> Save to DB (outgoing)

4. If peer is offline:
   └─> Add to queue (queueMessage)

5. Update UI
   └─> chatPanel.AddMessage(message, isOutgoing)
```

### 10.3. Receiving a P2P Message

```
1. Receive stream
   └─> p2p.ChatService.HandleChatStream(stream)

2. Read and deserialize
   └─> json.Unmarshal(data, &Message)

3. Verify signature
   └─> pubKey.Verify(data, signature)

4. Decrypt (if encrypted)
   └─> XOR decrypt with encryptionKey

5. Save to DB
   └─> queries.CreateChatMessage(message)

6. Notify UI
   └─> services.ChatService.NotifyNewMessage()
       └─> eventChannel ← ChatMessageEvent
           └─> UI.handleMessageEvents()
               └─> chatPanel.AddMessage(message, isIncoming)
```

---

## 11. Security

### 11.1. Cryptography

| Component | Algorithm | Purpose |
|-----------|----------|------------|
| **Access Keys** | Ed25519 (32 bytes) | Peer identification |
| **Message Signing** | Ed25519 Sign/Verify | Sender verification |
| **Message Encryption** | XOR + nonce | Confidentiality |
| **File Hashing** | SHA-256 | Integrity and deduplication |
| **Key Protection** | Master password (AES-256) | Private key encryption |

### 11.2. Threat Model

| Threat | Protection Measure |
|--------|-------------|
| **Peer spoofing** | Ed25519 profile signatures |
| **Message interception** | Symmetric key encryption |
| **Replay attack** | Timestamp + nonce in signature |
| **DoS attack** | Connection limits, timeouts |
| **Key leakage** | Master password encryption |
| **Unwanted peers** | Contact blacklist |

### 11.3. Key Storage

```
┌─────────────────────────────────────────────────────────┐
│  Keys stored in DB (local_profiles)                      │
│                                                          │
│  • public_key  - public key (unencrypted)                │
│  • private_key - private key (AES-256 encrypted)         │
│                                                          │
│  Decryption:                                            │
│  1. User enters master password                          │
│  2. Key derived from password (PBKDF2)                   │
│  3. private_key decrypted with derived key               │
│  4. Key kept in memory until application closes          │
└─────────────────────────────────────────────────────────┘
```

---

## 12. Performance

### 12.1. Optimizations

| Area | Optimization | Effect |
|---------|-------------|--------|
| **DB** | Indexes on foreign keys | Fast JOIN queries |
| **DB** | Batch message insertion | Reduced I/O operations |
| **Files** | Hash-based deduplication | Disk space savings |
| **UI** | Lazy card loading | Fast application startup |
| **UI** | Preview caching | Instant hover effect |
| **P2P** | Keep-alive ping | Fast offline detection |
| **P2P** | Message queue | Delivery guarantee |

### 12.2. Performance Metrics

| Metric | Value | Notes |
|---------|----------|------------|
| **Application startup** | < 2 sec | Depends on DB size |
| **Grid load (100 items)** | < 500 ms | With caching |
| **Message send** | < 100 ms | Local |
| **P2P connection** | 1-5 sec | Depends on network |
| **File transfer (1 MB)** | 2-10 sec | Depends on bandwidth |
| **Tag search** | < 100 ms | With indexes |

### 12.3. Profiling

**Tools:**
- `go test -bench=` - Benchmarks
- `go tool pprof` - CPU/Memory profiling
- `go test -race` - Race condition detection

**Benchmark example:**

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

## 📎 Appendices

### A. Sequence Diagram: P2P Chat

```
User A                  P2P Network          User B
    │                       │                       │
    │── Type message ─────> │                       │
    │                       │                       │
    │                       │── Create Message ───> │
    │                       │   Sign & Encrypt      │
    │                       │                       │
    │                       │── stream.Write() ───> │
    │                       │                       │── Receive stream ──>
    │                       │                       │── Verify Signature ──>
    │                       │                       │── Decrypt ──>
    │                       │                       │── Save to DB ──>
    │                       │                       │
    │                       │<── stream.Write(ACK) ─│
    │                       │                       │
    │<── Update UI ──────── │                       │
    │                       │                       │── Update UI ──>
```

### B. Go Packages Used

```
internal/
├── app/              # Application initialization
├── config/           # Configuration
├── services/         # Business logic
│   ├── favorites/    # Favorites
│   ├── pinned/       # Pinned items
│   └── p2p/          # P2P services
│       ├── chat/     # Chat
│       ├── network/  # Network
│       ├── profile/  # Profiles
│       ├── itemsync/ # Synchronization
│       └── transfer/ # File transfer
├── storage/
│   └── database/
│       ├── migrations/  # Migrations
│       ├── models/      # Data models
│       └── queries/     # SQL queries
└── ui/
    ├── workspace/    # Workspace
    ├── sidebar/      # Sidebar
    ├── header/       # Header bar
    └── cards/        # Cards
```

---

<div align="center">

**ProjectT — Diploma Project Architecture**

*Documentation is current as of the defense date*

Made with ❤️ by Egor Redoran

</div>
