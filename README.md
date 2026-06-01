<div align="center">

# ProjectT

### 🗂️ A Hybrid of File Explorer and Pinterest on Steroids

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Version](https://img.shields.io/badge/Version-0.2.0--dev-blue?style=flat-square)](CHANGELOG.md)
[![Fyne](https://img.shields.io/badge/UI-Fyne-blue?style=flat-square&logo=go&logoColor=white)](https://fyne.io/)
[![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://www.sqlite.org/)
[![libp2p](https://img.shields.io/badge/P2P-libp2p-ff69b4?style=flat-square&logo=go&logoColor=white)](https://libp2p.io/)

**This is a hybrid of a File Explorer and Pinterest, where objects live as semantic units rather than scattered files, with P2P sharing to share collections without compromising privacy.**

<img src="assets/screenshots/ProjctT_true.png" alt="Logo" width="30%">

</div>

---

## 📊 Technologies

- **UI** - Fyne with custom theme, card widgets, adaptive grid, breadcrumbs navigation
- **Business Logic** - Services for managing items, tags, favorites, pinned items, group chats
- **Storage** - SQLite for metadata + file system for content
- **P2P** - libp2p for decentralized exchange (DHT, relay, STUN, NAT traversal, pubsub)
- **Cryptography** - Ed25519 for signing, SHA-256 for hashing, XOR encryption
- **Monitoring** - Prometheus + Grafana for metrics and observability

---

## 📖 About the Project

**ProjectT** is an attempt to combine the convenience of a file explorer with the visuality of Pinterest, but without their limitations.

In a file explorer, it's tedious to store related things:
an image sits in one place, its description in a `.txt` file, and a link — somewhere else entirely.

In Pinterest, you can't add poetry or a document — only images.

Here, **an object is a whole**.
One card can contain:
- an image
- text (poetry, description)
- a link
- any file

And all of this is tied together with tags, not scattered across folders.

---

## 🟢 Current State (v0.2.0-dev)

> **Development build.** Significant progress since v0.1.0, but not yet stable for release.

**Already working:**
- ✅ Cards, tags, folders, search, breadcrumbs navigation
- ✅ Private/public item visibility control
- ✅ Remote items (preview from other peers)
- ✅ P2P 1-on-1 chat with encryption
- ✅ Group chats and channels (pubsub-based)
- ✅ File transfer (single + batch)
- ✅ Profile exchange and sync
- ✅ Peer discovery (DHT, mDNS, bootstrap)
- ✅ Cross-network connections (NAT traversal helpers, public IP detection)
- ✅ Prometheus metrics + Grafana dashboards

**Known issues:**
- 🔧 Group chats not yet integrated with UI
- 🔧 Cross-network connections work but have bugs
- 🔧 Tab transitions are slow
- 🔧 No system notifications
- 🔧 Batch transfer needs testing

**Plans:** UI integration for group chats → connection bug fixes → stabilization → feed → global search

- [Detailed Roadmap](ROADMAP.md)

---

## 📸 Screenshots

![Main Screen](assets/screenshots/scrin1.png)
![Item Editor](assets/screenshots/scrin2.png)
![Chat](assets/screenshots/scrin3.png)

---

## ✨ Features

- **Card Containers** — unified format for files, links, text, and images
  (audio and video in development)
- **Smart Tags** — folder-free grouping: colored, with autocomplete
- **Item Visibility** — private (not shared) or public (available to peers)
- **Remote Items** — preview items from other peers without saving
- **Local Storage** — everything lives on your machine, no clouds or subscriptions
- **Pinterest-style Grid** — adaptive, with resizable cards
- **Breadcrumbs Navigation** — hierarchical folder navigation
- **Search & Sorting** — by tags, title, date, type
- **Favorites & Pinned** — for quick access

**P2P Functionality:**
- **Peer-to-Peer Chat** — text messages with Ed25519 encryption
- **Group Chats** — pubsub-based mesh groups with roles (creator, admin, member)
- **Channels** — one-way broadcasting (only admins publish)
- **File Transfer** — single files and batch transfers (folders, pinned, selections)
- **Profile Exchange** — automatic synchronization on connection
- **Peer Discovery** — DHT, mDNS, bootstrap nodes
- **NAT Traversal** — public IP detection, relay support, multi-address format
- **Security** — master password, cryptographic keys, blacklist
- **Offline Mode** — message queue for unavailable peers

**Monitoring:**
- **Prometheus Metrics** — items, tags, chat, P2P, database, runtime metrics
- **Grafana Dashboards** — visual monitoring with docker-compose setup

---

## ⛓️ Running

Download the latest [release](https://github.com/Drekard/projectT/releases) and run `projectT.exe`.

---

## ❓ FAQ

**How to connect two devices?**
Currently, both devices must be connected to the same WiFi network for the most reliable connection.
Cross-network connections are supported but may have issues depending on NAT configuration.
To connect, go to the P2P tab, copy your address, and transfer it to the second device.
The new compact address format (`projectt:peerid@ip:port`) supports multiple endpoints for better connectivity.
Now you can start chatting between devices, exchange profiles, and share your items via chat.

**How to create a group chat?**
Group chats are implemented in the backend but not yet fully integrated into the UI.
They use libp2p pubsub for message distribution and support:
- Invite tokens with depth limits (chain invitations)
- Roles: creator, admin, member, subscriber
- Message history synchronization on join
- Admin actions: kick, ban, change roles

**What are channels?**
Channels are a special type of group chat where only admins/creators can publish messages.
Subscribers can read but not write. Useful for broadcasting curated content.

**What is item visibility?**
Each item has a visibility setting:
- **Public** — can be shared with other peers during transfer
- **Private** — never sent to other peers, stays local

**What are remote items?**
When another peer shares items with you, they appear as "remote" items in preview mode.
You can browse them without saving to your collection. Saved items become local copies.

**Where is data stored?**
- Metadata: `projectT.db`
- Files: `storage/files/`

**How is content structured inside a card?**
Each item is a card with content blocks.
Blocks are serialized to JSON and saved in the `content_meta` field.
Example:
```json
{
  "type": "image|file|link|text",
  "content": "text or link",
  "file_hash": "sha256-file-hash",
  "original_name": "file name",
  "extension": "extension"
}
```

**How is file integrity ensured?**
When saving a file:
- SHA-256 hash of the content is computed
- The file is saved with the hash as its name
- When reading, the hash is verified against the file name
This eliminates duplicates and speeds up search

**How does P2P chat work?**
ProjectT uses libp2p for direct connection between users.
Peer discovery — via DHT (globally) and mDNS (locally).
Messages are encrypted and signed with cryptographic keys.
Group chats use libp2p pubsub for message distribution.
More details — in [FEATURES.md](FEATURES.md#-p2p-chat-and-exchange).

**How does monitoring work?**
ProjectT can export Prometheus metrics on a configurable port.
Use `docker-compose.monitoring.yml` to start Prometheus + Grafana stack.
More details — in [ARCHITECTURE.md](ARCHITECTURE.md#13-monitoring--observability).

---

## 📚 **Learn More:**

- [Full Feature List](FEATURES.md)
- [Project Architecture](ARCHITECTURE.md)
- [This Version's Capabilities](CHANGELOG.md)
- [Development Roadmap](ROADMAP.md)

---

## 👨‍💻 Author

**ProjectT** is my personal project that I want to bring to the level of a full-fledged and independent network

Always looking for a job. Strong in both backend and layout

Have ideas, suggestions, or just a question? Feel free to reach out via my personal messages

Telegram - @Redoranar

If you're interested, I also have a Pinterest at https://ru.pinterest.com/egors3206/

---

<div align="center">

**This is not a file explorer replacement. It's a new decentralized space for those who collect, store, and get inspired.**
</div>
