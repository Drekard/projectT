<div align="center">

# ProjectT

### 🗂️ A Hybrid of File Explorer and Pinterest on Steroids

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Version](https://img.shields.io/badge/Version-0.1.0-blue?style=flat-square)](CHANGELOG.md)
[![Fyne](https://img.shields.io/badge/UI-Fyne-blue?style=flat-square&logo=go&logoColor=white)](https://fyne.io/)
[![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://www.sqlite.org/)
[![libp2p](https://img.shields.io/badge/P2P-libp2p-ff69b4?style=flat-square&logo=go&logoColor=white)](https://libp2p.io/)

**This is a hybrid of a File Explorer and Pinterest, where objects live as semantic units rather than scattered files, with P2P sharing to share collections without compromising privacy.**

<img src="assets/screenshots/ProjctT_true.png" alt="Logo" width="30%">

</div>

---

## 📊 Technologies

- **UI** - Fyne with custom theme, card widgets, and adaptive grid
- **Business Logic** - Services for managing items, tags, favorites, pinned items
- **Storage** - SQLite for metadata + file system for content
- **P2P** - libp2p for decentralized exchange (DHT, relay, STUN)
- **Cryptography** - Ed25519 for signing, SHA-256 for hashing, XOR encryption

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

## 🟢 Current State (v0.1.0)

> **Beta release.** The application works, but has limitations.

**Already working:**
- ✅ Cards, tags, folders, search
- ✅ P2P 1-on-1 chat, file transfer, profile exchange
- ✅ Peer discovery on the same network

**Known issues:**
- 🔧 Tab transitions are slow
- 🔧 Connection only works on the same WiFi network
- 🔧 No system notifications

**Plans:** stabilization → group chats → global search → channels → feed

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
- **Local Storage** — everything lives on your machine, no clouds or subscriptions
- **Pinterest-style Grid** — adaptive, with resizable cards
- **Search & Sorting** — by tags, title, date, type
- **Favorites & Pinned** — for quick access

**P2P Functionality:**
- **Peer-to-Peer Chat** — text messages with encryption
- **File Transfer** — direct between users
- **Profile Exchange** — automatic synchronization on connection
- **Peer Discovery** — DHT, mDNS, bootstrap nodes
- **Security** — master password, cryptographic keys, blacklist
- **Offline Mode** — message queue for unavailable peers

---

## ⛓️ Running

Download the latest [release](https://github.com/Drekard/projectT/releases) and run `projectT.exe`.

---

## ❓ FAQ

**How to connect two devices?**
Currently, both devices must be connected to the same WiFi network.
To connect, go to the P2P tab, copy your local address, and transfer it to the second device by pasting the address into the connection field on the same tab.
Now you can start chatting between devices, exchange profiles, and share your items via chat.

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
More details — in [FEATURES.md](FEATURES.md#-p2p-chat-and-exchange).

---

## 📚 **Learn More:**

- [Full Feature List](FEATURES.md)
- [Project Architecture](ARCHITECTURE.md)
- [This Version's Capabilities](CHANGELOG.md)

---

## 👨‍💻 Author

**ProjectT** is a diploma project for the portfolio of a student programmer in their final year of study, which is me.

I'm simultaneously looking for a developer position. I can design architecture, work with GUIs and databases, and have implemented a full P2P messenger on libp2p with encryption and data synchronization.

Have ideas, suggestions, or just a question? Feel free to reach out via my personal messages

Telegram - @Redoranar

If you're interested, I also have a Pinterest at https://ru.pinterest.com/egors3206/

---

<div align="center">

**This is not a file explorer replacement. It's a new decentralized space for those who collect, store, and get inspired.**
</div>