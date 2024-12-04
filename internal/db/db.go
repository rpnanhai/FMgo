package db

import (
	"FMgo/internal/config"
	"database/sql"
	"fmt"
	"log"
	"time"

	"FMgo/internal/model"
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func New() (*Database, error) {
	db, err := sql.Open("sqlite3", config.DBFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// 创建历史记录表
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			radio_name TEXT NOT NULL,
			play_url TEXT NOT NULL,
			played_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return nil, fmt.Errorf("failed to create history table: %v", err)
	}

	// 创建收藏表
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS favorites (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			radio_name TEXT NOT NULL UNIQUE,
			play_url TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return nil, fmt.Errorf("failed to create favorites table: %v", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) AddHistory(radio model.Radio) error {
	query := `
	INSERT INTO history (radio_name, play_url, played_at)
	VALUES (?, ?, ?)`

	_, err := d.db.Exec(query, radio.Name, radio.PlayURL, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add history: %v", err)
	}

	return nil
}

func (d *Database) GetHistory(limit int) ([]model.PlayHistory, error) {
	query := `
	SELECT id, radio_name, play_url, played_at
	FROM history
	ORDER BY played_at DESC
	LIMIT ?`

	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %v", err)
	}
	defer rows.Close()

	var history []model.PlayHistory
	for rows.Next() {
		var h model.PlayHistory
		err := rows.Scan(&h.ID, &h.RadioName, &h.PlayURL, &h.PlayedAt)
		if err != nil {
			log.Printf("Error scanning history row: %v", err)
			continue
		}
		history = append(history, h)
	}

	return history, nil
}

// AddFavorite 添加收藏
func (d *Database) AddFavorite(radio model.Radio) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO favorites (radio_name, play_url)
		VALUES (?, ?)
	`, radio.Name, radio.PlayURL)
	return err
}

// RemoveFavorite 移除收藏
func (d *Database) RemoveFavorite(radioName string) error {
	_, err := d.db.Exec(`
		DELETE FROM favorites
		WHERE radio_name = ?
	`, radioName)
	return err
}

// IsFavorite 检查是否已收藏
func (d *Database) IsFavorite(radioName string) (bool, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(*) FROM favorites
		WHERE radio_name = ?
	`, radioName).Scan(&count)
	return count > 0, err
}

// GetFavorites 获取收藏列表
func (d *Database) GetFavorites() ([]model.Radio, error) {
	rows, err := d.db.Query(`
		SELECT radio_name, play_url FROM favorites
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favorites []model.Radio
	for rows.Next() {
		var radio model.Radio
		if err := rows.Scan(&radio.Name, &radio.PlayURL); err != nil {
			return nil, err
		}
		favorites = append(favorites, radio)
	}
	return favorites, nil
}
