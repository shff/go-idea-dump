package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Migration struct {
	ID      int
	Name    string
	UpSQL   string
	DownSQL string
}

func main() {
	// Parse command-line arguments
	rollback := flag.Int("rollback", 0, "Rollback the last N migrations")
	create := flag.String("create", "", "Create a new migration with the given name")
	flag.Parse()

	if *create != "" {
		if err := createMigrationFile(*create); err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
		return
	}

	// Connect to the database
	db, err := sql.Open("sqlite3", "file:mydb.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Ensure the migration tracking table exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id INT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create schema_migrations table: %v", err)
	}

	// Load all migrations
	migrations, err := loadMigrations()
	if err != nil {
		log.Fatalf("Failed to load migrations: %v", err)
	}

	if *rollback > 0 {
		if err := rollbackMigration(db, migrations, *rollback); err != nil {
			log.Fatalf("Failed to rollback migration: %v", err)
		}
		return
	}

	// Apply each migration
	for _, migration := range migrations {
		if err := applyMigration(db, migration); err != nil {
			log.Fatalf("Failed to apply migration %d: %v", migration.ID, err)
		}
	}
}

func createMigrationFile(name string) error {
	// Generate timestamp-based filenames
	timestamp := time.Now().Format("20060102150405")
	upFile := fmt.Sprintf("migrations/%s_%s_up.sql", timestamp, name)
	downFile := fmt.Sprintf("migrations/%s_%s_down.sql", timestamp, name)

	// Create the files
	if err := os.WriteFile(upFile, []byte("-- Write your UP migration here\n"), 0644); err != nil {
		return fmt.Errorf("failed to create up migration file: %w", err)
	}
	if err := os.WriteFile(downFile, []byte("-- Write your DOWN migration here\n"), 0644); err != nil {
		return fmt.Errorf("failed to create down migration file: %w", err)
	}

	log.Printf("Created migration files:\n  %s\n  %s", upFile, downFile)
	return nil
}

func loadMigrations() ([]Migration, error) {
	// Read all embedded SQL files
	files, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Parse migration files
	migrationsMap := map[int]*Migration{}
	for _, file := range files {
		name := file.Name()
		parts := strings.SplitN(name, "_", -1)
		if len(parts) < 3 {
			continue
		}

		// Extract ID and type (up/down)
		id := parts[0]
		title := strings.Join(parts[1:len(parts)-1], "_")
		migrationType := parts[len(parts)-1]

		migrationID, err := strconv.Atoi(id)
		if err != nil {
			return nil, fmt.Errorf("invalid migration ID in file %s: %w", name, err)
		}

		// Read the SQL content
		content, err := migrationFiles.ReadFile(filepath.Join("migrations", name))
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", name, err)
		}

		// Add or update the migration
		if _, exists := migrationsMap[migrationID]; !exists {
			migrationsMap[migrationID] = &Migration{
				ID:   migrationID,
				Name: title,
			}
		}

		if migrationType == "up.sql" {
			migrationsMap[migrationID].UpSQL = string(content)
		} else if migrationType == "down.sql" {
			migrationsMap[migrationID].DownSQL = string(content)
		}
	}

	// Convert map to slice and sort by ID
	migrations := make([]Migration, 0, len(migrationsMap))
	for _, migration := range migrationsMap {
		migrations = append(migrations, *migration)
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})

	return migrations, nil
}

func applyMigration(db *sql.DB, migration Migration) error {
	// Check if the migration has already been applied
	var exists bool
	err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE id = ?)", migration.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check migration: %w", err)
	}
	if exists {
		log.Printf("Migration %d (%s) already applied", migration.ID, migration.Name)
		return nil
	}

	// Apply the migration
	log.Printf("Applying migration %d (%s)...", migration.ID, migration.Name)
	if _, err := db.Exec(migration.UpSQL); err != nil {
		return fmt.Errorf("failed to apply migration: %w", err)
	}

	// Record the migration as applied
	_, err = db.Exec("INSERT INTO schema_migrations (id, name, applied_at) VALUES (?, ?, CURRENT_TIMESTAMP)", migration.ID, migration.Name)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	log.Printf("Migration %d (%s) applied successfully", migration.ID, migration.Name)
	return nil
}

func rollbackMigration(db *sql.DB, migrations []Migration, count int) error {
	rows, err := db.Query("SELECT id FROM schema_migrations ORDER BY id DESC LIMIT ?", count)
	if err != nil {
		return fmt.Errorf("failed to fetch last migrations: %w", err)
	}
	defer rows.Close()

	var migrationIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan migration ID: %w", err)
		}
		migrationIDs = append(migrationIDs, id)
	}

	if len(migrationIDs) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	for _, id := range migrationIDs {
		var migration *Migration
		for _, m := range migrations {
			if m.ID == id {
				migration = &m
				break
			}
		}
		if migration == nil {
			return fmt.Errorf("migration %d not found", id)
		}

		log.Printf("Rolling back migration %d (%s)...", migration.ID, migration.Name)
		if _, err := db.Exec(migration.DownSQL); err != nil {
			return fmt.Errorf("failed to rollback migration %d: %w", migration.ID, err)
		}

		_, err := db.Exec("DELETE FROM schema_migrations WHERE id = ?", migration.ID)
		if err != nil {
			return fmt.Errorf("failed to delete migration record %d: %w", migration.ID, err)
		}

		log.Printf("Migration %d (%s) rolled back successfully", migration.ID, migration.Name)
	}
	return nil
}
