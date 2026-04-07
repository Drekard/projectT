# 📖 ProjectT Features

**Last updated:** March 2026 (updated)

**ProjectT** is a hybrid application combining a file manager, a visual board (like Pinterest), and a P2P messenger for sharing collections.

---

## 🎯 Main Purpose

ProjectT solves the problem of data fragmentation:
- In a regular file explorer, files, links, text, and images are stored separately
- In Pinterest, you can't add a document or arbitrary file
- In messengers, it's difficult to organize long-term storage with tags and search

**ProjectT** combines everything into unified card containers with a flexible tag system and P2P sharing capabilities.

---

## ✨ Key Features

### 📦 1. Card Containers (Items)

Each card is a self-contained object that can include:

| Content Type | Description |
|--------------|----------|
| 📝 **Text** | Notes, descriptions, poetry, code |
| 🖼️ **Images** | Pictures with thumbnails and preview |
| 📎 **Files** | Any files: documents, archives, executables |
| 🔗 **Links** | URLs with auto-generated preview |
| 🎬 **Media** | Audio and video files (in development) |

**Advantages:**
- Everything linked in one card — no need to search for files across folders
- Cards have types: `element`, `folder`, `link`
- Drag-and-drop support for adding files
- Auto-save on edit

---

### 🏷️ 2. Tag System

Tags replace folders and provide flexible grouping:

| Feature | Description |
|-------------|----------|
| 🎨 **Colored Tags** | Visual category distinction |
| 🔄 **Autocomplete** | Smart search of existing tags on input |
| 📊 **Filtering** | Display items by one or multiple tags |
| 🔍 **Tag Search** | Quick search via sidebar |
| ⭐ **Favorite Tags** | Pin frequently used tags |

**Example usage:**
```
A "Diploma Project" card might have tags:
- 📚 Study (blue)
- 💻 Programming (green)
- ⏰ Urgent (red)
```

---

### 📚 3. Collections and Folders

Space organization:

| Function | Description |
|---------|----------|
| 📁 **Folders** | Hierarchical structure for grouping cards |
| 📌 **Pinned Items** | Quick access to important cards |
| ⭐ **Favorites** | Separate list of beloved items |
| 🎯 **Item Showcase** | Visual display of pinned items in profile |

---

### 🔍 4. Search and Filtering

Powerful tools for finding what you need:

| Search Type | Description |
|------------|----------|
| 🔎 **Search by Title** | Search by card title |
| 🏷️ **Search by Tags** | Filter by one or multiple tags |
| 📅 **Search by Date** | Sort by creation/modification date |
| 📝 **Search by Type** | Filter by item type (file, link, text) |
| 🎨 **Search by Tag Color** | Visual search |

**Display Settings:**
- Sorting: by date, title, type
- Order: ascending/descending
- Grid size: adaptive with density adjustment

---

### 🖼️ 5. Visual Grid (Pinterest-style)

Content display:

| Feature | Description |
|-------------|----------|
| 📱 **Adaptive Grid** | Automatic adjustment to window size |
| 🎚️ **Size Adjustment** | Change card density |
| 🎨 **Custom Cards** | Different design for different content types |
| 👁️ **Preview** | Hover effect with quick view |
| 🖱️ **Context Menu** | Right-click for card actions |

---

### 💬 6. P2P Chat and Exchange

**A full-featured P2P messenger for exchanging messages and files:**

#### 🔗 Connection

| Method | Description |
|--------|----------|
| 📍 **Direct Connection** | By peer address (`projectt:peerid@/ip4/...`) |
| 🌐 **DHT Discovery** | Global peer search via internet |
| 🏠 **mDNS Discovery** | Local network (automatic discovery) |
| 🤝 **Bootstrap Peers** | Static nodes for initial connection |
| 🔄 **Relay via Intermediary** | NAT traversal through intermediate nodes |
| 📡 **STUN Client** | External address determination for connection |

#### 💬 Messaging

| Feature | Description |
|---------|----------|
| ✉️ **Text Messages** | Instant sending with encryption |
| 📎 **File Transfer** | Send files of any size |
| 🖼️ **Image Sending** | With preview in chat |
| 📤 **Avatar Transfer** | Profile image exchange |
| ⏸️ **Offline Mode** | Messages queued and sent on connection |
| ✅ **Delivery Confirmation** | Guaranteed message receipt |
| 🔒 **Encryption** | XOR with symmetric key + Ed25519 signature |

#### 👤 Profiles

| Feature | Description |
|-------------|----------|
| 🆔 **Unique PeerID** | Identifier based on cryptographic keys |
| 🎭 **Profile Avatar** | Image displayed in chat |
| 📝 **Characteristics** | Arbitrary profile fields (context, description) |
| 🔄 **Auto Profile Exchange** | Peers exchange profiles on connection |
| 🔐 **Profile Signature** | Cryptographic verification of authenticity |

#### 📦 Item Synchronization

| Feature | Description |
|---------|----------|
| 🔄 **Item Exchange** | Transfer cards between peers |
| 📤 **Collection Export** | Send item selections |
| 📥 **Item Import** | Receive items from other users |

---

### 🔐 7. Security and Privacy

