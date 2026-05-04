package database

import (
	"strings"
)

// RunMigrations выполняет миграции базы данных
func RunMigrations() {
	// ============================================================
	// ЧАСТЬ 1: СОЗДАНИЕ ТАБЛИЦ
	// ============================================================

	createItemsTable()
	createFilesTable()
	createTagsTable()
	createItemTagsTable()
	createFavoritesTable()
	createPinnedItemsTable()
	createItemFilesTable()
	createProfilesTable()
	createProfileKeysTable()
	createContactsTable()
	createChatsTable()
	createChatMessagesTable()
	createPeerAddressesTable()

	// ============================================================
	// ЧАСТЬ 2: СОЗДАНИЕ ИНДЕКСОВ
	// ============================================================

	createItemsIndexes()
	createFilesIndexes()
	createTagsIndexes()
	createItemTagsIndexes()
	createItemFilesIndexes()
	createProfilesIndexes()
	createContactsIndexes()
	createChatsIndexes()
	createChatMessagesIndexes()
	createPeerAddressesIndexes()

	// ============================================================
	// ЧАСТЬ 3: ТРИГГЕРЫ И ОГРАНИЧЕНИЯ
	// ============================================================

	createElementUUIDTrigger()

	// ============================================================
	// ЧАСТЬ 4: SEED ДАННЫЕ
	// ============================================================

	seedBootstrapPeers()
}

// ============================================================
// ЧАСТЬ 1: ФУНКЦИИ СОЗДАНИЯ ТАБЛИЦ
// ============================================================

func createItemsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id              INTEGER PRIMARY KEY,
			element_uuid    TEXT,
			hash            TEXT,
			owner_type      TEXT DEFAULT 'local' CHECK (owner_type IN ('local', 'remote')),
			source_peer_id  TEXT,
			type            TEXT NOT NULL CHECK (type IN ('folder', 'element')),
			title           TEXT,
			description     TEXT,
			content_meta    TEXT,
			parent_id       INTEGER,
			signature       BLOB,
			version         INTEGER DEFAULT 1,
			status          TEXT DEFAULT 'saved' CHECK (status IN ('saved', 'preview', 'archived')),
			cached_at       DATETIME,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_id) REFERENCES items (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы items:" + err.Error())
	}
}

func createFilesTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			id          INTEGER PRIMARY KEY,
			hash        TEXT UNIQUE NOT NULL,
			size        INTEGER NOT NULL,
			mime_type   TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы files:" + err.Error())
	}
}

func createTagsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS tags (
			id              INTEGER PRIMARY KEY,
			tag_uuid        TEXT,
			owner_peer_id   TEXT DEFAULT 'local',
			name            TEXT UNIQUE NOT NULL,
			color           TEXT DEFAULT '#FFBB00',
			description     TEXT DEFAULT '',
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы tags:" + err.Error())
	}
}

func createItemTagsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS item_tags (
			item_id             INTEGER,
			item_element_uuid   TEXT,
			tag_id              INTEGER,
			tag_uuid            TEXT,
			PRIMARY KEY (item_id, tag_id),
			FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE CASCADE,
			FOREIGN KEY (tag_id)  REFERENCES tags (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы item_tags:" + err.Error())
	}
}

func createFavoritesTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS favorites (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_type TEXT NOT NULL CHECK (entity_type IN ('tag', 'folder')),
			entity_uuid TEXT NOT NULL
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы favorites:" + err.Error())
	}
}

func createPinnedItemsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS pinned_items (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id           INTEGER NOT NULL,
			item_element_uuid TEXT,
			order_num         INTEGER DEFAULT 0,
			created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы pinned_items:" + err.Error())
	}
}

func createItemFilesTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS item_files (
			item_id             INTEGER NOT NULL,
			item_element_uuid   TEXT,
			hash                TEXT NOT NULL,
			file_path           TEXT NOT NULL,
			size                INTEGER,
			mime_type           TEXT,
			is_remote           BOOLEAN DEFAULT 0,
			source_peer_id      TEXT,
			PRIMARY KEY (item_id, hash)
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы item_files:" + err.Error())
	}
}

func createProfilesTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS profiles (
			id              INTEGER PRIMARY KEY,
			owner_type      TEXT NOT NULL CHECK (owner_type IN ('local', 'remote')),
			peer_id         TEXT UNIQUE NOT NULL,
			username        TEXT NOT NULL,
			title           TEXT,
			avatar_path     TEXT DEFAULT 'storage/files/avatars/local/ProjctT_true.png',
			background_path TEXT DEFAULT '',
			content_char    TEXT,
			pinned_uuids    TEXT DEFAULT '[]',
			cached_at       DATETIME,
			last_connected  DATETIME,
			connection_count INTEGER DEFAULT 0,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы profiles:" + err.Error())
	}
}

func createProfileKeysTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS profile_keys (
			profile_id       INTEGER PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
			private_key      BLOB,
			public_key       BLOB NOT NULL,
			signature        BLOB,
			is_key_encrypted BOOLEAN DEFAULT 0
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы profile_keys:" + err.Error())
	}
}

func createContactsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS contacts (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			peer_id     TEXT UNIQUE NOT NULL REFERENCES profiles(peer_id),
			multiaddr   TEXT,
			notes       TEXT,
			is_blocked  BOOLEAN DEFAULT 0,
			is_favorite BOOLEAN DEFAULT 1,
			last_seen   DATETIME,
			added_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы contacts:" + err.Error())
	}
}

func createChatMessagesTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS chat_messages (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_id      INTEGER NOT NULL,
			from_peer_id TEXT NOT NULL,
			content      TEXT NOT NULL,
			content_type TEXT DEFAULT 'text',
			metadata     TEXT,
			is_read      BOOLEAN DEFAULT 0,
			sent_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at   DATETIME,
			FOREIGN KEY (chat_id) REFERENCES chats (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы chat_messages:" + err.Error())
	}
}

func createChatsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS chats (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			contact_id      INTEGER,
			peer_id         TEXT NOT NULL,
			is_temporary    BOOLEAN DEFAULT 0,
			last_message_at DATETIME,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (contact_id) REFERENCES contacts (id) ON DELETE SET NULL,
			UNIQUE (peer_id)
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы chats:" + err.Error())
	}
}

func createPeerAddressesTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS peer_addresses (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id      INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
			multiaddr       TEXT NOT NULL,
			address_type    TEXT NOT NULL CHECK (address_type IN ('bootstrap', 'contact', 'discovered')),
			is_active       BOOLEAN DEFAULT 1,
			last_connected  DATETIME,
			last_seen       DATETIME,
			priority        INTEGER DEFAULT 0,
			source          TEXT,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(profile_id, multiaddr)
		);
	`)
	if err != nil {
		panic("Ошибка при создании таблицы peer_addresses:" + err.Error())
	}
}

// ============================================================
// ЧАСТЬ 2: ФУНКЦИИ СОЗДАНИЯ ИНДЕКСОВ
// ============================================================

func createItemsIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_items_parent ON items(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_items_type ON items(type)`,
		`CREATE INDEX IF NOT EXISTS idx_items_updated ON items(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_items_owner_type ON items(owner_type)`,
		`CREATE INDEX IF NOT EXISTS idx_items_source_peer ON items(source_peer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_items_element_uuid ON items(element_uuid)`,
		`CREATE INDEX IF NOT EXISTS idx_items_hash ON items(hash)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_items_remote_unique ON items(source_peer_id, element_uuid)`,
	}

	for _, sql := range indexes {
		_, _ = DB.Exec(sql)
	}
}

func createFilesIndexes() {
	_, _ = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash)`)
}

func createTagsIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_tags_tag_uuid ON tags(tag_uuid)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_owner_peer_id ON tags(owner_peer_id)`,
	}

	for _, sql := range indexes {
		_, _ = DB.Exec(sql)
	}
}

func createItemTagsIndexes() {
	_, _ = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_item_tags_element_uuid ON item_tags(item_element_uuid)`)
}

func createItemFilesIndexes() {
	_, _ = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_item_files_item_id ON item_files(item_id)`)
	_, _ = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_item_files_element_uuid ON item_files(item_element_uuid)`)
}

func createProfilesIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_profiles_peer_id ON profiles(peer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_profiles_owner_type ON profiles(owner_type)`,
	}

	for _, sql := range indexes {
		_, _ = DB.Exec(sql)
	}
}

func createContactsIndexes() {
	_, _ = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_contacts_peer_id ON contacts(peer_id)`)
}

func createChatMessagesIndexes() {
	_, _ = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_messages_chat_id ON chat_messages(chat_id)`)
}

func createChatsIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_chats_contact_id ON chats(contact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chats_peer_id ON chats(peer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chats_last_message ON chats(last_message_at DESC)`,
	}
	for _, sql := range indexes {
		_, _ = DB.Exec(sql)
	}
}

func createPeerAddressesIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_peer_addresses_profile_id ON peer_addresses(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_peer_addresses_address_type ON peer_addresses(address_type)`,
		`CREATE INDEX IF NOT EXISTS idx_peer_addresses_active ON peer_addresses(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_peer_addresses_priority ON peer_addresses(priority DESC)`,
	}

	for _, sql := range indexes {
		_, _ = DB.Exec(sql)
	}
}

// ============================================================
// ЧАСТЬ 3: ТРИГГЕРЫ И ОГРАНИЧЕНИЯ
// ============================================================

func createElementUUIDTrigger() {
	_, _ = DB.Exec(`
		CREATE TRIGGER IF NOT EXISTS validate_element_uuid_insert
		BEFORE INSERT ON items
		FOR EACH ROW
		WHEN NEW.element_uuid IS NULL
		BEGIN
			SELECT RAISE(ABORT, 'element_uuid cannot be NULL');
		END
	`)
}

// ============================================================
// ЧАСТЬ 4: SEED ДАННЫЕ
// ============================================================

func seedBootstrapPeers() {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM peer_addresses WHERE address_type = 'bootstrap'`).Scan(&count)
	if err != nil || count > 0 {
		return
	}

	initialBootstrap := []string{}

	for _, addr := range initialBootstrap {
		var profileID int64
		peerID := extractPeerID(addr)

		err := DB.QueryRow(`
			INSERT INTO profiles (owner_type, peer_id, username, cached_at)
			VALUES ('remote', ?, 'Bootstrap Node', CURRENT_TIMESTAMP)
			ON CONFLICT(peer_id) DO UPDATE SET cached_at = CURRENT_TIMESTAMP
			RETURNING id
		`, peerID).Scan(&profileID)

		if err != nil {
			continue
		}

		_, _ = DB.Exec(`
			INSERT INTO peer_addresses (profile_id, multiaddr, address_type, source, priority, is_active)
			VALUES (?, ?, 'bootstrap', 'hardcoded', 10, 1)
		`, profileID, addr)
	}
}

// extractPeerID извлекает PeerID из multiaddr строки
func extractPeerID(multiaddr string) string {
	parts := strings.Split(multiaddr, "/p2p/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return "unknown_bootstrap_" + multiaddr[len(multiaddr)-8:]
}
