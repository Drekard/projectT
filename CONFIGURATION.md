# Настройка ProjectT

ProjectT поддерживает три способа конфигурации с разным приоритетом:

## Приоритет источников конфигурации

1. **Флаги командной строки** (наивысший приоритет)
2. **Переменные окружения** (Env variables)
3. **Файл конфигурации** (config.yaml)
4. **Значения по умолчанию** (низший приоритет)

Это означает, что флаги переопределяют переменные окружения, которые переопределяют файл конфигурации.

---

## 📁 Файл конфигурации (config.yaml)

Скопируйте `config.example.yaml` в `config.yaml` и отредактируйте под ваши нужды:

```yaml
# config.yaml

database:
  path: "./storage/projectT.db"
  busy_timeout: 30000
  max_open_conns: 1
  max_idle_conns: 1

storage:
  path: "./storage"
  files_dir: "files"

p2p:
  enabled: true
  port: 4000
  enable_relay: true
  enable_relay_discovery: true
```

### Где искать файл конфигурации

Приложение ищет файл в следующих местах (по порядку):
- `./config.yaml` (текущая директория)
- `./config.yml`
- `./config/config.yaml`
- `<директория_приложения>/config.yaml`
- `<директория_приложения>/config/config.yaml`

### 📍 Пути по умолчанию

**Важно:** Все относительные пути (например, `./storage`) разрешаются **относительно текущей рабочей директории**, из которой запущено приложение.

Это означает:
- При запуске `go run cmd\main.go` из папки проекта → пути будут относительно проекта
- При запуске `projectT.exe` из папки с exe → пути будут относительно этой папки
- При запуске `projectT.exe` из другой папки → пути будут относительно той папки

**Рекомендация для standalone exe:**
Для скомпилированного приложения используйте один из способов:

1. **Запускать из той же папки, где лежит exe:**
   ```
   cd C:\Apps\projectT
   projectT.exe
   ```

2. **Использовать абсолютные пути в config.yaml:**
   ```yaml
   database:
     path: "C:\Apps\projectT\storage\projectT.db"
   storage:
     path: "C:\Apps\projectT\storage"
   ```

3. **Использовать флаги при запуске:**
   ```bash
   projectT.exe --db-path="C:\Apps\projectT\storage\projectT.db" --storage-path="C:\Apps\projectT\storage"
   ```

Структура папок по умолчанию:
```
projectT.exe          ← исполняемый файл
├── config.yaml       ← файл конфигурации (опционально)
└── storage/          ← создаётся автоматически
    ├── projectT.db   ← база данных
    └── files/        ← файловое хранилище (00/, 01/, ...)
```

---

## 🖥️ Переменные окружения (Env variables)

Установите переменные окружения перед запуском приложения:

### Windows (PowerShell)
```powershell
$env:PROJECTT_DB_PATH="D:\Data\projectT.db"
$env:PROJECTT_STORAGE_PATH="E:\Files"
$env:PROJECTT_P2P_PORT="5000"
.\projectT.exe
```

### Windows (cmd)
```cmd
set PROJECTT_DB_PATH=D:\Data\projectT.db
set PROJECTT_STORAGE_PATH=E:\Files
set PROJECTT_P2P_PORT=5000
projectT.exe
```

### Linux/macOS
```bash
export PROJECTT_DB_PATH="/home/user/data/projectT.db"
export PROJECTT_STORAGE_PATH="/mnt/data/files"
export PROJECTT_P2P_PORT="5000"
./projectT
```

### Доступные переменные окружения

| Переменная | Описание | Пример |
|------------|----------|--------|
| `PROJECTT_DB_PATH` | Путь к базе данных | `D:\Data\projectT.db` |
| `PROJECTT_DB_BUSY_TIMEOUT` | Таймаут БД (мс) | `30000` |
| `PROJECTT_STORAGE_PATH` | Путь к хранилищу | `./internal/storage` |
| `PROJECTT_STORAGE_FILES_DIR` | Директория файлов | `files` |
| `PROJECTT_P2P_ENABLED` | Включить P2P | `true` / `false` |
| `PROJECTT_P2P_PORT` | Порт P2P | `4000` |
| `PROJECTT_P2P_RELAY` | Использовать relay | `true` / `false` |
| `PROJECTT_P2P_RELAY_DISCOVERY` | Автообнаружение relay | `true` / `false` |

---

## ⌨️ Флаги командной строки

Используйте флаги для быстрой настройки при запуске:

