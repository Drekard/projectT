package database

import (
	"log"
	"strings"
)

// RunMigrations выполняет миграции базы данных
func RunMigrations() {
	// 1. ТАБЛИЦА ЭЛЕМЕНТОВ (основная)
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id          INTEGER PRIMARY KEY,
			type        TEXT NOT NULL CHECK (type IN ('folder', 'element')),
			title       TEXT,
			description TEXT,
			content_meta TEXT,
			parent_id   INTEGER,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_id) REFERENCES items (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы items:", err)
	}

	// 2. ТАБЛИЦА ФАЙЛОВ (для дедупликации)
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			id          INTEGER PRIMARY KEY,
			hash        TEXT UNIQUE NOT NULL,
			size        INTEGER NOT NULL,
			mime_type   TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы files:", err)
	}

	// 3. ТЕГИ
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS tags (
			id          INTEGER PRIMARY KEY,
			name        TEXT UNIQUE NOT NULL,
			color       TEXT DEFAULT '#FFBB00',
			description TEXT DEFAULT ''
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы tags:", err)
	}

	// 4. СВЯЗЬ ЭЛЕМЕНТОВ С ТЕГАМИ
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS item_tags (
			item_id INTEGER,
			tag_id  INTEGER,
			PRIMARY KEY (item_id, tag_id),
			FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE CASCADE,
			FOREIGN KEY (tag_id)  REFERENCES tags (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы item_tags:", err)
	}

	// 5. ИЗБРАННОЕ
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS favorites (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_type TEXT NOT NULL CHECK (entity_type IN ('tag', 'folder')),
			entity_id   INTEGER NOT NULL
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы favorites:", err)
	}

	// 6. ЗАКРЕПЛЁННЫЕ ЭЛЕМЕНТЫ В ПРОФИЛЕ
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pinned_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id INTEGER NOT NULL,
			order_num INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы pinned_items:", err)
	}

	// ИНДЕКСЫ для производительности
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_items_parent ON items(parent_id);`)
	if err != nil {
		log.Fatal("Ошибка при создании индекса idx_items_parent:", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_items_type ON items(type);`)
	if err != nil {
		log.Fatal("Ошибка при создании индекса idx_items_type:", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_items_updated ON items(updated_at DESC);`)
	if err != nil {
		log.Fatal("Ошибка при создании индекса idx_items_updated:", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash);`)
	if err != nil {
		log.Fatal("Ошибка при создании индекса idx_files_hash:", err)
	}

	// Если таблица tags уже существует, добавляем поле color
	_, err = DB.Exec(`ALTER TABLE tags ADD COLUMN color TEXT DEFAULT '#FFBB00'`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		// Проверяем, возможно столбец уже существует
		if err.Error() != "duplicate column name: color" {
			_ = err //nolint:staticcheck // Логируем ошибку, но не выводим в пользовательский интерфейс
		}
	}

	// Добавляем поле description, если оно не существует
	_, err = DB.Exec(`ALTER TABLE tags ADD COLUMN description TEXT DEFAULT ''`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		_ = err //nolint:staticcheck // Логируем ошибку, но не выводим в пользовательский интерфейс
	}

	// 7. ТАБЛИЦА ПРОФИЛЯ ПОЛЬЗОВАТЕЛЯ
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS profile (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL DEFAULT 'Аноним',
			status TEXT NOT NULL DEFAULT 'Доступен',
			avatar_path TEXT,
			background_path TEXT DEFAULT '',
			content_characteristic TEXT,
			demo_elements TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы profile:", err)
	}

	// Добавляем поле background_path, если оно не существует
	_, err = DB.Exec(`ALTER TABLE profile ADD COLUMN background_path TEXT DEFAULT ''`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		// Проверяем, возможно столбец уже существует
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении столбца background_path: %v", err)
		}
	}

	// Добавляем поле content_characteristic, если оно не существует
	_, err = DB.Exec(`ALTER TABLE profile ADD COLUMN content_characteristic TEXT`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		// Проверяем, возможно столбец уже существует
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении столбца content_characteristic: %v", err)
		}
	}

	// Добавляем поле demo_elements, если оно не существует
	_, err = DB.Exec(`ALTER TABLE profile ADD COLUMN demo_elements TEXT`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		// Проверяем, возможно столбец уже существует
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении столбца demo_elements: %v", err)
		}
	}

	// Добавляем индекс для производительности
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_profile_username ON profile(username);`)
	if err != nil {
		_ = err //nolint:staticcheck // Логируем ошибку, но не выводим в пользовательский интерфейс
	}

	// Создаём P2P таблицы
	createP2PTables()
	seedBootstrapPeers()
}

// createP2PTables создаёт таблицы для P2P функциональности
func createP2PTables() {
	// 1. Таблица p2p_profile - профиль P2P узла
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS p2p_profile (
			id           INTEGER PRIMARY KEY CHECK (id = 1),
			peer_id      TEXT UNIQUE NOT NULL,
			private_key  BLOB NOT NULL,
			public_key   BLOB NOT NULL,
			username     TEXT NOT NULL,
			status       TEXT DEFAULT 'online',
			listen_addrs TEXT,
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Printf("Ошибка при создании таблицы p2p_profile: %v", err)
	}

	// 2. Таблица contacts - адресная книга
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS contacts (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			peer_id     TEXT UNIQUE NOT NULL,
			username    TEXT NOT NULL,
			public_key  BLOB,
			multiaddr   TEXT,
			status      TEXT DEFAULT 'offline',
			last_seen   DATETIME,
			notes       TEXT,
			is_blocked  BOOLEAN DEFAULT 0,
			added_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Printf("Ошибка при создании таблицы contacts: %v", err)
	}

	// 3. Таблица chat_messages - история сообщений
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS chat_messages (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			contact_id   INTEGER NOT NULL,
			from_peer_id TEXT NOT NULL,
			content      TEXT NOT NULL,
			content_type TEXT DEFAULT 'text',
			metadata     TEXT,
			is_read      BOOLEAN DEFAULT 0,
			sent_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (contact_id) REFERENCES contacts (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Printf("Ошибка при создании таблицы chat_messages: %v", err)
	}

	// 4. Таблица bootstrap_peers - узлы для входа в сеть
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS bootstrap_peers (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			multiaddr     TEXT UNIQUE NOT NULL,
			peer_id       TEXT,
			is_active     BOOLEAN DEFAULT 1,
			last_connected DATETIME,
			added_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Printf("Ошибка при создании таблицы bootstrap_peers: %v", err)
	}

	// Создаём индексы для производительности
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_contacts_peer_id ON contacts(peer_id);`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_contacts_peer_id: %v", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_messages_contact_id ON chat_messages(contact_id);`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_chat_messages_contact_id: %v", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_bootstrap_peers_multiaddr ON bootstrap_peers(multiaddr);`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_bootstrap_peers_multiaddr: %v", err)
	}

	log.Println("P2P таблицы успешно созданы")
}

// seedBootstrapPeers добавляет предопределённые bootstrap-узлы
func seedBootstrapPeers() {
	// Добавляем несколько публичных bootstrap-узлов libp2p
	bootstrapPeers := []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	}

	for _, addr := range bootstrapPeers {
		_, err := DB.Exec(`
			INSERT OR IGNORE INTO bootstrap_peers (multiaddr, is_active, added_at)
			VALUES (?, 1, CURRENT_TIMESTAMP)
		`, addr)
		if err != nil {
			log.Printf("Ошибка при добавлении bootstrap-узла %s: %v", addr, err)
		}
	}

	log.Println("Bootstrap-узлы успешно добавлены")
}
