package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

const (
	DefaultNotificationRepeats = 3
)

var (
	SysConfig   Config
	SysSecrets  Secrets
	SysMessages SystemMessages
)

type IcloudConfig struct {
	Username            string `yaml:"icloud_username"`              // Your iCloud email address
	AppSpecificPassword string `yaml:"icloud_app_specific_password"` // iCloud app-specific password
	CalDAVBaseUrl       string `yaml:"icloud_caldav_base_url"`       // iCloud CalDAV base URL
}

type Secrets struct {
	WebServerPassword string       `yaml:"web_server_password"`
	IcloudConfig      IcloudConfig `yaml:"icloud_config"`
}

type SystemMessages struct {
	// Error messages and system messages
	SpeechErrorMessages []string `yaml:"speech_error_messages"` // List of speech error messages
	SystemFailMessages  []string `yaml:"system_fail_messages"`  // List of system failure messages
	SystemErrorMessages []string `yaml:"system_error_messages"` // List of system error messages
	SystemSleepMessages []string `yaml:"system_sleep_messages"` // List of system sleep messages
	SystemWakeMessages  []string `yaml:"system_wake_messages"`  // List of system wake messages
	SystemInitMessages  []string `yaml:"system_init_messages"`  // List of system initialization messages
}

type Config struct {
	DebugLogEnabled     bool   `yaml:"debug_log_enabled"`
	EventsPath          string `yaml:"events_path"`
	NotificationRepeats int    `yaml:"notification_repeats"`

	// Message Templates
	AnnounceMessageTemplate    string `yaml:"announce_message_template"`
	AnnounceEndMessageTemplate string `yaml:"announce_end_message_template"`
	CheckStartMessageTemplate  string `yaml:"check_start_message_template"`
	RemindMessageTemplate      string `yaml:"remind_message_template"`

	// TTS Configuration
	TtsConfig TtsConfig `yaml:"tts_config"`

	// AI Speech TTS Configuration
	AiSpeechTtsConfig AiSpeechTtsConfig `yaml:"ai_speech_tts_config"`
}

type VitsConfig struct {
	NoiseScale  float32 `yaml:"noise_scale"`
	NoiseScaleW float32 `yaml:"noise_scale_w"`
	LengthScale float32 `yaml:"length_scale"`
}

type TtsModelConfig struct {
	NumThreads int        `yaml:"num_threads"`
	Provider   string     `yaml:"provider"`
	Vits       VitsConfig `yaml:"vits"`
}

type TtsConfig struct {
	Model           TtsModelConfig `yaml:"model"`
	MaxNumSentences int            `yaml:"max_num_sentences"`
}

type AiSpeechTtsConfig struct {
	Speed           float32 `yaml:"speed"`             // Speed of TTS models
	Speaker         int     `yaml:"speaker"`           // Speaker index for TTS models
	NumThreads      int     `yaml:"num_threads"`       // Number of threads for TTS models
	Provider        string  `yaml:"provider"`          // Provider for model execution
	MaxNumSentences int     `yaml:"max_num_sentences"` // Maximum number of sentences for TTS processing
	TtsModel        string  `yaml:"tts_model"`         // TTS model name, kokoro, glados, etc.

	// Glados-specific model configurations
	GladosModel   string `yaml:"glados_model"`    // Path to Glados model
	GladosDataDir string `yaml:"glados_data_dir"` // Path to espeak-ng data for Glados
	GladosTokens  string `yaml:"glados_tokens"`   // Path to tokens for Glados
	GladosLexicon string `yaml:"glados_lexicon"`  // Path to lexicon for Glados

	KokoroSpeaker     int     `yaml:"kokoro_speaker"`      // Speaker index for Kokoro
	KokoroModel       string  `yaml:"kokoro_model"`        // Path to Kokoro model
	KokoroVoices      string  `yaml:"kokoro_voices"`       // Path to voices.bin for Kokoro
	KokoroTokens      string  `yaml:"kokoro_tokens"`       // Path to tokens.txt for Kokoro
	KokoroDataDir     string  `yaml:"kokoro_data_dir"`     // Path to espeak-ng-data for Kokoro
	KokoroLengthScale float32 `yaml:"kokoro_length_scale"` // Length scale for Kokoro

	// LibriTTS-specific model configurations
	LibrittsModel   string `yaml:"libritts_model"`    // Path to LibriTTS model
	LibrittsDataDir string `yaml:"libritts_data_dir"` // Path to espeak-ng data for LibriTTS
	LibrittsTokens  string `yaml:"libritts_tokens"`   // Path to tokens for LibriTTS
	LibrittsLexicon string `yaml:"libritts_lexicon"`  // Path to lexicon for LibriTTS
}

func loadConfig() error {
	// Load main configuration
	file := openFile(realPath(defaultConfig))
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&SysConfig); err != nil {
		logError("Error decoding configuration file: %v", err)
		return err
	}

	if SysConfig.NotificationRepeats <= 0 {
		SysConfig.NotificationRepeats = DefaultNotificationRepeats
	}

	// Load secrets, with fallback to environment variables
	secretsPath := realPath(defaultSecrets)
	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		logInfo("Secrets file not found at %s", secretsPath)
	} else {
		file = openFile(secretsPath)
		defer file.Close()
		decoder = yaml.NewDecoder(file)
		if err := decoder.Decode(&SysSecrets); err != nil {
			logError("Error decoding secrets file: %v", err)
			return err
		}
	}

	// Override with environment variables if they exist
	overrideSecretsWithEnv()

	return nil
}

// overrideSecretsWithEnv overrides secrets with environment variables if they exist
func overrideSecretsWithEnv() {
	if username := os.Getenv("ICLOUD_USERNAME"); username != "" {
		SysSecrets.IcloudConfig.Username = username
	}
	if password := os.Getenv("ICLOUD_APP_PASSWORD"); password != "" {
		SysSecrets.IcloudConfig.AppSpecificPassword = password
	}
	if url := os.Getenv("ICLOUD_CALDAV_URL"); url != "" {
		SysSecrets.IcloudConfig.CalDAVBaseUrl = url
	}
	if password := os.Getenv("WEB_SERVER_PASSWORD"); password != "" {
		SysSecrets.WebServerPassword = password
	}
}