```bash
# Показать справку
projectT.exe -help

# Показать версию
projectT.exe --version

# Переопределить путь к базе данных
projectT.exe --db-path="D:\Data\projectT.db"

# Переопределить путь к хранилищу
projectT.exe --storage-path="E:\Files"

# Настроить P2P порт
projectT.exe --p2p-port=5000

# Комбинировать несколько флагов
projectT.exe --db-path="D:\DB\projectT.db" --storage-path="E:\Storage" --p2p-port=5000
```

### Доступные флаги

| Флаг | Описание | Пример |
|------|----------|--------|
| `-config` | Путь к файлу конфигурации | `-config=config.yaml` |
| `-db-path` | Путь к базе данных | `-db-path="D:\DB\projectT.db"` |
| `-db-timeout` | Таймаут БД (мс) | `-db-timeout=50000` |
| `-storage-path` | Путь к хранилищу | `-storage-path="E:\Files"` |
| `-storage-files-dir` | Директория файлов | `-storage-files-dir="content"` |
| `-p2p-enabled` | Включить P2P | `-p2p-enabled=true` |
| `-p2p-port` | Порт P2P | `-p2p-port=5000` |
| `-p2p-relay` | Использовать relay | `-p2p-relay=true` |
| `-p2p-relay-discovery` | Автообнаружение relay | `-p2p-relay-discovery=true` |
| `-version`, `-v` | Показать версию | `-version` |
| `-help`, `-h` | Показать справку | `-help` |

---

## 📊 Примеры использования

### Пример 1: Разработка

```bash
# Используем config.yaml по умолчанию
projectT.exe
```

### Пример 2: Тестирование с отдельной БД

```bash
# Создаём тестовую базу в памяти или временном файле
projectT.exe --db-path="./test.db" --storage-path="./test_storage"
```

### Пример 3: Продакшен с кастомными путями

```bash
# Windows - пути относительно расположения exe
projectT.exe

# Или указать абсолютные пути
projectT.exe --db-path="D:\ProjectT\data.db" --storage-path="D:\ProjectT\files" --p2p-port=4000

# Linux
./projectT --db-path="/var/lib/projectT/data.db" --storage-path="/mnt/storage/files" --p2p-port=4000
```

### Пример 4: Docker / Контейнер

```dockerfile
ENV PROJECTT_DB_PATH=/data/projectT.db
ENV PROJECTT_STORAGE_PATH=/data/files
ENV PROJECTT_P2P_PORT=4000

CMD ["./projectT"]
```

### Пример 5: Временное переопределение

```bash
# Используем config.yaml, но переопределяем порт P2P
projectT.exe --config=config.yaml --p2p-port=6000
```

---

## 🔧 Структура директорий

Приложение хранит данные в папке `storage`, которая создаётся в текущей рабочей директории:

```
projectT/
├── config.yaml           # Файл конфигурации (опционально)
├── projectT.exe          # Исполняемый файл
└── storage/              # Создаётся автоматически
    ├── projectT.db       # База данных
    └── files/            # Файловое хранилище
        ├── 00/
        ├── 01/
        └── ...
```

**Примечание:** Относительные пути в конфигурации разрешаются относительно текущей рабочей директории. Для скомпилированного приложения рекомендуется использовать абсолютные пути в `config.yaml` или запускать exe из его собственной папки.

---

## ❓ Частые вопросы

### Q: Где хранятся данные?

**A:** По умолчанию в папке `storage`, которая создаётся рядом с `projectT.exe`:
```
projectT.exe
└── storage/
    ├── projectT.db   ← база данных
    └── files/        ← файлы
```

### Q: Можно ли изменить расположение базы данных и хранилища?

**A:** Да, используйте флаги или переменные окружения:
```bash
projectT.exe --db-path="D:\Data\projectT.db" --storage-path="E:\Files"
```

Или укажите абсолютные пути в `config.yaml`.

### Q: Как отключить P2P?

**A:** Используйте флаг или переменную окружения:
```bash
projectT.exe --p2p-enabled=false
```

Или в `config.yaml`:
```yaml
p2p:
  enabled: false
```

### Q: Как создать резервную копию?

**A:** Скопируйте папку `storage`:
```bash
# Windows
xcopy storage backup\storage /E /I

# Linux
cp -r storage backup/
```

---

## 📝 Примечания

- Все пути автоматически нормализуются (относительные → абсолютные)
- Файл конфигурации YAML, но можно использовать и JSON для отладки
- Изменения в config.yaml применяются только после перезапуска приложения
- Переменные окружения PROJECTT_* имеют префикс `PROJECTT_`
