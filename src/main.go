package main

import (
	"time"

	"github.com/sirupsen/logrus"
)

const (
	logLevel       = logrus.DebugLevel
	logPath        = "resources/app.log"
	defaultConfig  = "resources/configs/config.yml"
	defaultSecrets = "resources/configs/secrets.yml"
)

var (
	SysRootDir string
)

func init() {
	// Set the system root directory
	var err error
	SysRootDir, err = getAppRootDir()
	if err != nil {
		logrus.Fatal("Failed to get root directory:", err)
	}

	// Setup logging
	setupLogging()

	// Load configuration files
	if err := loadConfig(); err != nil {
		logrus.Fatal("Failed to load configuration:", err)
	}
}

func main() {
	// Setup web server for configuration management
	setupWebServer()

	// Initialize TTS system
	if err := initSherpaTts(); err != nil {
		logrus.Fatal("Failed to initialize TTS system:", err)
	}

	// check internet connection
	for {
		if !checkInternetConnection() {
			aiSpeak("I'm sorry, but I can't access the internet right now, I will try again later.")
			time.Sleep(15 * time.Second)
			continue
		}

		logDebug("Internet connection is available.")
		aiSpeak("Hello, I'm ready to help you.")
		break
	}

	// refresh tasks periodically in background
	go refreshTasks()

	// remind pending tasks periodocally
	for {
		remindCurrentEvents()
		time.Sleep(30 * time.Second)
	}
}

func refreshTasks() {
	for {
		todayEvents := getTodayCalEvents()
		err := syncLocalEvents(todayEvents)
		if err != nil {
			logrus.Error("Failed to save events:", err)
		}

		logDebug("Refreshed tasks from calendar, for today there is %d tasks.\n", len(todayEvents))
		time.Sleep(15 * time.Second)
	}
}
