# Changelog

All notable changes to ProjectT.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.1.0] — 2026-04-04

Initial public release of ProjectT — a hybrid of a file manager, visual board, and P2P messenger.

### Added

**Application Core:**
- Card containers with support for text, files, images, and links
- Tag system with color coding, autocomplete, and filtering
- Favorite and pinned items for quick access
- Pinterest-style visual grid with adaptive card sizing
- Card editor with drag-and-drop file support
- Search and sorting by title, tags, date, type
- Dark theme with custom styling

**P2P Subsystem:**
- Decentralized network based on libp2p (DHT, mDNS, relay, STUN)
- P2P chat with text messages and Ed25519 encryption
- Direct file transfer between peers
- Profile exchange with avatars on connection
- Peer discovery via DHT (globally) and mDNS (locally)
- Auto-connect to known peers on startup
- Offline message queue
- Cryptographic message signing (Ed25519)
- Master password for private key protection

**Data Storage:**
- SQLite for metadata (items, tags, chats, contacts, profiles)
- File storage with Content-Addressable Storage (SHA-256)
- Automatic file deduplication
- YAML configuration with environment variable support

**Interface:**
- Three-panel chat interface (contacts, chat, profile)
- Sidebar with tags, favorites, and settings
- Context menus for cards and items
- Hover preview for cards
- P2P settings panel with connection management

### Known Issues

- **DHT discovery** may take up to 1 minute on first launch — this is normal, the network is warming up
- **P2P behind strict NAT** may not work without a relay server — add relay peers to bootstrap
- **Two nodes on the same PC** require different databases (unique PeerID) — use `--db-path`
- **Console window** on Windows launch — will be fixed in the next version

### Technologies

- Go 1.26
- Fyne v2.4.4 (UI)
- SQLite (mattn/go-sqlite3 v1.14.32)
- libp2p v0.32.0 (P2P network)
- Ed25519, SHA-256 (cryptography)

---

[0.1.0]: https://github.com/Drekard/projectT/releases/tag/v0.1.0
