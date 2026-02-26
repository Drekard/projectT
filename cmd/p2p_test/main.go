// P2P Test Console - утилита для тестирования P2P соединений
// Запуск:
//   node1: go run cmd/p2p_test/main.go --node=1 --port=4001
//   node2: go run cmd/p2p_test/main.go --node=2 --port=4002 --connect=<адрес node1>

package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"

	"projectT/internal/services/p2p"
)

var (
	nodeID       = flag.Int("node", 1, "ID узла (1 или 2)")
	port         = flag.Int("port", 4001, "Порт для прослушивания")
	connectAddr  = flag.String("connect", "", "Адрес для подключения (multiaddr)")
	enableDHT    = flag.Bool("dht", true, "Включить DHT обнаружение")
	enableMDNS   = flag.Bool("mdns", false, "Включить mDNS обнаружение")
	bootstrapAddr = flag.String("bootstrap", "", "Bootstrap узел (multiaddr)")
)

// TestNode представляет тестовый узел
type TestNode struct {
	host       host.Host
	dht        *dht.IpfsDHT
	dhtDisc    *routing.RoutingDiscovery
	pubsub     *pubsub.PubSub
	ctx        context.Context
	cancel     context.CancelFunc
	chatTopic  *pubsub.Topic
	sub        *pubsub.Subscription
}

