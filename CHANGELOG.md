# Changelog

All notable changes to ProjectT.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased] — v0.2.0 (in development)

### Added

**Group Chats & Channels:**
- Group chat creation with name, description, and type (group/channel)
- Invite tokens with configurable depth limits (chain invitations)
- Role system: creator, admin, member, subscriber
- Pubsub-based message distribution via libp2p
- Lamport clock-based message ordering and synchronization
- Cryptographic membership proofs (Ed25519 signed)
- Admin actions: kick, ban, change roles
- Channel mode: only admins/creators can publish
- History sync on join (fetch from existing members)

**Item Visibility & Status:**
- Public/private visibility per item (controls P2P sharing)
- Item status: saved, preview (from peer), archived
- Remote items: browse peer's items without saving to collection
- Owner type: local vs remote item tracking

**Batch Transfer:**
- Send multiple items in a single transfer session
- Folder transfer: send all items in a folder
- Pinned items transfer: share your pinned collection
- Custom selection transfer
- Progress tracking per item and overall batch
- Partial success handling

**NAT Traversal & Connection:**
- Public IP auto-detection
- Multi-address format: `projectt:peerid@ip1:port1;ip2:port2`
- Address type classification: localhost, LAN, public
- Improved connection logging with diagnostics
- Legacy address format compatibility

**Profile Sync:**
- Profile synchronization across peers
- Remote profile viewing with item showcase

**Monitoring:**
- Prometheus metrics export (configurable port)
- Metrics: items, tags, chat, P2P, database, runtime
- Grafana dashboard with pre-configured visualizations
- Docker Compose monitoring stack

**UI Improvements:**
- Breadcrumbs navigation for folder hierarchy
- Chat sidebar for quick access to conversations
- Search popup overlay
- Batch transfer progress indicator
- Remote profile view
- Compilation view
- Additional text display on cards
- Improved color scheme

**Infrastructure:**
- CI/CD pipeline (GitHub Actions)
- Cross-platform test adaptation (Linux/Windows)
- Pre-commit hooks

### Changed

- System language: full translation from Russian to English
- Item model: added visibility, status, owner_type, hash fields
- Database: split queries into focused files (item_crud, item_search, etc.)
- Transfer service: refactored into single_transfer + batch_transfer
- ItemSync: refactored into service/handler/requests/types/verify
- UI workspace: restructured into init/navigation/content/remote
- Chat UI: refactored center panel into separate components
- Sidebar: added chat sidebar, reorganized navigation
- Header: added breadcrumbs, search popup
- Configuration: added Prometheus settings

### Fixed

- Nil pointer checks in tests
- CI/CD pipeline operation
- Test adaptation for Linux environments

### Known Issues

- Group chats not yet integrated with UI (backend complete)
- Cross-network connections work but have bugs
- Batch transfer needs end-to-end testing

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

[Unreleased]: https://github.com/Drekard/projectT/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Drekard/projectT/releases/tag/v0.1.0
