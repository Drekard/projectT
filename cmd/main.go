package main

import (
	"flag"
	"fmt"
	"os"
	"projectT/internal/app"
)

var (
	version     = "dev"
	showVersion bool
	showHelp    bool
)

func main() {
	// Определяем флаги
	flag.BoolVar(&showVersion, "version", false, "Показать версию приложения")
	flag.BoolVar(&showVersion, "v", false, "Показать версию приложения (сокращённая форма)")
	flag.BoolVar(&showHelp, "help", false, "Показать справку")
	flag.BoolVar(&showHelp, "h", false, "Показать справку (сокращённая форма)")

	// Флаги конфигурации (определяются здесь для отображения в справке)
	flag.String("config", "", "Путь к файлу конфигурации (YAML)")
	flag.String("db-path", "", "Путь к файлу базы данных SQLite")
	flag.Int("db-timeout", 0, "Таймаут ожидания БД (мс)")
	flag.String("storage-path", "", "Путь к корневой директории хранилища")
	flag.String("storage-files-dir", "", "Поддиректория для файлов")
	flag.Bool("p2p-enabled", false, "Включить P2P режим")
	flag.Int("p2p-port", 0, "Порт для P2P соединений")
	flag.Bool("p2p-relay", false, "Использовать relay для обхода NAT")
	flag.Bool("p2p-relay-discovery", false, "Автообнаружение relay")

	// Парсим флаги
	flag.Parse()

	// Если запрошена помощь - показываем и выходим
	if showHelp {
		fmt.Println("ProjectT - Гибрид Проводника и Pinterest")
		fmt.Println()
		fmt.Println("Использование:")
		fmt.Println("  projectT [флаги]")
		fmt.Println()
		fmt.Println("Флаги:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Примеры:")
		fmt.Println("  projectT --db-path=\"D:\\Data\\projectT.db\"")
		fmt.Println("  projectT --storage-path=\"E:\\Files\" --p2p-port=5000")
		fmt.Println("  projectT --config=config.yaml")
		os.Exit(0)
	}

	// Если запрошена версия - показываем и выходим
	if showVersion {
		fmt.Printf("ProjectT version %s\n", version)
		os.Exit(0)
	}

	myApp := app.NewApp()

	myApp.Run()
}
