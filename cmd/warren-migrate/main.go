package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	bolt "go.etcd.io/bbolt"
)

var (
	dataDir    = flag.String("data-dir", "/var/lib/warren", "Warren data directory")
	dryRun     = flag.Bool("dry-run", false, "Show what would be migrated without making changes")
	backupPath = flag.String("backup", "", "Path to backup the database before migration (default: <data-dir>/warren.db.backup)")
)

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Warren Database Migration Tool - Task → Container")
	log.Println("=================================================")

	dbPath := filepath.Join(*dataDir, "warren.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("Database not found at %s", dbPath)
	}

	log.Printf("Database: %s", dbPath)
	log.Printf("Dry run: %v", *dryRun)

	// Create backup unless in dry-run mode
	if !*dryRun {
		backupFile := *backupPath
		if backupFile == "" {
			backupFile = dbPath + ".backup"
		}
		log.Printf("Creating backup: %s", backupFile)
		if err := copyFile(dbPath, backupFile); err != nil {
			log.Fatalf("Failed to create backup: %v", err)
		}
		log.Println("✓ Backup created successfully")
	}

	// Open database
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Perform migration
	if err := migrateTasksToContainers(db, *dryRun); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	if *dryRun {
		log.Println("\nDry run completed. No changes made.")
		log.Println("Run without --dry-run to perform the migration.")
	} else {
		log.Println("\n✓ Migration completed successfully!")
		log.Println("Old 'tasks' bucket has been preserved for rollback if needed.")
		log.Println("After verifying the migration, you can manually delete it using:")
		log.Printf("  bolt db rm %s tasks", dbPath)
	}
}

func migrateTasksToContainers(db *bolt.DB, dryRun bool) error {
	var taskCount int
	var migratedCount int

	// First, inspect what exists
	err := db.View(func(tx *bolt.Tx) error {
		tasksBucket := tx.Bucket([]byte("tasks"))
		if tasksBucket == nil {
			log.Println("✓ No 'tasks' bucket found - database is already using new schema")
			return nil
		}

		containersBucket := tx.Bucket([]byte("containers"))
		if containersBucket != nil {
			log.Println("⚠ Warning: Both 'tasks' and 'containers' buckets exist")
		}

		// Count tasks
		tasksBucket.ForEach(func(k, v []byte) error {
			taskCount++
			return nil
		})

		log.Printf("Found %d tasks to migrate", taskCount)
		return nil
	})

	if err != nil {
		return err
	}

	if taskCount == 0 {
		log.Println("✓ No tasks found to migrate")
		return nil
	}

	// Perform migration
	err = db.Update(func(tx *bolt.Tx) error {
		if dryRun {
			log.Println("\n[DRY RUN] Would perform the following operations:")
			log.Println("1. Create 'containers' bucket")
			log.Println("2. Copy all data from 'tasks' to 'containers'")
			log.Printf("3. Migrate %d task records", taskCount)
			log.Println("4. Preserve 'tasks' bucket for rollback")
			return nil
		}

		// Get or create containers bucket
		containersBucket, err := tx.CreateBucketIfNotExists([]byte("containers"))
		if err != nil {
			return fmt.Errorf("failed to create containers bucket: %w", err)
		}

		// Get tasks bucket
		tasksBucket := tx.Bucket([]byte("tasks"))
		if tasksBucket == nil {
			return nil // Already migrated
		}

		// Copy all data from tasks to containers
		log.Println("\nMigrating tasks to containers...")
		err = tasksBucket.ForEach(func(k, v []byte) error {
			// Validate JSON
			var data map[string]interface{}
			if err := json.Unmarshal(v, &data); err != nil {
				log.Printf("⚠ Warning: Skipping invalid JSON for key %s: %v", k, err)
				return nil
			}

			// Copy to containers bucket
			if err := containersBucket.Put(k, v); err != nil {
				return fmt.Errorf("failed to copy task %s: %w", k, err)
			}

			migratedCount++
			if migratedCount%10 == 0 {
				log.Printf("  Migrated %d/%d...", migratedCount, taskCount)
			}
			return nil
		})

		if err != nil {
			return err
		}

		log.Printf("✓ Migrated %d/%d tasks to containers", migratedCount, taskCount)
		log.Println("✓ Preserved 'tasks' bucket for rollback")

		return nil
	})

	return err
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0600)
}
