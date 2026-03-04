// P2P Diagnostic Tool - утилита для диагностики P2P соединений с подробным логированием
// Запуск:
//   ПК 1: go run cmd/p2p_diag/main.go --mode=host --port=4001
//   ПК 2: go run cmd/p2p_diag/main.go --mode=client --port=4001 --target=<адрес ПК 1>

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
	mode       = flag.String("mode", "host", "Режим: host (сервер) или client (клиент)")
	port       = flag.Int("port", 4001, "Порт для прослушивания")
	targetAddr = flag.String("target", "", "Адрес цели для подключения (только client)")
	enableDHT  = flag.Bool("dht", true, "Включить DHT обнаружение")
	verbose    = flag.Bool("verbose", true, "Подробное логирование")
	advertise  = flag.Bool("advertise", false, "Рекламировать себя в DHT")
	findPeers  = flag.Bool("find", false, "Искать пиры через DHT")
)

// DiagNode диагностический узел
type DiagNode struct {
	host    host.Host
	dht     *dht.IpfsDHT
	dhtDisc *routing.RoutingDiscovery
	pubsub  *pubsub.PubSub
	ctx     context.Context
	cancel  context.CancelFunc
	events  chan string // канал событий для логирования
}

func main() {
	flag.Parse()

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║      P2P Diagnostic Tool - ProjectT                       ║")
	fmt.Println("║         Диагностика соединений с логом                    ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Создаём узел
	node, err := createDiagNode()
	if err != nil {
		log.Fatalf("❌ Ошибка создания узла: %v", err)
	}
	defer node.close()

	// Выводим информацию
	printDetailedInfo(node)

	// Запускаем логирование событий
	go node.logEvents()

	// Подключаемся к цели если клиент
	if *mode == "client" && *targetAddr != "" {
		node.log("🎯 Режим КЛИЕНТ - попытка подключения к цели")
		if err := node.connectWithDiagnostics(*targetAddr); err != nil {
			node.log(fmt.Sprintf("❌ ОШИБКА ПОДКЛЮЧЕНИЯ: %v", err))
			node.log("📋 Возможные причины:")
			node.log("   1. Брандмауэр блокирует порт")
			node.log("   2. Неверный адрес")
			node.log("   3. Целевой узел не запущен")
			node.log("   4. NAT не пропускает соединение")
		} else {
			node.log("✅ ПОДКЛЮЧЕНИЕ УСПЕШНО!")
		}
	} else if *mode == "host" {
		node.log("🏠 Режим СЕРВЕР - ожидание подключений")
		node.log("📋 Сообщите этот адрес клиенту:")
		for _, addr := range node.host.Addrs() {
			fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), node.host.ID().String())
			prefixedAddr := p2p.ProtocolPrefix + "://" + fullAddr
			node.log(fmt.Sprintf("   %s", prefixedAddr))
		}
	}

	// DHT операции
	if *enableDHT && node.dhtDisc != nil {
		node.log("🔍 DHT обнаружение включено")

		if *advertise {
			node.log("📢 Реклама себя в DHT...")
			go node.advertiseInDHT()
		}

		if *findPeers || *mode == "client" {
			node.log("🔎 Поиск пиров через DHT...")
			go node.findPeersInDHT()
		}
	}

	// Настраиваем обработчик событий сети
	node.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) {
			node.events <- fmt.Sprintf("✅ ПОДКЛЮЧЕНИЕ: %s (статус: %s)",
				conn.RemotePeer().ShortString(),
				conn.Stat().Direction)
		},
		DisconnectedF: func(n network.Network, conn network.Conn) {
			node.events <- fmt.Sprintf("❌ ОТКЛЮЧЕНИЕ: %s", conn.RemotePeer().ShortString())
		},
	})

	// Обработчик сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Интерактивный ввод
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("Команды:")
	fmt.Println("  /peers        - показать подключённые пиры")
	fmt.Println("  /connect <addr> - подключиться к пиру")
	fmt.Println("  /advertise    - рекламировать себя в DHT")
	fmt.Println("  /find         - найти пиры через DHT")
	fmt.Println("  /info         - показать информацию об узле")
	fmt.Println("  /log <text>   - записать сообщение в лог")
	fmt.Println("  /quit         - выход")
	fmt.Println(strings.Repeat("─", 60))

	go node.handleInput()

	node.log("📊 Логирование событий запущено...")

	// Ждём сигнал выхода
	<-sigChan
	fmt.Println("\n\n👋 Завершение работы...")
}

