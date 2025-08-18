package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"time"
)

func startOfDay(t time.Time) time.Time {
	return t.Truncate(24 * time.Hour)
}

func endOfDay(t time.Time) time.Time {
	return startOfDay(t).AddDate(0, 0, 1).Add(-time.Nanosecond)
}

func realPath(path string) string {
	return filepath.Join(SysRootDir, path)
}

func checkInternetConnection() bool {
	// Google's public DNS server IP
	const googleDNS = "8.8.8.8:53"
	timeout := 5 * time.Second

	conn, err := net.DialTimeout("udp", googleDNS, timeout)
	if err != nil {
		logError("No internet connection: %v", err)
		return false
	}
	defer conn.Close()
	return true
}

func getAppRootDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}

	appBundleDir := filepath.Dir(exePath)
	// if this is a MacOS app bundle, go up one more level
	if filepath.Base(appBundleDir) == "MacOS" {
		appBundleDir = filepath.Dir(appBundleDir)
	}

	return appBundleDir, nil
}

func openFile(path string) *os.File {
	fmt.Printf("Opening configuration file: %s\n", path)
	file, err := os.Open(path)
	if err != nil {
		logError("Error opening configuration file: %v", err)
		return nil
	}

	return file
}

// formatDuration is hHelper function to format duration in a human-readable way
func formatDuration(duration time.Duration) string {
	minutes := int(duration.Minutes())
	hours := minutes / 60
	minutes = minutes % 60

	if hours == 0 {
		return fmt.Sprintf("%d minutes", minutes)
	} else if minutes == 0 {
		return fmt.Sprintf("%d hour", hours)
	} else {
		return fmt.Sprintf("%d hour and %d minutes", hours, minutes)
	}
}

func writeFileAtomically(path string, data []byte) error {
	tempPath := fmt.Sprintf("%s.tmp.%d", path, rand.Int63())
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary secrets file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		if rmErr := os.Remove(tempPath); rmErr != nil {
			fmt.Printf("failed to remove temp file %s: %v\n", tempPath, rmErr)
		}
		return fmt.Errorf("failed to save secrets file: %w", err)
	}

	return nil
}
