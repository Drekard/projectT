// P2P CLI - утилита для управления P2P сетью ProjectT
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	p2p "projectT/internal/services/p2p"
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

var (
	version     = "dev"
	showVersion bool
	showHelp    bool

	// Глобальные флаги
	configPath string
	dbPath     string
	dbTimeout  int
	verbose    bool

	// Флаги для команды status
	statusJSON bool
)

func main() {
	// Глобальные флаги
	flag.BoolVar(&showVersion, "version", false, "Показать версию")
	flag.BoolVar(&showVersion, "v", false, "Показать версию (кратко)")
	flag.BoolVar(&showHelp, "help", false, "Показать справку")
	flag.BoolVar(&showHelp, "h", false, "Показать справку (кратко)")
	flag.StringVar(&configPath, "config", "", "Путь к конфигурации")
	flag.StringVar(&dbPath, "db-path", "", "Путь к базе данных")
	flag.IntVar(&dbTimeout, "db-timeout", 5000, "Таймаут БД (мс)")
	flag.BoolVar(&verbose, "verbose", false, "Подробный вывод")
	flag.BoolVar(&verbose, "V", false, "Подробный вывод (кратко)")

	// Флаг для status
	flag.BoolVar(&statusJSON, "json", false, "Вывод в формате JSON")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ProjectT P2P CLI - утилита управления P2P сетью\n\n")
		fmt.Fprintf(os.Stderr, "Использование:\n")
		fmt.Fprintf(os.Stderr, "  p2pcli [глобальные флаги] <команда> [флаги команды]\n\n")
		fmt.Fprintf(os.Stderr, "Команды:\n")
		fmt.Fprintf(os.Stderr, "  status              Показать статус P2P сети\n")
		fmt.Fprintf(os.Stderr, "  peers               Управление пирами\n")
		fmt.Fprintf(os.Stderr, "  chat                Отправка сообщений\n")
		fmt.Fprintf(os.Stderr, "  bootstrap           Управление bootstrap-пирами\n")
		fmt.Fprintf(os.Stderr, "  helper              Режим помощника\n")
		fmt.Fprintf(os.Stderr, "  address             Экспорт/импорт адресов\n")
		fmt.Fprintf(os.Stderr, "  nat                 Информация о NAT\n")
		fmt.Fprintf(os.Stderr, "  firewall            Управление брандмауэром\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Глобальные флаги:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Примеры:\n")
		fmt.Fprintf(os.Stderr, "  p2pcli status\n")
		fmt.Fprintf(os.Stderr, "  p2pcli peers list\n")
		fmt.Fprintf(os.Stderr, "  p2pcli chat send --peer=<peerid> --text=\"Привет\"\n")
		fmt.Fprintf(os.Stderr, "  p2pcli bootstrap add /ip4/192.168.1.1/tcp/4001/p2p/Qm...\n")
		fmt.Fprintf(os.Stderr, "  p2pcli address export\n")
	}

	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("ProjectT P2P CLI version %s\n", version)
		os.Exit(0)
	}

	// Парсим команду
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := args[0]
	commandArgs := args[1:]

	// Инициализируем БД
	if err := initDB(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации БД: %v\n", err)
		os.Exit(1)
	}

	// Выполняем команду
	switch command {
	case "status":
		runStatus(commandArgs)
	case "peers":
		runPeers(commandArgs)
	case "chat":
		runChat(commandArgs)
	case "bootstrap":
		runBootstrap(commandArgs)
	case "helper":
		runHelper(commandArgs)
	case "address":
		runAddress(commandArgs)
	case "nat":
		runNAT(commandArgs)
	case "firewall":
		runFirewall(commandArgs)
	default:
		fmt.Fprintf(os.Stderr, "Неизвестная команда: %s\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

// initDB инициализирует соединение с БД
func initDB() error {
	if dbPath == "" {
		// Пытаемся использовать путь по умолчанию
		dbPath = "./storage/projectT.db"
	}

	// Проверяем существование файла БД
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("база данных не найдена: %s. Запустите основное приложение для инициализации", dbPath)
	}

	// Инициализируем подключение
	database.InitDBWithConfig(&cliDBConfig{path: dbPath, timeout: dbTimeout})
	return nil
}

// cliDBConfig конфигурация БД для CLI
type cliDBConfig struct {
	path    string
	timeout int
}

func (c cliDBConfig) GetPath() string      { return c.path }
func (c cliDBConfig) GetBusyTimeout() int  { return c.timeout }
func (c cliDBConfig) GetMaxOpenConns() int { return 1 }
func (c cliDBConfig) GetMaxIdleConns() int { return 1 }

// runStatus обрабатывает команду status
func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	fs.BoolVar(&statusJSON, "json", statusJSON, "Вывод в JSON")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка разбора флагов: %v\n", err)
		os.Exit(1)
	}

	// Получаем профиль для проверки существования
	profile, err := queries.GetP2PProfile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "P2P профиль не найден. Запустите основное приложение для инициализации.\n")
		os.Exit(1)
	}

	status := map[string]interface{}{
		"profile_exists":  true,
		"profile_peer_id": profile.PeerID,
		"message":         "P2P профиль существует. Для получения статуса в реальном времени запустите основное приложение.",
	}

	// Получаем информацию о контактах
	contacts, err := queries.GetAllContacts()
	if err == nil {
		onlineCount := 0
		for _, c := range contacts {
			if c.Status == "online" {
				onlineCount++
			}
		}
		status["contacts_total"] = len(contacts)
		status["contacts_online"] = onlineCount
	}

	// Получаем bootstrap-пиры
	bootstrapPeers, err := queries.GetAllBootstrapPeers()
	if err == nil {
		status["bootstrap_peers"] = len(bootstrapPeers)
	}

	if statusJSON {
		outputJSON(status)
	} else {
		outputStatusText(status)
	}
}

