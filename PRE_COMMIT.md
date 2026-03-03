# Pre-commit хуки для ProjectT

## 📋 Что такое pre-commit?

**Pre-commit** — это фреймворк для управления pre-commit хуками в Git. Он позволяет запускать автоматические проверки кода перед каждым коммитом. Если проверки не проходят — коммит блокируется.

## 🔧 Что проверяется перед коммитом?

В этом проекте настроены следующие проверки:

### Перед коммитом (`pre-commit`):

1. **`go mod tidy`** — приводит зависимости в `go.mod` в порядок
2. **`gofmt`** — форматирует Go код согласно стандартам
3. **`golangci-lint`** — запускает линтер для поиска ошибок и code smells
4. **`go build`** — проверяет, что проект собирается без ошибок

### Перед отправкой в репозиторий (`pre-push`):

5. **`go test`** — запускает все тесты (push блокируется при провале тестов)

> **💡 Почему тесты на pre-push?** Тесты могут занимать много времени. Чтобы не замедлять локальные коммиты, они запускаются только перед отправкой кода в удалённый репозиторий.

## 🚀 Установка

### Шаг 1: Установи pre-commit

#### Windows (через pip)
```powershell
pip install pre-commit
```

#### Windows (через winget)
```powershell
winget install pre-commit.pre-commit
```

#### Linux/macOS
```bash
pip install pre-commit
# или
brew install pre-commit
```

### Шаг 2: Установи golangci-lint (если ещё не установлен)

#### Windows
```powershell
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

#### Linux/macOS
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest
```

### Шаг 3: Инициализируй pre-commit в проекте
```bash
pre-commit install
```

Это создаст Git hook в `.git/hooks/pre-commit`, который будет запускаться перед каждым коммитом.

## 📝 Использование

### Автоматическая проверка при коммите

После установки просто делай коммит как обычно:
```bash
git commit -m "feat: добавил новую фичу"
```

Pre-commit автоматически запустит все проверки. Если что-то не пройдёт — коммит будет отклонён.

### Ручной запуск проверок

Проверить все файлы:
```bash
pre-commit run --all-files
```

Проверить только изменённые файлы:
```bash
pre-commit run
```

Запустить конкретный хук:
```bash
pre-commit run go-test --all-files
```

### Обновление pre-commit

```bash
pre-commit autoupdate
```

## ⚙️ Пропуск проверок

### Временный пропуск всех хуков
```bash
git commit -m "commit message" --no-verify
```

### Пропуск конкретного хука
Добавь в коммит:
```bash
# SKIP=go-test git commit -m "commit message"
```

## 🛠️ Отключение pre-commit

```bash
pre-commit uninstall
```

Или удали файл `.git/hooks/pre-commit`

## 📄 Файл конфигурации

Конфигурация находится в `.pre-commit-config.yaml`. Пример добавления нового хука:

```yaml
- repo: local
  hooks:
    - id: my-custom-hook
      name: My Custom Check
      entry: my-command
      language: system
      types: [go]
      stages: [commit]
```

## 🐛 Решение проблем

### Ошибка: `golangci-lint: command not found`

Убедись, что golangci-lint установлен и путь к нему добавлен в PATH:

**Windows (PowerShell):**
```powershell
$env:Path += ";$(go env GOPATH)\bin"
```

**Linux/macOS:**
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Ошибка: тесты не проходят

Исправь падающие тесты или временно пропусти хук тестов:
```bash
# SKIP=go-test git commit -m "WIP: fixing tests"
```

### Хук завис или работает слишком долго

Увеличь таймаут в `.pre-commit-config.yaml`:
```yaml
- id: go-test
  timeout: 300  # 5 минут вместо стандартных 60 секунд
```

## 📊 Интеграция с CI/CD

Pre-commit дублирует проверки из GitHub Actions (`.github/workflows/ci.yml`):
- ✅ Тесты
- ✅ Линтер
- ✅ Сборка

Это позволяет отловить ошибки **до** отправки кода в репозиторий.

---

**💡 Совет:** Используй pre-commit вместе с расширениями для редактора кода (Go extension для VS Code), чтобы форматирование и линтинг работали ещё на этапе написания кода.
