package database

import (
	"log"
	"strings"
)

// RunMigrations выполняет миграции базы данных
// Структура:
// 1. Создание всех таблиц
// 2. Создание всех индексов
// 3. Триггеры и ограничения
// 4. Seed данные
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
	// ЧАСТЬ 4: Миграции существующих таблиц
	// ============================================================

	migrateItemsAddStatusColumn()
	createRemoteItemUniqueIndex()

	// ============================================================
	// ЧАСТЬ 4.1: Миграция таблицы favorites (исправление entity_id -> entity_uuid)
	// ============================================================

	migrateFavoritesTable()

	// ============================================================
	// ЧАСТЬ 4.2: Удаление устаревшего триггера validate_favorites_entity_uuid_insert
	// ============================================================

	dropFavoritesValidateTrigger()

	// ============================================================
	// ЧАСТЬ 4.3: Миграция для peer_addresses и profiles
	// ============================================================

	migratePeerAddressesAndProfiles()

	// ============================================================
	// ЧАСТЬ 5: SEED ДАННЫЕ
	// ============================================================

	seedBootstrapPeers()

	log.Println("Все миграции базы данных выполнены успешно")
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
		log.Fatal("Ошибка при создании таблицы items:", err)
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
		log.Fatal("Ошибка при создании таблицы files:", err)
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
		log.Fatal("Ошибка при создании таблицы tags:", err)
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
		log.Fatal("Ошибка при создании таблицы item_tags:", err)
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
		log.Fatal("Ошибка при создании таблицы favorites:", err)
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
		log.Fatal("Ошибка при создании таблицы pinned_items:", err)
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
		log.Fatal("Ошибка при создании таблицы item_files:", err)
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
			avatar_path     TEXT,
			background_path TEXT DEFAULT '',
			content_char    TEXT,
			pinned_uuids    TEXT DEFAULT '[]',
			cached_at       DATETIME,
			
			-- Поля для отслеживания подключений
			last_connected  DATETIME,
			connection_count INTEGER DEFAULT 0,
			
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы profiles:", err)
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
		log.Fatal("Ошибка при создании таблицы profile_keys:", err)
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
		log.Fatal("Ошибка при создании таблицы contacts:", err)
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
		log.Fatal("Ошибка при создании таблицы chat_messages:", err)
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
		log.Fatal("Ошибка при создании таблицы chats:", err)
	}
}

func createPeerAddressesTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS peer_addresses (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id      INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
			
			-- Адрес
			multiaddr       TEXT NOT NULL,
			
			-- Тип адреса (для приоритета подключения)
			address_type    TEXT NOT NULL CHECK (address_type IN (
				'bootstrap',   -- Публичный узел для входа в сеть
				'contact',     -- Личный контакт пользователя
				'discovered'   -- Найден через peer exchange / DHT
			)),
			
			-- Статус
			is_active       BOOLEAN DEFAULT 1,
			last_connected  DATETIME,
			last_seen       DATETIME,
			
			-- Метаданные подключения
			priority        INTEGER DEFAULT 0,
			source          TEXT,
			
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			
			UNIQUE(profile_id, multiaddr)
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы peer_addresses:", err)
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
		// UNIQUE индекс для предотвращения дубликатов remote элементов
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_items_remote_unique ON items(source_peer_id, element_uuid)`,
	}

	for _, sql := range indexes {
		if _, err := DB.Exec(sql); err != nil {
			log.Printf("Ошибка при создании индекса: %v", err)
		}
	}
}

func createFilesIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_files_hash: %v", err)
	}
}

func createTagsIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_tags_tag_uuid ON tags(tag_uuid)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_owner_peer_id ON tags(owner_peer_id)`,
	}

	for _, sql := range indexes {
		if _, err := DB.Exec(sql); err != nil {
			log.Printf("Ошибка при создании индекса: %v", err)
		}
	}
}

func createItemTagsIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_item_tags_element_uuid ON item_tags(item_element_uuid)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_item_tags_element_uuid: %v", err)
	}
}

func createItemFilesIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_item_files_item_id ON item_files(item_id)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_item_files_item_id: %v", err)
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_item_files_element_uuid ON item_files(item_element_uuid)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_item_files_element_uuid: %v", err)
	}
}

func createProfilesIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_profiles_peer_id ON profiles(peer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_profiles_owner_type ON profiles(owner_type)`,
	}

	for _, sql := range indexes {
		if _, err := DB.Exec(sql); err != nil {
			log.Printf("Ошибка при создании индекса: %v", err)
		}
	}
}

func createContactsIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_contacts_peer_id ON contacts(peer_id)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_contacts_peer_id: %v", err)
	}
}

func createChatMessagesIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_messages_chat_id ON chat_messages(chat_id)`)
	if err != nil {
		// Индекс не создаётся, если колонка chat_id не существует (старые БД)
		log.Printf("Предупреждение: не удалось создать индекс chat_messages.chat_id: %v", err)
	}
}

func createChatsIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_chats_contact_id ON chats(contact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chats_peer_id ON chats(peer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chats_last_message ON chats(last_message_at DESC)`,
	}
	for _, sql := range indexes {
		_, err := DB.Exec(sql)
		if err != nil {
			log.Printf("Ошибка при создании индекса: %v", err)
		}
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
		if _, err := DB.Exec(sql); err != nil {
			log.Printf("Ошибка при создании индекса: %v", err)
		}
	}
}

// ============================================================
// ЧАСТЬ 4: ТРИГГЕРЫ И ОГРАНИЧЕНИЯ
// ============================================================

func createElementUUIDTrigger() {
	_, err := DB.Exec(`
		CREATE TRIGGER IF NOT EXISTS validate_element_uuid_insert
		BEFORE INSERT ON items
		FOR EACH ROW
		WHEN NEW.element_uuid IS NULL
		BEGIN
			SELECT RAISE(ABORT, 'element_uuid cannot be NULL');
		END
	`)
	if err != nil {
		log.Printf("Ошибка при создании триггера validate_element_uuid_insert: %v", err)
	}
}

// ============================================================
// ЧАСТЬ 4.5: МИГРАЦИИ СУЩЕСТВУЮЩИХ ТАБЛИЦ
// ============================================================

// migrateItemsAddStatusColumn добавляет колонку status в таблицу items
// Для существующих элементов устанавливается status = 'saved'
func migrateItemsAddStatusColumn() {
	// Проверяем, существует ли уже колонка
	var count int
	err := DB.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('items') WHERE name = 'status'
	`).Scan(&count)

	if err != nil {
		log.Printf("Ошибка проверки колонки status: %v", err)
		return
	}

	if count > 0 {
		log.Println("[Миграция] Колонка status уже существует в таблице items")
		return
	}

	// Добавляем колонку
	_, err = DB.Exec(`
		ALTER TABLE items ADD COLUMN status TEXT DEFAULT 'saved' CHECK (status IN ('saved', 'preview', 'archived'))
	`)
	if err != nil {
		log.Printf("Ошибка добавления колонки status: %v", err)
		return
	}

	log.Println("[Миграция] Колонка status добавлена в таблицу items")
}