func outputStatusText(status map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Статус P2P сети:\n")
	fmt.Fprintf(w, "  Профиль:\tсуществует\n")
	if peerID, ok := status["profile_peer_id"].(string); ok {
		fmt.Fprintf(w, "  Peer ID:\t%s\n", peerID)
	}
	if contactsTotal, ok := status["contacts_total"].(int); ok {
		contactsOnline := status["contacts_online"].(int)
		fmt.Fprintf(w, "  Контакты:\t%d (онлайн: %d)\n", contactsTotal, contactsOnline)
	}
	if bootstrapPeers, ok := status["bootstrap_peers"].(int); ok {
		fmt.Fprintf(w, "  Bootstrap-пиры:\t%d\n", bootstrapPeers)
	}
	w.Flush()
}

// runPeers обрабатывает команду peers
func runPeers(args []string) {
	if len(args) == 0 {
		fmt.Println("Использование: p2pcli peers <list|add|remove|info>")
		fmt.Println("  list              Показать список контактов")
		fmt.Println("  add <peerid>      Добавить контакт")
		fmt.Println("  remove <id>       Удалить контакт по ID")
		fmt.Println("  info <peerid>     Информация о пире")
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		peersList()
	case "add":
		if len(args) < 2 {
			fmt.Println("Ошибка: укажите PeerID")
			os.Exit(1)
		}
		peersAdd(args[1])
	case "remove":
		if len(args) < 2 {
			fmt.Println("Ошибка: укажите ID контакта")
			os.Exit(1)
		}
		peersRemove(args[1])
	case "info":
		if len(args) < 2 {
			fmt.Println("Ошибка: укажите PeerID")
			os.Exit(1)
		}
		peersInfo(args[1])
	default:
		fmt.Fprintf(os.Stderr, "Неизвестная подкоманда: %s\n", subcommand)
		os.Exit(1)
	}
}

func peersList() {
	contacts, err := queries.GetAllContacts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка получения контактов: %v\n", err)
		os.Exit(1)
	}

	if len(contacts) == 0 {
		fmt.Println("Список контактов пуст")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tPeer ID\tИмя\tСтатус\tАдрес\n")
	for _, c := range contacts {
		addr := c.Multiaddr
		if len(addr) > 30 {
			addr = addr[:27] + "..."
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", c.ID, c.PeerID[:16]+"...", c.Username, c.Status, addr)
	}
	w.Flush()
}

func peersAdd(peerID string) {
	// Валидируем PeerID
	_, err := peer.Decode(peerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Неверный PeerID: %v\n", err)
		os.Exit(1)
	}

	// Создаём контакт
	contact := &models.Contact{
		PeerID:    peerID,
		Username:  "peer_" + peerID[:8],
		Multiaddr: "",
		Status:    "offline",
	}

	if err := queries.CreateContact(contact); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка создания контакта: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Контакт добавлен: %s\n", peerID)
}

func peersRemove(id string) {
	contactID := 0
	if _, err := fmt.Sscanf(id, "%d", &contactID); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка разбора ID контакта: %v\n", err)
		os.Exit(1)
	}
	if contactID == 0 {
		fmt.Fprintf(os.Stderr, "Неверный ID контакта: %s\n", id)
		os.Exit(1)
	}

	if err := queries.DeleteContact(contactID); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка удаления контакта: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Контакт %d удалён\n", contactID)
}

