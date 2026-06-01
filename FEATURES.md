# 📖 ProjectT Features

**Last updated:** May 2026

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
- Cards have types: `element`, `folder`
- Drag-and-drop support for adding files
- Auto-save on edit
- **Visibility control**: public (shareable) or private (local only)
- **Status tracking**: saved, preview (from peer), archived
- **Remote items**: browse peer's items without saving
- **Additional text** displayed directly on cards

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
| 🧭 **Breadcrumbs** | Navigation through folder hierarchy |
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
| 📍 **Direct Connection** | By peer address (`projectt:peerid@ip:port`) |
| 🌐 **DHT Discovery** | Global peer search via internet |
| 🏠 **mDNS Discovery** | Local network (automatic discovery) |
| 🤝 **Bootstrap Peers** | Static nodes for initial connection |
| 🔄 **Relay via Intermediary** | NAT traversal through intermediate nodes |
| 📡 **STUN Client** | External address determination for connection |
| 🌍 **Public IP Detection** | Automatic discovery of external address |
| 🔗 **Multi-address Format** | Combines LAN + public addresses in one string |

#### 💬 Direct Messaging

| Feature | Description |
|---------|----------|
| ✉️ **Text Messages** | Instant sending with encryption |
| 📎 **File Transfer** | Send files of any size |
| 🖼️ **Image Sending** | With preview in chat |
| 📤 **Avatar Transfer** | Profile image exchange |
| ⏸️ **Offline Mode** | Messages queued and sent on connection |
| ✅ **Delivery Confirmation** | Guaranteed message receipt |
| 🔒 **Encryption** | XOR with symmetric key + Ed25519 signature |

#### 👥 Group Chats

| Feature | Description |
|---------|----------|
| 🏠 **Create Groups** | Name, description, type (group/channel) |
| 📨 **Invite Tokens** | Chain invitations with depth limits |
| 👑 **Roles** | Creator, admin, member, subscriber |
| 💬 **Pubsub Messaging** | Real-time message distribution via libp2p pubsub |
| 🔄 **History Sync** | Lamport clock-based message synchronization |
| 🛡️ **Membership Proofs** | Cryptographic verification of membership |
| 👢 **Admin Actions** | Kick, ban, change roles with signed proofs |
| 📢 **Channels** | One-way broadcasting (admins only publish) |

#### 👤 Profiles

| Feature | Description |
|-------------|----------|
| 🆔 **Unique PeerID** | Identifier based on cryptographic keys |
| 🎭 **Profile Avatar** | Image displayed in chat |
| 📝 **Characteristics** | Arbitrary profile fields (context, description) |
| 🔄 **Auto Profile Exchange** | Peers exchange profiles on connection |
| 🔐 **Profile Signature** | Cryptographic verification of authenticity |
| 🌐 **Profile Sync** | Synchronize profile updates across peers |
| 🎯 **Remote Profile View** | View other peers' profiles with item showcase |

#### 📦 Item Synchronization

| Feature | Description |
|---------|----------|
| 🔄 **Item Exchange** | Transfer cards between peers |
| 📦 **Batch Transfer** | Send multiple items at once |
| 📁 **Folder Transfer** | Send entire folder contents |
| 📌 **Pinned Transfer** | Share your pinned items |
| 🎯 **Selection Transfer** | Send custom item selections |
| 📥 **Item Import** | Receive items from other users |
| 👁️ **Preview Mode** | Browse remote items without saving |

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
| 🏷️ **Item Visibility** | Public/private control per item |
| 🛡️ **Membership Proofs** | Cryptographic group membership verification |

---

### 🗄️ 8. Data Storage

| Component | Description |
|-----------|----------|
| 💾 **SQLite** | Metadata: items, tags, relationships, chats, contacts, group chats |
| 📁 **File System** | Files stored in `storage/files/` |
| 🔐 **SHA-256 Hashing** | Files saved under hash name |
| ♻️ **Deduplication** | Identical files are not duplicated |
| 📊 **Integrity** | Hash verification on file read |