// createDiagNode создаёт диагностический узел
func createDiagNode() (*DiagNode, error) {
	ctx, cancel := context.WithCancel(context.Background())

	node := &DiagNode{
		ctx:    ctx,
		cancel: cancel,
		events: make(chan string, 100),
	}

	// Генерируем ключи
	node.log("🔑 Генерация ключей...")
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("ошибка генерации ключей: %w", err)
	}
	node.log("✅ Ключи сгенерированы")

	// Публичные relay-узлы для bootstrap
	staticRelays := []string{
		"/dnsaddr/relay.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/relay.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	}

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

	if len(relayAddrs) > 0 {
		node.log(fmt.Sprintf("📡 Relay узлы: %d доступно", len(relayAddrs)))
	}

	// Опции хоста
	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port)),
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelayWithStaticRelays(relayAddrs),
		libp2p.EnableHolePunching(),
		libp2p.UserAgent("ProjectT-Diag/1.0"),
	}

	// Создаём хост
	node.log("🚀 Создание libp2p хоста...")
	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("ошибка создания хоста: %w", err)
	}
	node.host = h
	node.log(fmt.Sprintf("✅ Хост создан: %s", h.ID().ShortString()))

	// Инициализируем DHT
	if *enableDHT {
		node.log("📦 Инициализация DHT...")
		if err := node.initDHT(); err != nil {
			node.log(fmt.Sprintf("⚠️  DHT не инициализирована: %v", err))
		} else {
			node.log("✅ DHT инициализирована")
		}
	}

	// Инициализируем PubSub
	node.log("💬 Инициализация PubSub...")
	if err := node.initPubSub(); err != nil {
		node.log(fmt.Sprintf("⚠️  PubSub не инициализирована: %v", err))
	} else {
		node.log("✅ PubSub инициализирована")
	}

	return node, nil
}

// initDHT инициализирует DHT
func (n *DiagNode) initDHT() error {
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

	// Bootstrap DHT
	if err := kdht.Bootstrap(n.ctx); err != nil {
		return fmt.Errorf("ошибка bootstrap DHT: %w", err)
	}

	return nil
}

// initPubSub инициализирует PubSub
func (n *DiagNode) initPubSub() error {
	ps, err := pubsub.NewGossipSub(n.ctx, n.host)
	if err != nil {
		return fmt.Errorf("ошибка создания PubSub: %w", err)
	}
	n.pubsub = ps
	return nil
}