// createRemoteItemUniqueIndex создаёт UNIQUE индекс для remote элементов
// Это необходимо для работы ON CONFLICT в CreateRemoteItem
func createRemoteItemUniqueIndex() {
	_, err := DB.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_items_remote_unique 
		ON items(source_peer_id, element_uuid)
	`)
	if err != nil {
		log.Printf("[Миграция] Ошибка создания UNIQUE индекса: %v", err)
	} else {
		log.Println("[Миграция] UNIQUE индекс для remote элементов создан")
	}
}

// ============================================================
// ЧАСТЬ 5: ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ============================================================

// seedBootstrapPeers добавляет предопределённые bootstrap-узлы
// Пользователь может добавить свои bootstrap пиры самостоятельно через UI
// Начальные bootstrap-узлы добавляются при первом запуске
func seedBootstrapPeers() {
	// Добавляем начальные bootstrap-узлы если таблица пустая
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM peer_addresses WHERE address_type = 'bootstrap'`).Scan(&count)
	if err != nil || count > 0 {
		return // Таблица уже содержит bootstrap-узлы
	}

	// Пример начальных bootstrap-узлов (заглушки для демонстрации)
	// В реальности здесь будут адреса публичных узлов проекта ProjectT
	initialBootstrap := []string{
		// Формат: multiaddr адреса
		// "/ip4/bootstrap1.projectt.io/tcp/4001/p2p/QmBootstrap1",
		// "/ip4/bootstrap2.projectt.io/tcp/4001/p2p/QmBootstrap2",
	}

	for _, addr := range initialBootstrap {
		// Сначала создаём профиль для bootstrap пира
		var profileID int64
		peerID := extractPeerID(addr) // Извлекаем PeerID из адреса

		err := DB.QueryRow(`
			INSERT INTO profiles (owner_type, peer_id, username, cached_at)
			VALUES ('remote', ?, 'Bootstrap Node', CURRENT_TIMESTAMP)
			ON CONFLICT(peer_id) DO UPDATE SET cached_at = CURRENT_TIMESTAMP
			RETURNING id
		`, peerID).Scan(&profileID)

		if err != nil {
			log.Printf("Предупреждение: не удалось создать профиль для bootstrap %s: %v", peerID, err)
			continue
		}

		// Добавляем адрес
		_, err = DB.Exec(`
			INSERT INTO peer_addresses (profile_id, multiaddr, address_type, source, priority, is_active)
			VALUES (?, ?, 'bootstrap', 'hardcoded', 10, 1)
		`, profileID, addr)

		if err != nil {
			log.Printf("Предупреждение: не удалось добавить bootstrap адрес %s: %v", addr, err)
		}
	}

	if len(initialBootstrap) > 0 {
		log.Printf("Добавлено %d начальных bootstrap-узлов", len(initialBootstrap))
	}
}

