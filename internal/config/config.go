package config

import (
	"os"
	"path/filepath"
)

var (
	// AppDir is the application directory (.fmgo)
	AppDir string

	// LogFile is the path to the log file
	LogFile string

	// DBFile is the path to the database file
	DBFile string

	// TempDir is the directory for temporary files
	TempDir string
)

// Init initializes all paths relative to the current working directory
func Init() error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Setup app directory
	AppDir = filepath.Join(cwd, ".fmgo")
	if err := os.MkdirAll(AppDir, 0755); err != nil {
		return err
	}

	// Setup logs directory
	logsDir := filepath.Join(AppDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}
	LogFile = filepath.Join(logsDir, "fmgo.log")

	// Setup database file
	DBFile = filepath.Join(AppDir, "fmgo.db")

	// Setup temp directory
	TempDir = filepath.Join(AppDir, "temp")
	if err := os.MkdirAll(TempDir, 0755); err != nil {
		return err
	}

	return nil
}