// connectWithDiagnostics подключается с подробной диагностикой
func (n *DiagNode) connectWithDiagnostics(addrStr string) error {
	n.log("📍 Шаг 1: Парсинг адреса...")
	addrStr = strings.TrimPrefix(addrStr, p2p.ProtocolPrefix+"://")

	addr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		n.log(fmt.Sprintf("   ❌ Ошибка парсинга multiaddr: %v", err))
		return fmt.Errorf("ошибка парсинга адреса: %w", err)
	}
	n.log("   ✅ Адрес распарсен")

	n.log("📍 Шаг 2: Извлечение PeerID...")
	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		n.log(fmt.Sprintf("   ❌ Ошибка извлечения PeerID: %v", err))
		return fmt.Errorf("ошибка извлечения PeerID: %w", err)
	}
	n.log(fmt.Sprintf("   ✅ PeerID: %s", info.ID.ShortString()))
	n.log(fmt.Sprintf("   ✅ Адреса: %v", info.Addrs))

	n.log("📍 Шаг 3: Добавление в Peerstore...")
	n.host.Peerstore().AddAddr(info.ID, info.Addrs[0], time.Minute*5)
	n.log("   ✅ Добавлен в Peerstore")

	n.log("📍 Шаг 4: Попытка подключения...")
	n.log("   ⏳ Таймаут: 30 секунд")

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
	defer cancel()

	n.log(fmt.Sprintf("   🕐 Начало подключения: %s", startTime.Format("15:04:05")))

	err = n.host.Connect(ctx, *info)
	elapsed := time.Since(startTime)

	if err != nil {
		n.log(fmt.Sprintf("   ❌ ОШИБКА ПОДКЛЮЧЕНИЯ (%.2f сек): %v", elapsed.Seconds(), err))

		// Детальная диагностика ошибки
		if strings.Contains(err.Error(), "deadline") {
			n.log("   📋 Причина: истёк таймаут - узел недоступен или блокируется")
		} else if strings.Contains(err.Error(), "connection refused") {
			n.log("   📋 Причина: соединение отклонено - порт закрыт")
		} else if strings.Contains(err.Error(), "no addresses") {
			n.log("   📋 Причина: нет адресов для подключения")
		}

		return err
	}

	n.log(fmt.Sprintf("   ✅ ПОДКЛЮЧЕНИЕ УСПЕШНО (%.2f сек)", elapsed.Seconds()))

	// Проверка соединения
	n.log("📍 Шаг 5: Проверка статуса соединения...")
	connStatus := n.host.Network().Connectedness(info.ID)
	n.log(fmt.Sprintf("   📊 Статус: %s", connStatus))

	if connStatus == network.Connected {
		n.log("   ✅ Соединение активно")
	} else {
		n.log("   ⚠️  Соединение не активно")
	}

	return nil
}

// advertiseInDHT рекламирует себя в DHT
func (n *DiagNode) advertiseInDHT() {
	n.log("📢 Начало рекламы в DHT...")

	ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
	defer cancel()

	startTime := time.Now()
	_, err := n.dhtDisc.Advertise(ctx, p2p.ProtocolID)
	elapsed := time.Since(startTime)

	if err != nil {
		n.log(fmt.Sprintf("   ❌ ОШИБКА РЕКЛАМЫ (%.2f сек): %v", elapsed.Seconds(), err))
		return
	}

	n.log(fmt.Sprintf("   ✅ РЕКЛАМА УСПЕШНА (%.2f сек)", elapsed.Seconds()))
	n.log("   📋 Другие узлы могут найти вас через /find")
}

// findPeersInDHT ищет пиры в DHT
func (n *DiagNode) findPeersInDHT() {
	n.log("🔎 Начало поиска пиров в DHT...")

	ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
	defer cancel()

	startTime := time.Now()
	peersChan, err := n.dhtDisc.FindPeers(ctx, p2p.ProtocolID)
	if err != nil {
		n.log(fmt.Sprintf("   ❌ ОШИБКА ПОИСКА: %v", err))
		return
	}

	foundCount := 0
	for p := range peersChan {
		if p.ID != n.host.ID() && len(p.Addrs) > 0 {
			foundCount++
			n.log(fmt.Sprintf("   🎯 Найден пир #%d: %s", foundCount, p.ID.ShortString()))
			n.log(fmt.Sprintf("      Адреса: %v", p.Addrs))

			// Добавляем в Peerstore
			n.host.Peerstore().AddAddrs(p.ID, p.Addrs, time.Minute*10)
			n.log("      ✅ Добавлен в Peerstore")

			// Пытаемся подключиться
			n.log("      📡 Попытка подключения...")
			connectCtx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
			err := n.host.Connect(connectCtx, p)
			cancel()

			if err != nil {
				n.log(fmt.Sprintf("      ❌ Не удалось подключиться: %v", err))
			} else {
				n.log("      ✅ ПОДКЛЮЧЕНИЕ УСПЕШНО!")
			}
		}
	}

	elapsed := time.Since(startTime)
	n.log(fmt.Sprintf("📊 Поиск завершён (%.2f сек) - найдено пиров: %d", elapsed.Seconds(), foundCount))
}

