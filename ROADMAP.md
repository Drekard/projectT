# 🗺️ ProjectT Development Roadmap

**Current version:** v0.1.0 (beta)

---

## 🎯 Vision

ProjectT should become a place where people:
- store and organize collections (cards, tags, folders)
- share them in a P2P network without intermediaries
- communicate and find like-minded people

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

### 🔐 Privacy and Access
Control over who sees your items:
- private / shared / public cards
- access requests to items

### 🌍 Network Search
Find items from other peers:
- search by title and tags
- result aggregation from multiple peers

### 👥 Group Chats
Discuss collections in mesh network:
- create group chats
- invite participants by PeerID
- history synchronization

### 📢 Channels
One-way item broadcasting:
- create channels
- subscribe to other people's channels
- publish items to channels

### 📰 Feed
Unified stream of new items:
- subscribe to peer's feed
- notifications about new content
- filtering by tags and sources

### 👤 Social Elements
User profiles and categories:
- public profile with showcase
- categories / interests
- recommendations based on shared tags

---

## 📍 Current Focus (v0.2)

Coming weeks — **stabilization and polishing**:

- File logging instead of `fmt.Println`
- System notifications for messages
- Fix connection across different networks
- CI/CD (automatic release builds)
- Unit tests for P2P protocols

In parallel — bug fixes as they are found.

---

## 🔮 What's Next?

Order will be determined by:
- what I feel like building most
- what's easier to implement
- what will have the greatest impact

No rigid plans. There's a direction.

---

<div align="center">

**The project lives and evolves. ROADMAP is updated based on progress.**

</div>