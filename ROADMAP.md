# 🗺️ ProjectT Development Roadmap

**Current version:** v0.1.0 (beta)

---

## 🎯 Vision

ProjectT should become a place where people:
- store and organize collections (cards, tags, folders)
- share them in a P2P network without intermediaries
- communicate and find like-minded people
- get rewarded for contributing to the network

---

## 🔥 What Already Works (v0.1)

- ✅ Cards with any content (files, links, text, images)
- ✅ Tags (colored, with autocomplete)
- ✅ Local storage without clouds
- ✅ P2P 1-on-1 chat with encryption
- ✅ File transfer between peers
- ✅ Profile exchange
- ✅ Peer discovery (mDNS, DHT, bootstrap)

---

## 🚀 Major Development Areas (order not important)

### 🌐 Network Stability & Diagnostics
Make P2P work everywhere, not just on the same WiFi:
- connection testing across different networks (4G, different ISPs, NATs)
- relay and STUN configuration for NAT traversal
- structured file logging instead of `fmt.Println`
- `p2p status` diagnostics — connection type, latency, peer count
- visual indicator of connection quality (direct / relay / DHT)

### 💬 Group Chats
Discuss collections in a mesh network:
- create group chats with name, description, avatar
- invite participants by PeerID
- roles: admin, moderator, member
- message history synchronization on reconnect
- group encryption with shared keys
- built on libp2p pubsub (already in dependencies)

### 🎨 Telegram-Style Interface
Familiar and clean messaging experience:
- chat list with last message preview and unread counters
- rounded message bubbles with sender colors
- online status indicators (green dots on avatars)
- delivery confirmations (✓ / ✓✓)
- attachment button and emoji picker
- "typing..." status
- dark/light theme with Telegram-like palette
- bottom navigation bar (elements / chats / settings)

### 📢 Channels
One-way item broadcasting:
- create channels for curated content
- subscribe to other people's channels
- publish items to your channel
- followers receive new items in their feed

### 📰 Feed
Unified stream of new content:
- subscribe to peers' item feeds
- notifications about new content from subscriptions
- filtering by tags and sources
- mix of channel posts and peer shares

### 👤 Social Elements
User profiles and discovery:
- public profile with item showcase
- categories / interests
- recommendations based on shared tags
- reputation system

### 🪙 Network Rewards (Crypto)
Incentivize users for contributing resources:
- internal point system for storing and relaying data
- rewards for: file sharing, profile exchange, uptime, library hosting
- signed receipts as proof of service (Proof of Service)
- wallet UI with balance and transaction history
- anti-abuse: daily caps, rate limits, reputation penalties
- future: migration to real blockchain token (ERC-20 / TON)

### 🔐 Privacy and Access
Control over who sees your items:
- private / shared / public card visibility
- access requests to private items
- per-peer sharing permissions

### 🌍 Network Search
Find items from other peers:
- search by title and tags across the network
- result aggregation from multiple peers
- cached search results for offline access

---

## 📍 Current Focus

**Stabilization and foundations:**

- Replace `fmt.Println` with structured file logging
- Test and fix P2P connections across different networks
- System notifications for incoming messages (systray)
- Performance profiling and UI optimization
- Unit tests for P2P protocols
- CI/CD pipeline for automatic builds

In parallel — bug fixes and small quality-of-life improvements as they appear.

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