| Measure | Description |
|------|----------|
| 🔑 **Cryptographic Keys** | Ed25519 for signing messages and profiles |
| 🔒 **Message Encryption** | Symmetric encryption with nonce |
| 🔐 **Master Password** | Private key encryption |
| ✅ **Signature Verification** | Sender authenticity check |
| 🚫 **Blacklist** | Block unwanted peers |
| 📍 **Address Control** | Manual contact addition |

---

### 🗄️ 8. Data Storage

| Component | Description |
|-----------|----------|
| 💾 **SQLite** | Metadata: items, tags, relationships, chats, contacts |
| 📁 **File System** | Files stored in `storage/files/` |
| 🔐 **SHA-256 Hashing** | Files saved under hash name |
| ♻️ **Deduplication** | Identical files are not duplicated |
| 📊 **Integrity** | Hash verification on file read |

**Database structure:**
- `items` — items (cards)
- `tags` — tags
- `item_tags` — item-tag relationships
- `favorites` — favorite items
- `pinned_elements` — pinned items
- `chat_messages` — chat messages
- `contacts` — contacts/P2P peers
- `profiles` — user profiles
- `bootstrap_peers` — bootstrap nodes for P2P

---

### 🎨 9. Interface and UX

| Component | Description |
|-----------|----------|
| 🪟 **Window Interface** | Fyne GUI with native Windows/Linux/macOS support |
| 🎨 **Dark Theme** | Stylish design with custom colors |
| 📱 **Responsiveness** | Window size adaptation |
| 🖱️ **Context Menus** | Right-click for actions |
| ⌨️ **Hotkeys** | Quick actions (in development) |
| 📊 **Filter Panel** | Sidebar with tags and search |
| 💬 **Chat Panel** | Three-panel chat interface |

---

## 🛠️ Technical Details

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    UI Layer (Fyne)                      │
│  ┌───────────┐  ┌───────────┐  ┌─────────────────────┐ │
│  │ Sidebar   │  │ Workspace │  │ Header/Filters      │ │
│  │ - Tags    │  │ - Grid    │  │ - Search            │ │
│  │ - Fav.    │  │ - Chats   │  │ - Sorting           │ │
│  └───────────┘  └───────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────┐
│                 Business Logic Layer                    │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌─────────────────┐  │
│  │ Items  │ │ Tags   │ │ Chat   │ │ P2P Network     │  │
│  │ Service│ │ Service│ │ Service│ │ - Discovery     │  │
│  └────────┘ └────────┘ └────────┘ │ - Chat          │  │
│                                    │ - Transfer      │  │
│                                    │ - Profile       │  │
│                                    └─────────────────┘  │
└─────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────┐
│                  Data Access Layer                      │
│  ┌──────────────────┐  ┌────────────────────────────┐  │
│  │ SQLite (queries) │  │ File System (storage/)     │  │
│  └──────────────────┘  └────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Technology Stack

| Component | Technology |
|-----------|------------|
| **Language** | Go 1.26 |
| **UI** | Fyne v2.4.4 |
| **Database** | SQLite (mattn/go-sqlite3 v1.14.32) |
| **P2P** | libp2p v0.32.0 (DHT, pubsub, discovery) |
| **Cryptography** | Ed25519, SHA-256 (golang.org/x/crypto v0.48.0) |
| **Configuration** | YAML (gopkg.in/yaml.v3 v3.0.1) |
| **Testing** | testify v1.10.0 |

---

## 📊 Comparison with Alternatives

| Feature | ProjectT | File Explorer | Pinterest | Messengers |
|---------|----------|-----------|-----------|-------------|
| Card Containers | ✅ | ❌ | ❌ | ❌ |
| Tags Instead of Folders | ✅ | ❌ | ⚠️ | ❌ |
| P2P Sharing | ✅ | ❌ | ❌ | ⚠️ |
| Local Storage | ✅ | ✅ | ❌ | ⚠️ |
| Visual Grid | ✅ | ❌ | ✅ | ❌ |
| File Transfer | ✅ | ⚠️ | ❌ | ✅ |
| Privacy | ✅ | ✅ | ❌ | ⚠️ |

---

## 🚀 Use Cases

### 📚 For Students
- Lecture notes with subject tags
- Literature link collections
- Material sharing with classmates via P2P

### 🎨 For Designers
- Visual inspiration board
- Reference collections with tags
- Quick sharing of selections with colleagues

### 🔬 For Researchers
- Organization of scientific papers
- Linking data with publications
- Private data exchange with colleagues

### 💼 For Project Teams
- Centralized project resource storage
- Quick file sharing without clouds
- Chat with history and search

---

## 🔮 Future Plans

### In Development
- [ ] Audio/video cards with player
- [ ] Cross-device synchronization
- [ ] Group chats (mesh network)
- [ ] Advanced encryption (AES-256)
- [ ] Plugins for extensibility

### Planned
- [ ] Mobile version (iOS/Android)
- [ ] Web interface for browser access
- [ ] Cloud storage integration
- [ ] Automatic backups

---

## 📞 Contact

**Telegram:** [@Redoranar](https://t.me/Redoranar)

**Pinterest:** [ru.pinterest.com/egors3206](https://ru.pinterest.com/egors3206/)

**GitHub:** [github.com/Redoranar/projectT](https://github.com/Redoranar/projectT)

---

<div align="center">

**ProjectT — not just a file manager. A new space for your ideas.**

Made with ❤️ by Egor Redoran

</div>