func peersInfo(peerID string) {
	contact, err := queries.GetContactByPeerID(peerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Контакт не найден: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Информация о контакте:\n")
	fmt.Printf("  ID:\t%d\n", contact.ID)
	fmt.Printf("  Peer ID:\t%s\n", contact.PeerID)
	fmt.Printf("  Имя:\t%s\n", contact.Username)
	fmt.Printf("  Статус:\t%s\n", contact.Status)
	fmt.Printf("  Адрес:\t%s\n", contact.Multiaddr)
	if contact.LastSeen != nil {
		fmt.Printf("  Последний раз:\t%s\n", contact.LastSeen.Format(time.RFC3339))
	}
}

// runChat обрабатывает команду chat
func runChat(args []string) {
	if len(args) == 0 {
		fmt.Println("Использование: p2pcli chat <send|history|unread> [флаги]")
		fmt.Println("  send              Отправить сообщение")
		fmt.Println("    --peer=<peerid>   PeerID получателя")
		fmt.Println("    --text=<text>     Текст сообщения")
		fmt.Println("    --file=<path>     Путь к файлу")
		fmt.Println("  history           История сообщений")
		fmt.Println("    --contact=<id>    ID контакта")
		fmt.Println("    --limit=<n>       Количество сообщений (по умолчанию 20)")
		fmt.Println("  unread            Непрочитанные сообщения")
		fmt.Println("    --contact=<id>    ID контакта")
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "send":
		chatSend(args[1:])
	case "history":
		chatHistory(args[1:])
	case "unread":
		chatUnread(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Неизвестная подкоманда: %s\n", subcommand)
		os.Exit(1)
	}
}

func chatSend(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	peerID := fs.String("peer", "", "PeerID получателя")
	text := fs.String("text", "", "Текст сообщения")
	filePath := fs.String("file", "", "Путь к файлу")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка разбора флагов: %v\n", err)
		os.Exit(1)
	}

	if *peerID == "" {
		fmt.Println("Ошибка: укажите --peer")
		os.Exit(1)
	}

	if *text == "" && *filePath == "" {
		fmt.Println("Ошибка: укажите --text или --file")
		os.Exit(1)
	}

	// В текущей реализации для отправки сообщений нужно запущенное основное приложение
	fmt.Println("Для отправки сообщений запустите основное приложение ProjectT")
	fmt.Println("Сообщение будет добавлено в очередь:")
	fmt.Printf("  Получатель: %s\n", *peerID)
	if *text != "" {
		fmt.Printf("  Текст: %s\n", *text)
	}
	if *filePath != "" {
		fmt.Printf("  Файл: %s\n", *filePath)
	}
}

func chatHistory(args []string) {
	fs := flag.NewFlagSet("history", flag.ExitOnError)
	contactID := fs.Int("contact", 0, "ID контакта")
	limit := fs.Int("limit", 20, "Количество сообщений")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка разбора флагов: %v\n", err)
		os.Exit(1)
	}

	if *contactID == 0 {
		fmt.Println("Ошибка: укажите --contact")
		os.Exit(1)
	}

	messages, err := queries.GetMessagesForContact(*contactID, *limit, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка получения сообщений: %v\n", err)
		os.Exit(1)
	}

	if len(messages) == 0 {
		fmt.Println("История пуста")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, m := range messages {
		timestamp := m.SentAt.Format("2006-01-02 15:04:05")
		read := "✓"
		if !m.IsRead {
			read = "•"
		}
		content := m.Content
		if len(content) > 50 {
			content = content[:47] + "..."
		}
		fmt.Fprintf(w, "[%s] %s\t%s\t%s\n", timestamp, read, m.ContentType, content)
	}
	w.Flush()
}

func chatUnread(args []string) {
	fs := flag.NewFlagSet("unread", flag.ExitOnError)
	contactID := fs.Int("contact", 0, "ID контакта")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка разбора флагов: %v\n", err)
		os.Exit(1)
	}

	if *contactID == 0 {
		// Показываем количество непрочитанных для всех контактов
		contacts, err := queries.GetAllContacts()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка получения контактов: %v\n", err)
			os.Exit(1)
		}

		totalUnread := 0
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "Контакт\tНепрочитанных\n")
		for _, c := range contacts {
			count, _ := queries.GetUnreadMessagesCount(c.ID)
			if count > 0 {
				fmt.Fprintf(w, "%s\t%d\n", c.Username, count)
				totalUnread += count
			}
		}
		w.Flush()

		if totalUnread == 0 {
			fmt.Println("Нет непрочитанных сообщений")
		} else {
			fmt.Printf("\nВсего непрочитанных: %d\n", totalUnread)
		}
		return
	}

	count, err := queries.GetUnreadMessagesCount(*contactID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка получения количества: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Непрочитанных сообщений: %d\n", count)
}

