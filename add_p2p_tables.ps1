# Скрипт для добавления P2P таблиц в существующую базу данных ProjectT
# Запуск: powershell -ExecutionPolicy Bypass -File .\add_p2p_tables.ps1

$dbPath = ".\internal\storage\projectT.db"

# Проверяем существование файла базы данных
if (-not (Test-Path $dbPath)) {
    Write-Host "Ошибка: Файл базы данных не найден: $dbPath" -ForegroundColor Red
    Write-Host "Убедитесь, что приложение было запущено хотя бы один раз для создания БД" -ForegroundColor Yellow
    exit 1
}

Write-Host "Найден файл базы данных: $dbPath" -ForegroundColor Green

# Подключаемся к базе данных SQLite с помощью System.Data.SQLite
try {
    # Пробуем загрузить assembly
    Add-Type -AssemblyName "System.Data"
    
    # Создаем подключение
    $connectionString = "Data Source=$dbPath;Version=3;"
    $connection = New-Object System.Data.SQLite.SQLiteConnection($connectionString)
    $connection.Open()
    
    Write-Host "Подключение к базе данных успешно установлено" -ForegroundColor Green
}
catch {
    Write-Host "Ошибка: Не удалось подключиться к базе данных" -ForegroundColor Red
    Write-Host "Убедитесь, что установлен SQLite для PowerShell или используйте альтернативный метод" -ForegroundColor Yellow
    Write-Host "Ошибка: $($_.Exception.Message)" -ForegroundColor Red
    
    # Альтернативный вариант с использованием sqlite3.exe
    if (Test-Path "sqlite3.exe") {
        Write-Host "Попытка использования sqlite3.exe..." -ForegroundColor Yellow
        & .\sqlite3.exe $dbPath @'
-- 1. Таблица p2p_profile
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

-- 2. Таблица contacts
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

-- 3. Таблица chat_messages
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

-- 4. Таблица bootstrap_peers
CREATE TABLE IF NOT EXISTS bootstrap_peers (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    multiaddr      TEXT UNIQUE NOT NULL,
    peer_id        TEXT,
    is_active      BOOLEAN DEFAULT 1,
    last_connected DATETIME,
    added_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_contacts_peer_id ON contacts(peer_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_contact_id ON chat_messages(contact_id);
CREATE INDEX IF NOT EXISTS idx_bootstrap_peers_multiaddr ON bootstrap_peers(multiaddr);

-- Добавляем bootstrap-узлы
INSERT OR IGNORE INTO bootstrap_peers (multiaddr, is_active, added_at) VALUES ('/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN', 1, CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO bootstrap_peers (multiaddr, is_active, added_at) VALUES ('/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa', 1, CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO bootstrap_peers (multiaddr, is_active, added_at) VALUES ('/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb', 1, CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO bootstrap_peers (multiaddr, is_active, added_at) VALUES ('/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt', 1, CURRENT_TIMESTAMP);
'@
        Write-Host "P2P таблицы успешно добавлены через sqlite3.exe" -ForegroundColor Green
        exit 0
    }
    else {
        Write-Host "sqlite3.exe не найден. Попробуйте установить:" -ForegroundColor Red
        Write-Host "  choco install sqlite" -ForegroundColor Yellow
        exit 1
    }
}

# Создаем команду для выполнения SQL
$command = $connection.CreateCommand()

# SQL скрипт для создания P2P таблиц
$sqlScript = @"
-- 1. Таблица p2p_profile - профиль P2P узла
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

-- 2. Таблица contacts - адресная книга
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

-- 3. Таблица chat_messages - история сообщений
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

-- 4. Таблица bootstrap_peers - узлы для входа в сеть
CREATE TABLE IF NOT EXISTS bootstrap_peers (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    multiaddr      TEXT UNIQUE NOT NULL,
    peer_id        TEXT,
    is_active      BOOLEAN DEFAULT 1,
    last_connected DATETIME,
    added_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для производительности
CREATE INDEX IF NOT EXISTS idx_contacts_peer_id ON contacts(peer_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_contact_id ON chat_messages(contact_id);
CREATE INDEX IF NOT EXISTS idx_bootstrap_peers_multiaddr ON bootstrap_peers(multiaddr);

-- Добавляем публичные bootstrap-узлы libp2p
INSERT OR IGNORE INTO bootstrap_peers (multiaddr, is_active, added_at) VALUES ('/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN', 1, CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO bootstrap_peers (multiaddr, is_active, added_at) VALUES ('/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa', 1, CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO bootstrap_peers (multiaddr, is_active, added_at) VALUES ('/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb', 1, CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO bootstrap_peers (multiaddr, is_active, added_at) VALUES ('/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt', 1, CURRENT_TIMESTAMP);
"@

try {
    Write-Host "Выполнение SQL скрипта..." -ForegroundColor Cyan
    
    $command.CommandText = $sqlScript
    $result = $command.ExecuteNonQuery()
    
    Write-Host "P2P таблицы успешно созданы!" -ForegroundColor Green
    Write-Host "Количество затронутых строк: $result" -ForegroundColor Green
    
    # Проверяем созданные таблицы
    $command.CommandText = "SELECT name FROM sqlite_master WHERE type='table' AND name IN ('p2p_profile', 'contacts', 'chat_messages', 'bootstrap_peers');"
    $reader = $command.ExecuteReader()
    
    Write-Host "`nСозданные P2P таблицы:" -ForegroundColor Cyan
    while ($reader.Read()) {
        Write-Host "  - $($reader.GetString(0))" -ForegroundColor Green
    }
    $reader.Close()
    
    # Проверяем bootstrap-узлы
    $command.CommandText = "SELECT COUNT(*) FROM bootstrap_peers;"
    $count = $command.ExecuteScalar()
    Write-Host "`nДобавлено bootstrap-узлов: $count" -ForegroundColor Green
}
catch {
    Write-Host "Ошибка при выполнении SQL скрипта" -ForegroundColor Red
    Write-Host "Ошибка: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
finally {
    $connection.Close()
    $connection.Dispose()
}

Write-Host "`nГотово! P2P таблицы добавлены в базу данных." -ForegroundColor Green