// ChatMessage сообщение чата
type ChatMessage struct {
	From      string `json:"from"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func main() {
	flag.Parse()

	if *nodeID != 1 && *nodeID != 2 {
		log.Fatal("node должен быть 1 или 2")
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║         P2P Test Console - ProjectT                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Создаём узел
	node, err := createTestNode()
	if err != nil {
		log.Fatalf("Ошибка создания узла: %v", err)
	}
	defer node.close()

	// Выводим информацию об узле
	printNodeInfo(node, *nodeID)

	// Подключаемся если указан адрес
	if *connectAddr != "" {
		fmt.Printf("\n📡 Подключение к: %s\n", *connectAddr)
		if err := node.connectToPeer(*connectAddr); err != nil {
			log.Printf("⚠️  Ошибка подключения: %v", err)
		} else {
			fmt.Println("✅ Подключено!")
		}
	}

	// Подключаем bootstrap если указан
	if *bootstrapAddr != "" {
		fmt.Printf("\n🌱 Подключение к bootstrap: %s\n", *bootstrapAddr)
		if err := node.connectToPeer(*bootstrapAddr); err != nil {
			log.Printf("⚠️  Ошибка подключения к bootstrap: %v", err)
		}
	}

	// Запускаем чат
	if err := node.initChat(); err != nil {
		log.Printf("⚠️  Чат не инициализирован: %v", err)
	}

	// Запускаем discovery
	if *enableDHT {
		fmt.Println("\n🔍 DHT обнаружение запущено...")
		go node.startDHTDiscovery()
	}

	// Обработчик сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Интерактивный ввод
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("Команды:")
	fmt.Println("  /peers         - показать подключённые пиры")
	fmt.Println("  /connect <addr> - подключиться к пиру")
	fmt.Println("  /discovery      - показать обнаруженные пиры")
	fmt.Println("  /help           - эта справка")
	fmt.Println("  /quit           - выход")
	fmt.Println("  иначе           - отправить сообщение в чат")
	fmt.Println(strings.Repeat("─", 60))

	go node.handleInput()

	// Ждём сигнал выхода
	<-sigChan
	fmt.Println("\n\n👋 Завершение работы...")
}

// createTestNode создаёт тестовый узел
func createTestNode() (*TestNode, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Генерируем ключи
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("ошибка генерации ключей: %w", err)
	}

	// Публичные relay-узлы libp2p
	staticRelays := []string{
		"/dnsaddr/relay.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/relay.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	}

	// Парсим relay-узлы
	var relayAddrs []peer.AddrInfo
	for _, addrStr := range staticRelays {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			continue
		}
		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			continue
		}
		relayAddrs = append(relayAddrs, *info)
	}

	// Опции хоста
	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port)),
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelayWithStaticRelays(relayAddrs),
		libp2p.EnableHolePunching(),
	}

	// Создаём хост
	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("ошибка создания хоста: %w", err)
	}

	node := &TestNode{
		host:   h,
		ctx:    ctx,
		cancel: cancel,
	}

	// Инициализируем DHT если включена
	if *enableDHT {
		if err := node.initDHT(); err != nil {
			log.Printf("⚠️  DHT не инициализирована: %v", err)
		}
	}

	// Инициализируем PubSub
	if err := node.initPubSub(); err != nil {
		log.Printf("⚠️  PubSub не инициализирована: %v", err)
	}

	// Обработчик входящих соединений
	h.SetStreamHandler(p2p.ChatProtocolID, node.handleChatStream)

	return node, nil
}

// initDHT инициализирует DHT с префиксом проекта
func (n *TestNode) initDHT() error {
	dhtOpts := []dht.Option{
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix("/" + p2p.ProtocolPrefix),
	}

	kdht, err := dht.New(n.ctx, n.host, dhtOpts...)
	if err != nil {
		return fmt.Errorf("ошибка создания DHT: %w", err)
	}

	n.dht = kdht
	n.dhtDisc = routing.NewRoutingDiscovery(kdht)

	log.Println("✅ DHT инициализирована")
	return nil
}

// initPubSub инициализирует PubSub
func (n *TestNode) initPubSub() error {
	ps, err := pubsub.NewGossipSub(n.ctx, n.host)
	if err != nil {
		return fmt.Errorf("ошибка создания PubSub: %w", err)
	}

	n.pubsub = ps
	log.Println("✅ PubSub инициализирована")
	return nil
}

// initChat инициализирует чат
func (n *TestNode) initChat() error {
	if n.pubsub == nil {
		return fmt.Errorf("PubSub не инициализирована")
	}

	topic, err := n.pubsub.Join("projectt-chat")
	if err != nil {
		return fmt.Errorf("ошибка создания топика: %w", err)
	}

	n.chatTopic = topic

	sub, err := topic.Subscribe()
	if err != nil {
		return fmt.Errorf("ошибка подписки: %w", err)
	}

	n.sub = sub

	// Запускаем обработчик сообщений
	go n.handleChatMessages()

	log.Println("✅ Чат инициализирован")
	return nil
}

// handleChatMessages обрабатывает входящие сообщения чата
func (n *TestNode) handleChatMessages() {
	for {
		msg, err := n.sub.Next(n.ctx)
		if err != nil {
			return
		}

		// Не показываем свои сообщения
		if msg.ReceivedFrom == n.host.ID() {
			continue
		}

		fmt.Printf("\n💬 [%s] %s\n", msg.ReceivedFrom.ShortString(), string(msg.Data))
		fmt.Print("> ")
	}
}

// handleChatStream обрабатывает входящий стрим чата
func (n *TestNode) handleChatStream(stream network.Stream) {
	defer stream.Close()

	buf := make([]byte, 1024)
	nBytes, err := stream.Read(buf)
	if err != nil {
		return
	}

	fmt.Printf("\n💌 Получено от %s: %s\n", stream.Conn().RemotePeer().ShortString(), string(buf[:nBytes]))
	fmt.Print("> ")
}

// handleInput обрабатывает ввод пользователя
func (n *TestNode) handleInput() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/") {
			n.handleCommand(input)
		} else {
			n.sendMessage(input)
		}
	}
}

// handleCommand обрабатывает команды
func (n *TestNode) handleCommand(input string) {
	parts := strings.SplitN(input, " ", 2)
	cmd := parts[0]

	switch cmd {
	case "/quit", "/exit", "/q":
		n.cancel()
		os.Exit(0)

	case "/peers":
		peers := n.host.Network().Peers()
		if len(peers) == 0 {
			fmt.Println("Нет подключённых пиров")
		} else {
			fmt.Printf("Подключённые пиры (%d):\n", len(peers))
			for _, p := range peers {
				status := n.host.Network().Connectedness(p)
				statusStr := "unknown"
				switch status {
				case network.Connected:
					statusStr = "connected"
				case network.NotConnected:
					statusStr = "disconnected"
				}
				fmt.Printf("  • %s [%s]\n", p.ShortString(), statusStr)
			}
		}

	case "/connect":
		if len(parts) < 2 {
			fmt.Println("Использование: /connect <multiaddr>")
			fmt.Println("Пример: /connect /ip4/127.0.0.1/tcp/4001/p2p/Qm...")
			return
		}
		addr := parts[1]
		if err := n.connectToPeer(addr); err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)
		} else {
			fmt.Println("✅ Подключено!")
		}

	case "/discovery":
		if n.dhtDisc == nil {
			fmt.Println("DHT discovery не инициализирован")
			return
		}
		fmt.Println("🔍 Поиск пиров...")
		ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
		defer cancel()

		peersChan, err := n.dhtDisc.FindPeers(ctx, p2p.ProtocolID)
		if err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)
			return
		}

		count := 0
		for p := range peersChan {
			if p.ID != n.host.ID() && len(p.Addrs) > 0 {
				fmt.Printf("  • %s\n", p.ID.ShortString())
				n.host.Peerstore().AddAddrs(p.ID, p.Addrs, time.Minute*10)
				count++
			}
		}

		if count == 0 {
			fmt.Println("  Пиров не найдено")
		}

	case "/advertise":
		if n.dhtDisc == nil {
			fmt.Println("DHT discovery не инициализирован")
			return
		}
		fmt.Println("📢 Реклама сервиса...")
		_, err := n.dhtDisc.Advertise(n.ctx, p2p.ProtocolID)
		if err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)
		} else {
			fmt.Println("✅ Сервис рекламируется в DHT")
		}

	case "/info":
		printNodeInfo(n, 0)

	case "/help":
		fmt.Println("\nКоманды:")
		fmt.Println("  /peers         - показать подключённые пиры")
		fmt.Println("  /connect <addr> - подключиться к пиру")
		fmt.Println("  /discovery      - найти пиры через DHT")
		fmt.Println("  /advertise      - рекламировать себя в DHT")
		fmt.Println("  /info           - показать информацию об узле")
		fmt.Println("  /help           - эта справка")
		fmt.Println("  /quit           - выход")
		fmt.Println()

	default:
		fmt.Printf("Неизвестная команда: %s\n", cmd)
		fmt.Println("Введите /help для справки")
	}
}

// sendMessage отправляет сообщение
func (n *TestNode) sendMessage(msg string) {
	// Отправляем в чат
	if n.chatTopic != nil {
		if err := n.chatTopic.Publish(n.ctx, []byte(msg)); err != nil {
			fmt.Printf("❌ Ошибка отправки: %v\n", err)
		}
	}

	// Отправляем подключённым пирам напрямую
	for _, p := range n.host.Network().Peers() {
		stream, err := n.host.NewStream(n.ctx, p, p2p.ChatProtocolID)
		if err != nil {
			continue
		}
		stream.Write([]byte(msg))
		stream.Close()
	}
}

// connectToPeer подключается к пиру с обработкой префикса
func (n *TestNode) connectToPeer(addrStr string) error {
	// Удаляем префикс если есть
	addrStr = strings.TrimPrefix(addrStr, p2p.ProtocolPrefix+"://")

	addr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		return fmt.Errorf("ошибка парсинга адреса: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("ошибка извлечения информации: %w", err)
	}

	ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
	defer cancel()

	if err := n.host.Connect(ctx, *info); err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}

	return nil
}

// startDHTDiscovery запускает обнаружение через DHT
func (n *TestNode) startDHTDiscovery() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			// Рекламируем себя
			_, err := n.dhtDisc.Advertise(n.ctx, p2p.ProtocolID)
			if err != nil {
				log.Printf("⚠️  Ошибка рекламы: %v", err)
			}

			// Ищем других
			ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
			peersChan, err := n.dhtDisc.FindPeers(ctx, p2p.ProtocolID)
			if err != nil {
				cancel()
				continue
			}

			for p := range peersChan {
				if p.ID != n.host.ID() && len(p.Addrs) > 0 {
					log.Printf("🔍 Обнаружен пир: %s", p.ID.ShortString())
					n.host.Peerstore().AddAddrs(p.ID, p.Addrs, time.Minute*10)
				}
			}
			cancel()
		}
	}
}

// close закрывает узел
func (n *TestNode) close() {
	n.cancel()

	if n.sub != nil {
		n.sub.Cancel()
	}

	if n.chatTopic != nil {
		n.chatTopic.Close()
	}

	if n.dht != nil {
		n.dht.Close()
	}

	if n.host != nil {
		n.host.Close()
	}
}

// printNodeInfo выводит информацию об узле с префиксом
func printNodeInfo(node *TestNode, nodeID int) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Node ID:          %d\n", nodeID)
	fmt.Printf("Peer ID:          %s\n", p2p.ProtocolPrefix+":"+node.host.ID().String())
	fmt.Println()
	fmt.Println("Адреса для подключения:")
	for _, addr := range node.host.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), node.host.ID().String())
		// Добавляем префикс
		prefixedAddr := p2p.ProtocolPrefix + "://" + fullAddr
		fmt.Printf("  %s\n", prefixedAddr)
	}
	fmt.Println("═══════════════════════════════════════════════════════════")
}
