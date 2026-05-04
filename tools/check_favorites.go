package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Determine path to database
	dbPath := "./storage/projectT.db"

	// Check if database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Try absolute path
		absPath, _ := filepath.Abs(dbPath)
		log.Printf("Database file not found at: %s, trying: %s", dbPath, absPath)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			log.Fatalf("Database file not found at: %s", absPath)
		}
		dbPath = absPath
	}

	fmt.Printf("Opening database: %s\n\n", dbPath)

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Check database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	fmt.Println("=== FAVORITES TABLE STRUCTURE ===")
	rows, err := db.Query("PRAGMA table_info(favorites)")
	if err != nil {
		log.Fatalf("Error getting table info: %v", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("Column %d: %s (%s), NOT NULL: %v, PK: %v, DEFAULT: %v\n",
			cid, name, typ, notnull == 1, pk == 1, dfltValue.String)
	}

	fmt.Println("\n=== ALL FAVORITES DATA ===")
	rows2, err := db.Query("SELECT id, entity_type, entity_uuid FROM favorites ORDER BY id")
	if err != nil {
		log.Fatalf("Error querying favorites: %v", err)
	}
	defer func() { _ = rows2.Close() }()

	count := 0
	for rows2.Next() {
		var id int
		var entityType, entityUUID string
		if err := rows2.Scan(&id, &entityType, &entityUUID); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		count++
		fmt.Printf("[%d] ID=%d, Type=%s, UUID=%s\n", count, id, entityType, entityUUID)
	}

	if count == 0 {
		fmt.Println("No favorites found in database!")
	} else {
		fmt.Printf("\nTotal favorites: %d\n", count)
	}

	// Check related items
	fmt.Println("\n=== CHECKING RELATED FOLDERS (items) ===")
	rows3, err := db.Query(`
		SELECT i.id, i.element_uuid, i.type, i.title 
		FROM items i 
		INNER JOIN favorites f ON i.element_uuid = f.entity_uuid 
		WHERE f.entity_type = 'folder'
	`)
	if err != nil {
		log.Printf("Error querying favorite folders: %v", err)
	} else {
		defer func() { _ = rows3.Close() }()
		folderCount := 0
		for rows3.Next() {
			var id int
			var uuid, typ, title string
			if err := rows3.Scan(&id, &uuid, &typ, &title); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			folderCount++
			fmt.Printf("  Folder: ID=%d, UUID=%s, Title=%s\n", id, uuid, title)
		}
		if folderCount == 0 {
			fmt.Println("  No favorite folders found!")
		} else {
			fmt.Printf("  Total favorite folders: %d\n", folderCount)
		}
	}

	fmt.Println("\n=== CHECKING RELATED TAGS ===")
	rows4, err := db.Query(`
		SELECT t.id, t.tag_uuid, t.name, t.color 
		FROM tags t 
		INNER JOIN favorites f ON t.tag_uuid = f.entity_uuid 
		WHERE f.entity_type = 'tag'
	`)
	if err != nil {
		log.Printf("Error querying favorite tags: %v", err)
	} else {
		defer func() { _ = rows4.Close() }()
		tagCount := 0
		for rows4.Next() {
			var id int
			var uuid, name, color string
			if err := rows4.Scan(&id, &uuid, &name, &color); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			tagCount++
			fmt.Printf("  Tag: ID=%d, UUID=%s, Name=%s, Color=%s\n", id, uuid, name, color)
		}
		if tagCount == 0 {
			fmt.Println("  No favorite tags found!")
		} else {
			fmt.Printf("  Total favorite tags: %d\n", tagCount)
		}
	}

	fmt.Println("\n=== ALL TAGS IN DATABASE ===")
	rows5, err := db.Query(`SELECT id, tag_uuid, name, color FROM tags ORDER BY id`)
	if err != nil {
		log.Printf("Error querying all tags: %v", err)
	} else {
		defer func() { _ = rows5.Close() }()
		allTagsCount := 0
		for rows5.Next() {
			var id int
			var uuid, name, color string
			if err := rows5.Scan(&id, &uuid, &name, &color); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			allTagsCount++
			fmt.Printf("  Tag: ID=%d, UUID=%s, Name=%s, Color=%s\n", id, uuid, name, color)
		}
		fmt.Printf("\nTotal tags in database: %d\n", allTagsCount)
	}

	fmt.Println("\n=== TABLE: tags STRUCTURE ===")
	rows6, err := db.Query("PRAGMA table_info(tags)")
	if err != nil {
		log.Fatalf("Error getting tags table info: %v", err)
	}
	defer func() { _ = rows6.Close() }()

	for rows6.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows6.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("Column %d: %s (%s), NOT NULL: %v, PK: %v, DEFAULT: %v\n",
			cid, name, typ, notnull == 1, pk == 1, dfltValue.String)
	}

	fmt.Println("\n=== ALL ITEMS (FOLDERS) IN DATABASE ===")
	rows7, err := db.Query(`SELECT id, element_uuid, type, title FROM items WHERE type = 'folder' ORDER BY id`)
	if err != nil {
		log.Printf("Error querying all folders: %v", err)
	} else {
		defer func() { _ = rows7.Close() }()
		folderCount := 0
		for rows7.Next() {
			var id int
			var uuid, typ, title string
			if err := rows7.Scan(&id, &uuid, &typ, &title); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			folderCount++
			fmt.Printf("  Folder: ID=%d, UUID=%s, Title=%s\n", id, uuid, title)
		}
		fmt.Printf("\nTotal folders in database: %d\n", folderCount)
	}

	fmt.Println("\n=== DEBUG: JOIN QUERY FOR FAVORITE FOLDERS ===")
	rows8, err := db.Query(`
		SELECT f.id, f.entity_type, f.entity_uuid, i.id, i.element_uuid, i.title
		FROM favorites f
		LEFT JOIN items i ON i.element_uuid = f.entity_uuid
		WHERE f.entity_type = 'folder'
	`)
	if err != nil {
		log.Printf("Error querying join: %v", err)
	} else {
		defer func() { _ = rows8.Close() }()
		joinCount := 0
		for rows8.Next() {
			var favID, itemID int
			var entityType, entityUUID, itemUUID, title string
			if err := rows8.Scan(&favID, &entityType, &entityUUID, &itemID, &itemUUID, &title); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			joinCount++
			fmt.Printf("  Join result: fav.id=%d, entity_type=%s, entity_uuid=%s, item.id=%d, item.element_uuid=%s, item.title=%s\n",
				favID, entityType, entityUUID, itemID, itemUUID, title)
		}
		if joinCount == 0 {
			fmt.Println("  No join results - UUID mismatch!")
		} else {
			fmt.Printf("  Total join results: %d\n", joinCount)
		}
	}
}