// extractPeerID извлекает PeerID из multiaddr строки
func extractPeerID(multiaddr string) string {
	// Простая реализация: ищем последнюю часть после /p2p/
	// В production используйте github.com/multiformats/go-multiaddr
	parts := strings.Split(multiaddr, "/p2p/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return "unknown_bootstrap_" + multiaddr[len(multiaddr)-8:]
}

// migrateFavoritesTable исправляет схему таблицы favorites
// Удаляет таблицу с entity_id и пересоздает с entity_uuid
func migrateFavoritesTable() {
	// Проверяем, есть ли таблица favorites
	var tableName string
	err := DB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='favorites'`).Scan(&tableName)
	if err != nil {
		// Таблицы нет, ничего делать не нужно
		return
	}

	// Проверяем, есть ли колонка entity_id (старая схема)
	var columnName string
	err = DB.QueryRow(`PRAGMA table_info(favorites)`).Scan(&columnName)
	if err != nil {
		return
	}

	// Получаем информацию о колонках
	rows, err := DB.Query(`PRAGMA table_info(favorites)`)
	if err != nil {
		return
	}
	defer rows.Close()

	hasEntityID := false
	hasEntityUUID := false
	for rows.Next() {
		var cid, notnull, pk int
		var name, dfltValue string
		var typ string
		err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk)
		if err != nil {
			continue
		}
		if name == "entity_id" {
			hasEntityID = true
		}
		if name == "entity_uuid" {
			hasEntityUUID = true
		}
	}

	// Если есть entity_id и нет entity_uuid, нужно пересоздать таблицу
	if hasEntityID && !hasEntityUUID {
		log.Println("[Миграция favorites] Обнаружена старая схема с entity_id, пересоздаю таблицу...")

		// Сохраняем данные
		type FavData struct {
			ID         int
			EntityType string
			EntityID   int
		}
		var oldData []FavData

		rows, err := DB.Query(`SELECT id, entity_type, entity_id FROM favorites`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var fav FavData
				if err := rows.Scan(&fav.ID, &fav.EntityType, &fav.EntityID); err == nil {
					oldData = append(oldData, fav)
				}
			}
		}

		// Удаляем старую таблицу
		_, err = DB.Exec(`DROP TABLE favorites`)
		if err != nil {
			log.Printf("[Миграция favorites] Ошибка удаления старой таблицы: %v", err)
			return
		}

		// Создаем новую таблицу с правильной схемой
		createFavoritesTable()

		// Вставляем данные обратно (используем entity_id как entity_uuid для совместимости)
		// В реальной ситуации нужно мапить ID на UUID
		for _, fav := range oldData {
			_, err := DB.Exec(`INSERT INTO favorites (entity_type, entity_uuid) VALUES (?, ?)`,
				fav.EntityType, fav.EntityID)
			if err != nil {
				log.Printf("[Миграция favorites] Ошибка вставки данных: %v", err)
			}
		}

		log.Println("[Миграция favorites] Таблица успешно обновлена")
	} else if !hasEntityUUID {
		// Таблица есть, но нет ни entity_id ни entity_uuid - что-то не так
		log.Println("[Миграция favorites] Таблица имеет неизвестную схему, пересоздаю...")
		_, err = DB.Exec(`DROP TABLE favorites`)
		if err != nil {
			log.Printf("[Миграция favorites] Ошибка удаления таблицы: %v", err)
			return
		}
		createFavoritesTable()
		log.Println("[Миграция favorites] Таблица пересоздана")
	} else {
		log.Println("[Миграция favorites] Таблица уже имеет правильную схему")
	}
}

// dropFavoritesValidateTrigger удаляет устаревший триггер, требующий entity_uuid
func dropFavoritesValidateTrigger() {
	// Проверяем, существует ли триггер
	var triggerName string
	err := DB.QueryRow(`SELECT name FROM sqlite_master WHERE type='trigger' AND name='validate_favorites_entity_uuid_insert'`).Scan(&triggerName)
	if err != nil {
		// Триггера нет, ничего делать не нужно
		return
	}

	// Удаляем триггер
	_, err = DB.Exec(`DROP TRIGGER validate_favorites_entity_uuid_insert`)
	if err != nil {
		log.Printf("[Миграция] Ошибка удаления триггера: %v", err)
		return
	}

	log.Println("[Миграция] Триггер validate_favorites_entity_uuid_insert удален")
}

// migratePeerAddressesAndProfiles создаёт таблицу peer_addresses и обновляет profiles
func migratePeerAddressesAndProfiles() {
	// Проверяем, существует ли таблица peer_addresses
	var tableName string
	err := DB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='peer_addresses'`).Scan(&tableName)
	if err != nil {
		// Таблицы нет - она будет создана через createPeerAddressesTable()
		// Эта функция вызывается выше в RunMigrations()
		log.Println("[Миграция] Таблица peer_addresses будет создана")
	} else {
		log.Println("[Миграция] Таблица peer_addresses уже существует")
	}

	// Проверяем, существует ли таблица bootstrap_peers
	err = DB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='bootstrap_peers'`).Scan(&tableName)
	if err == nil {
		// Таблица bootstrap_peers существует - переносим данные и удаляем
		log.Println("[Миграция] Перенос данных из bootstrap_peers...")

		// Переносим данные из bootstrap_peers в peer_addresses
		_, err = DB.Exec(`
			INSERT OR IGNORE INTO peer_addresses (profile_id, multiaddr, address_type, source, priority, is_active)
			SELECT 
				COALESCE(
					(SELECT id FROM profiles WHERE profiles.peer_id = bp.peer_id),
					(SELECT id FROM profiles WHERE profiles.peer_id = (
						SELECT peer_id FROM bootstrap_peers WHERE multiaddr = bp.multiaddr LIMIT 1
					))
				) as profile_id,
				bp.multiaddr,
				'bootstrap',
				'hardcoded',
				10,
				bp.is_active
			FROM bootstrap_peers bp
		`)
		if err != nil {
			log.Printf("[Миграция] Предупреждение: не удалось перенести bootstrap_peers: %v", err)
		}

		// Проверяем, есть ли ещё записи в bootstrap_peers
		var count int
		row := DB.QueryRow(`SELECT COUNT(*) FROM peer_addresses WHERE address_type = 'bootstrap'`)
		if err := row.Scan(&count); err != nil {
			log.Printf("[Миграция] Предупреждение: ошибка подсчёта bootstrap-узлов: %v", err)
		} else if count > 0 {
			log.Printf("[Миграция] Перенесено %d bootstrap-узлов", count)
		}

		// Удаляем старую таблицу
		_, err = DB.Exec(`DROP TABLE IF EXISTS bootstrap_peers`)
		if err != nil {
			log.Printf("[Миграция] Предупреждение: не удалось удалить bootstrap_peers: %v", err)
		} else {
			log.Println("[Миграция] Таблица bootstrap_peers удалена")
		}
	}

	// Проверяем, существует ли таблица contacts для переноса адресов
	err = DB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='contacts'`).Scan(&tableName)
	if err == nil {
		log.Println("[Миграция] Перенос адресов из contacts...")

		// Переносим адреса из contacts в peer_addresses
		_, err = DB.Exec(`
			INSERT OR IGNORE INTO peer_addresses (profile_id, multiaddr, address_type, source, priority, is_active)
			SELECT 
				p.id,
				c.multiaddr,
				'contact',
				'user_added',
				10,
				1
			FROM contacts c
			JOIN profiles p ON c.peer_id = p.peer_id
			WHERE c.multiaddr IS NOT NULL AND c.multiaddr != ''
		`)
		if err != nil {
			log.Printf("[Миграция] Предупреждение: не удалось перенести contacts: %v", err)
		} else {
			var count int
			row := DB.QueryRow(`SELECT COUNT(*) FROM peer_addresses WHERE address_type = 'contact'`)
			if err := row.Scan(&count); err != nil {
				log.Printf("[Миграция] Предупреждение: ошибка подсчёта адресов контактов: %v", err)
			} else if count > 0 {
				log.Printf("[Миграция] Перенесено %d адресов контактов", count)
			}
		}
	}

	// Проверяем, есть ли уже поля last_connected и connection_count в profiles
	var hasLastConnected, hasConnectionCount bool
	rows, err := DB.Query(`PRAGMA table_info(profiles)`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull, dfltValue, pk int
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
				continue
			}
			if name == "last_connected" {
				hasLastConnected = true
			}
			if name == "connection_count" {
				hasConnectionCount = true
			}
		}
	}

	// Добавляем поля если их нет
	if !hasLastConnected {
		_, err = DB.Exec(`ALTER TABLE profiles ADD COLUMN last_connected DATETIME`)
		if err != nil {
			log.Printf("[Миграция] Предупреждение: не удалось добавить last_connected: %v", err)
		} else {
			log.Println("[Миграция] Добавлено поле last_connected")
		}
	}

	if !hasConnectionCount {
		_, err = DB.Exec(`ALTER TABLE profiles ADD COLUMN connection_count INTEGER DEFAULT 0`)
		if err != nil {
			log.Printf("[Миграция] Предупреждение: не удалось добавить connection_count: %v", err)
		} else {
			log.Println("[Миграция] Добавлено поле connection_count")
		}
	}

	log.Println("[Миграция] Миграция peer_addresses и profiles завершена")
}
