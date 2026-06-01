# 🗺️ ProjectT Development Roadmap

**Current version:** v0.2.0-dev (pre-release)

---

## 🎯 Vision

ProjectT should become a place where people:
- store and organize collections (cards, tags, folders)
- share them in a P2P network without intermediaries
- communicate and find like-minded people
- get rewarded for contributing to the network

---

## 🔥 What Already Works (v0.2.0-dev)

- ✅ Cards with any content (files, links, text, images)
- ✅ Tags (colored, with autocomplete)
- ✅ Local storage without clouds
- ✅ Item visibility (public/private)
- ✅ Remote items (preview from peers)
- ✅ Breadcrumbs navigation
- ✅ P2P 1-on-1 chat with encryption
- ✅ Group chats (pubsub-based, roles, invites)
- ✅ Channels (one-way broadcasting)
- ✅ File transfer (single + batch)
- ✅ Profile exchange and sync
- ✅ Peer discovery (mDNS, DHT, bootstrap)
- ✅ NAT traversal helpers (public IP detection, multi-address)
- ✅ Prometheus metrics + Grafana dashboards

---

## 🚀 Major Development Areas (order not important)

### 🐛 Bug Fixes & Stabilization (Current Focus)
Make everything work reliably:
- [ ] Fix cross-network connection bugs
- [ ] Integrate group chats into UI
- [ ] Test and stabilize batch transfer
- [ ] Replace `fmt.Println` with structured file logging
- [ ] Performance profiling and UI optimization
- [ ] Unit tests for P2P protocols
- [ ] System notifications for incoming messages (systray)

### 🎨 Telegram-Style Interface
Familiar and clean messaging experience:
- [ ] Chat list with last message preview and unread counters
- [ ] Rounded message bubbles with sender colors
- [ ] Online status indicators (green dots on avatars)
- [ ] Delivery confirmations (✓ / ✓✓)
- [ ] Attachment button and emoji picker
- [ ] "typing..." status
- [ ] Dark/light theme with Telegram-like palette
- [ ] Bottom navigation bar (elements / chats / settings)

### 📰 Feed
Unified stream of new content:
- [ ] Subscribe to peers' item feeds
- [ ] Notifications about new content from subscriptions
- [ ] Filtering by tags and sources
- [ ] Mix of channel posts and peer shares

### 🌍 Network Search
Find items from other peers:
- [ ] Search by title and tags across the network
- [ ] Result aggregation from multiple peers
- [ ] Cached search results for offline access

### 🪙 Network Rewards (Crypto)
Incentivize users for contributing resources:
- [ ] Internal point system for storing and relaying data
- [ ] Rewards for: file sharing, profile exchange, uptime, library hosting
- [ ] Signed receipts as proof of service (Proof of Service)
- [ ] Wallet UI with balance and transaction history
- [ ] Anti-abuse: daily caps, rate limits, reputation penalties
- [ ] Future: migration to real blockchain token (ERC-20 / TON)

---

## 📍 Current Focus

**Stabilization before v0.2.0 release:**

1. Fix cross-network connection bugs (NAT, relay, firewall)
2. Integrate group chats UI (create, join, message, manage)
3. Test batch transfer end-to-end
4. Structured file logging (replace `fmt.Println`)
5. System notifications (systray)
6. CI/CD pipeline for automatic builds

---

## 🔮 What's Next?

Order will be determined by:
- what I feel like building most
- what's easier to implement first
- what will have the greatest impact on the experience

No rigid plans. There's a direction.

---

<div align="center">

**The project lives and evolves. ROADMAP is updated based on progress.**

</div>
