package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := "C:/Users/egors/Desktop/projectT/storage/projectT.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error opening DB: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err = db.Ping(); err != nil {
		log.Fatalf("Error pinging DB: %v", err)
	}

	fmt.Println("=== TABLE: peer_addresses ===")
	rows, err := db.Query(`SELECT id, profile_id, multiaddr, address_type, is_active, priority, source FROM peer_addresses`)
	if err != nil {
		log.Printf("Error reading peer_addresses: %v", err)
	} else {
		count := 0
		for rows.Next() {
			var id, profileID, priority int
			var multiaddr, addrType, source string
			var isActive bool
			if err := rows.Scan(&id, &profileID, &multiaddr, &addrType, &isActive, &priority, &source); err != nil {
				log.Printf("Error scanning: %v", err)
				continue
			}
			fmt.Printf("  id=%d profile_id=%d multiaddr=%s type=%s active=%v priority=%d source=%s\n",
				id, profileID, multiaddr, addrType, isActive, priority, source)
			count++
		}
		_ = rows.Close()
		if count == 0 {
			fmt.Println("  (EMPTY)")
		} else {
			fmt.Printf("  Total rows: %d\n", count)
		}
	}

	fmt.Println("\n=== TABLE: profiles ===")
	rows, err = db.Query(`SELECT id, owner_type, peer_id, username FROM profiles`)
	if err != nil {
		log.Printf("Error reading profiles: %v", err)
	} else {
		count := 0
		for rows.Next() {
			var id int
			var ownerType, peerID, username sql.NullString
			if err := rows.Scan(&id, &ownerType, &peerID, &username); err != nil {
				log.Printf("Error scanning: %v", err)
				continue
			}
			fmt.Printf("  id=%d owner_type=%s peer_id=%s username=%s\n",
				id, nullStr(ownerType), nullStr(peerID), nullStr(username))
			count++
		}
		_ = rows.Close()
		if count == 0 {
			fmt.Println("  (EMPTY)")
		} else {
			fmt.Printf("  Total rows: %d\n", count)
		}
	}

	fmt.Println("\n=== TABLE: contacts ===")
	rows, err = db.Query(`SELECT id, peer_id, multiaddr FROM contacts`)
	if err != nil {
		log.Printf("Error reading contacts: %v", err)
	} else {
		count := 0
		for rows.Next() {
			var id int
			var peerID, multiaddr sql.NullString
			if err := rows.Scan(&id, &peerID, &multiaddr); err != nil {
				log.Printf("Error scanning: %v", err)
				continue
			}
			fmt.Printf("  id=%d peer_id=%s multiaddr=%s\n",
				id, nullStr(peerID), nullStr(multiaddr))
			count++
		}
		_ = rows.Close()
		if count == 0 {
			fmt.Println("  (EMPTY)")
		} else {
			fmt.Printf("  Total rows: %d\n", count)
		}
	}
}

func nullStr(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return "NULL"
}
