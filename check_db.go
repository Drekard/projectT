//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := `C:\Users\egors\Desktop\projectT\storage\projectT.db`

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Ошибка открытия БД:", err)
	}
	defer db.Close()

	fmt.Println("=== Таблица favorites ===")
	var sqlStr sql.NullString
	err = db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='favorites'`).Scan(&sqlStr)
	if err != nil {
		fmt.Println("Ошибка:", err)
	} else if sqlStr.Valid {
		fmt.Println(sqlStr.String)
	}

	fmt.Println("\n=== Триггеры на favorites ===")
	rows, err := db.Query(`SELECT name, sql FROM sqlite_master WHERE type='trigger' AND tbl_name='favorites'`)
	if err != nil {
		fmt.Println("Ошибка:", err)
	} else {
		defer rows.Close()
		found := false
		for rows.Next() {
			found = true
			var name string
			var sqlStr sql.NullString
			err := rows.Scan(&name, &sqlStr)
			if err != nil {
				fmt.Printf("  Ошибка: %v\n", err)
			} else if sqlStr.Valid {
				fmt.Printf("  %s:\n%s\n\n", name, sqlStr.String)
			}
		}
		if !found {
			fmt.Println("  (нет триггеров)")
		}
	}

	fmt.Println("\n=== Все CHECK ограничения ===")
	rows, err = db.Query(`SELECT name, sql FROM sqlite_master WHERE type='table' AND sql LIKE '%CHECK%'`)
	if err != nil {
		fmt.Println("Ошибка:", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var name string
			var sqlStr sql.NullString
			err := rows.Scan(&name, &sqlStr)
			if err != nil {
				fmt.Printf("  Ошибка: %v\n", err)
			} else if sqlStr.Valid {
				fmt.Printf("  %s:\n%s\n\n", name, sqlStr.String)
			}
		}
	}
}