// logEvents логирует события из канала
func (n *DiagNode) logEvents() {
	for event := range n.events {
		n.log(event)
	}
}

// log записывает сообщение в лог
func (n *DiagNode) log(message string) {
	if *verbose {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[%s] %s\n", timestamp, message)
	}
}

// handleInput обрабатывает ввод пользователя
func (n *DiagNode) handleInput() {
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
			n.log(fmt.Sprintf("💬 Сообщение: %s", input))
		}
	}
}

// handleCommand обрабатывает команды
func (n *DiagNode) handleCommand(input string) {
	parts := strings.SplitN(input, " ", 2)
	cmd := parts[0]

	switch cmd {
	case "/quit", "/exit", "/q":
		n.cancel()
		os.Exit(0)

	case "/peers":
		peers := n.host.Network().Peers()
		n.log(fmt.Sprintf("📊 Подключённые пиры: %d", len(peers)))
		for _, p := range peers {
			status := n.host.Network().Connectedness(p)
			statusStr := "unknown"
			switch status {
			case network.Connected:
				statusStr = "connected"
			case network.NotConnected:
				statusStr = "disconnected"
			}
			n.log(fmt.Sprintf("   • %s [%s]", p.ShortString(), statusStr))

			// Показываем адреса
			addrs := n.host.Peerstore().Addrs(p)
			for _, addr := range addrs {
				n.log(fmt.Sprintf("      %s", addr.String()))
			}
		}

	case "/connect":
		if len(parts) < 2 {
			n.log("Использование: /connect <multiaddr>")
			return
		}
		addr := parts[1]
		if err := n.connectWithDiagnostics(addr); err != nil {
			n.log(fmt.Sprintf("❌ Ошибка: %v", err))
		}

	case "/advertise":
		go n.advertiseInDHT()

	case "/find":
		go n.findPeersInDHT()

	case "/info":
		printDetailedInfo(n)

	case "/log":
		if len(parts) < 2 {
			n.log("Использование: /log <текст>")
			return
		}
		n.log(parts[1])

	case "/help":
		n.log("Команды:")
		n.log("  /peers        - показать подключённые пиры")
		n.log("  /connect <addr> - подключиться к пиру")
		n.log("  /advertise    - рекламировать себя в DHT")
		n.log("  /find         - найти пиры через DHT")
		n.log("  /info         - показать информацию об узле")
		n.log("  /log <text>   - записать сообщение в лог")
		n.log("  /quit         - выход")

	default:
		n.log(fmt.Sprintf("Неизвестная команда: %s", cmd))
		n.log("Введите /help для справки")
	}
}

// printDetailedInfo выводит подробную информацию
func printDetailedInfo(node *DiagNode) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Peer ID:          %s\n", node.host.ID().String())
	fmt.Println()
	fmt.Println("Адреса для подключения:")
	for _, addr := range node.host.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), node.host.ID().String())
		prefixedAddr := p2p.ProtocolPrefix + "://" + fullAddr
		fmt.Printf("  %s\n", prefixedAddr)
	}
	fmt.Println()
	fmt.Println("Сетевая информация:")
	fmt.Printf("  NAT Port Map:     включено\n")
	fmt.Printf("  Relay:            включено\n")
	fmt.Printf("  Hole Punching:    включено\n")
	fmt.Printf("  DHT:              %v\n", *enableDHT)
	fmt.Println("═══════════════════════════════════════════════════════════")
}

// close закрывает узел
func (n *DiagNode) close() {
	n.cancel()

	if n.dht != nil {
		n.dht.Close()
	}

	if n.host != nil {
		n.host.Close()
	}

	close(n.events)
}