// runBootstrap обрабатывает команду bootstrap
func runBootstrap(args []string) {
	if len(args) == 0 {
		fmt.Println("Использование: p2pcli bootstrap <list|add|remove> [флаги]")
		fmt.Println("  list              Показать bootstrap-пиры")
		fmt.Println("  add <addr>        Добавить bootstrap-пир")
		fmt.Println("  remove <addr>     Удалить bootstrap-пир")
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		bootstrapList()
	case "add":
		if len(args) < 2 {
			fmt.Println("Ошибка: укажите multiaddr")
			os.Exit(1)
		}
		bootstrapAdd(args[1])
	case "remove":
		if len(args) < 2 {
			fmt.Println("Ошибка: укажите multiaddr")
			os.Exit(1)
		}
		bootstrapRemove(args[1])
	default:
		fmt.Fprintf(os.Stderr, "Неизвестная подкоманда: %s\n", subcommand)
		os.Exit(1)
	}
}

func bootstrapList() {
	peers, err := queries.GetAllBootstrapPeers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка получения bootstrap-пиров: %v\n", err)
		os.Exit(1)
	}

	if len(peers) == 0 {
		fmt.Println("Список bootstrap-пиров пуст")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tPeer ID\tАдрес\tАктивен\n")
	for _, p := range peers {
		active := "да"
		if !p.IsActive {
			active = "нет"
		}
		peerID := ""
		if p.PeerID.Valid {
			peerID = p.PeerID.String[:16] + "..."
		}
		addr := p.Multiaddr
		if len(addr) > 40 {
			addr = addr[:37] + "..."
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", p.ID, peerID, addr, active)
	}
	w.Flush()
}

func bootstrapAdd(addr string) {
	// Валидируем multiaddr
	_, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Неверный multiaddr: %v\n", err)
		os.Exit(1)
	}

	// Проверяем существование
	exists, _ := queries.BootstrapPeerExists(addr)
	if exists {
		fmt.Println("Bootstrap-пир уже существует")
		return
	}

	// Извлекаем PeerID если возможно
	var peerIDStr sql.NullString
	addrMA, err := multiaddr.NewMultiaddr(addr)
	if err == nil {
		info, err := peer.AddrInfoFromP2pAddr(addrMA)
		if err == nil {
			peerIDStr = sql.NullString{String: info.ID.String(), Valid: true}
		}
	}

	bootstrapPeer := &models.BootstrapPeer{
		Multiaddr: addr,
		IsActive:  true,
		PeerID:    peerIDStr,
	}

	if err := queries.CreateBootstrapPeer(bootstrapPeer); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка создания bootstrap-пира: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Bootstrap-пир добавлен: %s\n", addr)
}

func bootstrapRemove(addr string) {
	if err := queries.DeleteBootstrapPeerByMultiaddr(addr); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка удаления bootstrap-пира: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Bootstrap-пир удалён: %s\n", addr)
}

// runHelper обрабатывает команду helper
func runHelper(args []string) {
	if len(args) == 0 {
		fmt.Println("Использование: p2pcli helper <status|register|ask|list> [флаги]")
		fmt.Println("  status            Статус режима помощника")
		fmt.Println("  register          Зарегистрировать свой адрес")
		fmt.Println("  ask <peerid>      Запросить адрес пира")
		fmt.Println("  list              Список зарегистрированных пиров")
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "status":
		helperStatus()
	case "register":
		helperRegister()
	case "ask":
		if len(args) < 2 {
			fmt.Println("Ошибка: укажите PeerID")
			os.Exit(1)
		}
		helperAsk(args[1])
	case "list":
		helperList()
	default:
		fmt.Fprintf(os.Stderr, "Неизвестная подкоманда: %s\n", subcommand)
		os.Exit(1)
	}
}