**Database structure:**
- `items` — items (cards) with visibility and status
- `tags` — tags
- `item_tags` — item-tag relationships
- `favorites` — favorite items
- `pinned_elements` — pinned items
- `chat_messages` — chat messages
- `contacts` — contacts/P2P peers
- `profiles` — user profiles
- `bootstrap_peers` — bootstrap nodes for P2P
- `group_chats` — group chats and channels
- `group_members` — group membership with roles
- `group_messages` — group chat messages
- `group_invitations` — invite tokens with depth limits
- `group_blocks` — banned members
- `group_membership_proofs` — cryptographic membership proofs
- `local_profiles` — local user profile
- `sort_settings` — sorting preferences

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
| 🧭 **Breadcrumbs** | Hierarchical folder navigation |
| 📋 **Chat Sidebar** | Quick access to chats from sidebar |
| 🔍 **Search Popup** | Quick search overlay |
| 📦 **Batch Progress** | Visual progress for multi-item transfers |

---

### 📊 10. Monitoring and Observability

| Component | Description |
|-----------|----------|
| 📈 **Prometheus Metrics** | Export metrics on configurable port |
| 📊 **Grafana Dashboards** | Pre-configured visualizations |
| 🔧 **Docker Compose** | One-command monitoring stack |

**Available metrics:**
- Items: total count, created, deleted
- Tags: total count
- Chat: messages, contacts, active contacts
- P2P: peers, connections, transferred bytes, files
- Database: query duration, errors
- Runtime: goroutines, memory, GC pauses

---

## 🛠️ Technical Details

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    UI Layer (Fyne)                      │
│  ┌───────────┐  ┌───────────┐  ┌─────────────────────┐ │
│  │ Sidebar   │  │ Workspace │  │ Header/Filters      │ │
│  │ - Tags    │  │ - Grid    │  │ - Search            │ │
│  │ - Chats   │  │ - Chats   │  │ - Sorting           │ │
│  │ - Nav     │  │ - Profile │  │ - Breadcrumbs       │ │
│  └───────────┘  └───────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────┐
│                 Business Logic Layer                    │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌─────────────────┐  │
│  │ Items  │ │ Tags   │ │ Chat   │ │ P2P Network     │  │
│  │ Service│ │ Service│ │ Service│ │ - Discovery     │  │
│  └────────┘ └────────┘ └────────┘ │ - Chat          │  │
│  ┌────────┐ ┌────────┐            │ - Group Chat    │  │
│  │ Fav.   │ │ Pinned │            │ - Transfer      │  │
│  └────────┘ └────────┘            │ - Profile       │  │
│                                    │ - NAT Helpers   │  │
│                                    └─────────────────┘  │
└─────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────┐
│                  Data Access Layer                      │
│  ┌──────────────────┐  ┌────────────────────────────┐  │
│  │ SQLite (queries) │  │ File System (storage/)     │  │
│  └──────────────────┘  └────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────┐
│              Monitoring Layer                           │
│  ┌──────────────────┐  ┌────────────────────────────┐  │
│  │ Prometheus       │  │ Grafana Dashboards         │  │
│  │ (:9090/metrics)  │  │ (:3000)                    │  │
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
| **Monitoring** | Prometheus client_golang |

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
| Group Chats | ✅ | ❌ | ❌ | ✅ |
| Channels | ✅ | ❌ | ❌ | ⚠️ |
| Monitoring | ✅ | ❌ | ❌ | ❌ |

---

## 🚀 Use Cases

### 📚 For Students
- Lecture notes with subject tags
- Literature link collections
- Material sharing with classmates via P2P
- Study groups for collaboration

### 🎨 For Designers
- Visual inspiration board
- Reference collections with tags
- Quick sharing of selections with colleagues

### 🔬 For Researchers
- Organization of scientific papers
- Linking data with publications
- Private data exchange with colleagues
- Research groups with shared resources

### 💼 For Project Teams
- Centralized project resource storage
- Quick file sharing without clouds
- Chat with history and search
- Channels for announcements

---

## 🔮 Future Plans

### In Development
- [ ] Audio/video cards with player
- [ ] Cross-device synchronization
- [ ] Advanced encryption (AES-256)
- [ ] Plugins for extensibility
- [ ] UI integration for group chats
- [ ] Connection stability improvements

### Planned
- [ ] Mobile version (iOS/Android)
- [ ] Web interface for browser access
- [ ] Cloud storage integration
- [ ] Automatic backups
- [ ] Feed (unified content stream)
- [ ] Global network search

---

## 📞 Contact

**Telegram:** [@Redoranar](https://t.me/Redoranar)

**Pinterest:** [ru.pinterest.com/egors3206](https://ru.pinterest.com/egors3206/)

**GitHub:** [github.com/Drekard/projectT](https://github.com/Drekard/projectT)

---

<div align="center">

**ProjectT — not just a file manager. A new space for your ideas.**

Made with ❤️ by Egor Redoran

</div>