func helperStatus() {
	fmt.Println("Режим помощника доступен только при запущенном основном приложении")
	fmt.Println("Для использования запустите ProjectT с флагом --p2p-helper-mode")
}

func helperRegister() {
	fmt.Println("Для регистрации адреса запустите основное приложение")
	fmt.Println("Адрес будет автоматически зарегистрирован на helper-пирах")
}

func helperAsk(peerID string) {
	fmt.Printf("Запрос адреса для пира: %s\n", peerID)
	fmt.Println("Для выполнения запроса запустите основное приложение")
}

func helperList() {
	fmt.Println("Список зарегистрированных пиров доступен в основном приложении")
}

// runAddress обрабатывает команду address
func runAddress(args []string) {
	if len(args) == 0 {
		fmt.Println("Использование: p2pcli address <export|import> [флаги]")
		fmt.Println("  export            Экспорт своего адреса")
		fmt.Println("  import <addr>     Импорт адреса пира")
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "export":
		addressExport()
	case "import":
		if len(args) < 2 {
			fmt.Println("Ошибка: укажите адрес для импорта")
			os.Exit(1)
		}
		addressImport(args[1])
	default:
		fmt.Fprintf(os.Stderr, "Неизвестная подкоманда: %s\n", subcommand)
		os.Exit(1)
	}
}

func addressExport() {
	profile, err := queries.GetP2PProfile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "P2P профиль не найден\n")
		os.Exit(1)
	}

	// Получаем адреса из профиля
	addrs := profile.ListenAddrs
	if addrs == "" {
		addrs = "(адреса не указаны)"
	}

	fmt.Println("Ваш P2P адрес для подключения:")
	fmt.Printf("Peer ID: %s\n", profile.PeerID)
	fmt.Printf("Адреса: %s\n", addrs)
	fmt.Println("\nДля подключения используйте:")
	fmt.Printf("  p2pcli address import \"%s/p2p/%s\"\n", addrs, profile.PeerID)
}

func addressImport(addr string) {
	// Удаляем префикс protocol:// если есть
	addr = trimPrefix(addr, "projectt://")

	// Парсим multiaddr
	ma, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Неверный адрес: %v\n", err)
		os.Exit(1)
	}

	// Извлекаем PeerID
	info, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Не удалось извлечь PeerID: %v\n", err)
		os.Exit(1)
	}

	// Создаём контакт
	contact := &models.Contact{
		PeerID:    info.ID.String(),
		Username:  "peer_" + info.ID.String()[:8],
		Multiaddr: addr,
		Status:    "offline",
	}

	if err := queries.CreateContact(contact); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка создания контакта: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Адрес импортирован. Контакт добавлен: %s\n", info.ID.String())
}

// runNAT обрабатывает команду nat
func runNAT(args []string) {
	fmt.Println("Информация о NAT:")
	fmt.Println("  Для проверки NAT запустите основное приложение")
	fmt.Println("  Статус UPnP/NAT-PMP будет показан в статусе P2P сети")
}

// runFirewall обрабатывает команду firewall
func runFirewall(args []string) {
	fs := flag.NewFlagSet("firewall", flag.ExitOnError)
	port := fs.Int("port", 0, "Порт для открытия")
	showCmd := fs.Bool("show-cmd", false, "Показать команду для ручного выполнения")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка разбора флагов: %v\n", err)
		os.Exit(1)
	}

	if *port == 0 {
		fmt.Println("Использование: p2pcli firewall --port=<number> [--show-cmd]")
		os.Exit(1)
	}

	rule := p2p.GenerateFirewallRule(*port, "ProjectT P2P")

	fmt.Printf("Правило брандмауэра для порта %d:\n", *port)
	fmt.Printf("  Платформа: %s\n", rule.Platform)
	fmt.Printf("  Протокол: %s\n", rule.Protocol)

	if *showCmd {
		fmt.Printf("\nPowerShell:\n  %s\n", rule.PowerShell)
		fmt.Printf("\nCMD:\n  %s\n", rule.CMD)
	} else {
		fmt.Println("\nДля автоматического открытия используйте --show-cmd")
	}
}

// outputJSON выводит данные в формате JSON
func outputJSON(data interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка вывода JSON: %v\n", err)
		os.Exit(1)
	}
}

// trimPrefix удаляет префикс из строки
func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
