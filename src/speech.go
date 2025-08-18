package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-audio/wav"
	"github.com/hajimehoshi/oto/v2"
	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

var (
	ttsHandle *sherpa.OfflineTts
	ttsConfig sherpa.OfflineTtsConfig
)

func initSherpaTts() error {
	ttsConfig.Model.NumThreads = SysConfig.TtsConfig.Model.NumThreads
	ttsConfig.Model.Provider = SysConfig.TtsConfig.Model.Provider
	ttsConfig.MaxNumSentences = SysConfig.TtsConfig.MaxNumSentences

	if strings.ToLower(SysConfig.AiSpeechTtsConfig.TtsModel) == "kokoro" {
		ttsConfig.Model.Kokoro.Model = realPath(SysConfig.AiSpeechTtsConfig.KokoroModel)
		ttsConfig.Model.Kokoro.Voices = realPath(SysConfig.AiSpeechTtsConfig.KokoroVoices)
		ttsConfig.Model.Kokoro.Tokens = realPath(SysConfig.AiSpeechTtsConfig.KokoroTokens)
		ttsConfig.Model.Kokoro.DataDir = realPath(SysConfig.AiSpeechTtsConfig.KokoroDataDir)
		ttsConfig.Model.Kokoro.LengthScale = SysConfig.AiSpeechTtsConfig.KokoroLengthScale
		SysConfig.AiSpeechTtsConfig.Speaker = SysConfig.AiSpeechTtsConfig.KokoroSpeaker
	} else if strings.ToLower(SysConfig.AiSpeechTtsConfig.TtsModel) == "glados" {
		ttsConfig.Model.Vits.Model = realPath(SysConfig.AiSpeechTtsConfig.GladosModel)
		ttsConfig.Model.Vits.Lexicon = realPath(SysConfig.AiSpeechTtsConfig.GladosLexicon)
		ttsConfig.Model.Vits.Tokens = realPath(SysConfig.AiSpeechTtsConfig.GladosTokens)
		ttsConfig.Model.Vits.DataDir = realPath(SysConfig.AiSpeechTtsConfig.GladosDataDir)
		ttsConfig.Model.Vits.NoiseScale = 0.667
		ttsConfig.Model.Vits.NoiseScaleW = 0.8
		SysConfig.AiSpeechTtsConfig.Speaker = 0
	}

	ttsHandle = sherpa.NewOfflineTts(&ttsConfig)
	return nil
}

func aiSpeak(text string) error {
	err := sherpaSpeak(ttsHandle, text, SysConfig.AiSpeechTtsConfig.Speaker)
	if err != nil {
		return fmt.Errorf("failed to speak: %w", err)
	}
	return nil
}

func sherpaSpeak(ttsHandle *sherpa.OfflineTts, text string, ttsSpeaker int) error {
	logDebug("Generating audio for %s", text)

	audio := ttsHandle.Generate(text, ttsSpeaker, SysConfig.AiSpeechTtsConfig.Speed)
	if audio == nil {
		return fmt.Errorf("failed to generate audio")
	}

	tempFile, err := os.CreateTemp("/tmp", "speech_generated_*.wav")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	filename := tempFile.Name()
	tempFile.Close()
	defer func() {
		if err := os.Remove(filename); err != nil {
			logError("Failed to remove temporary audio file %s: %v", filename, err)
		}
	}()

	// save and play
	ok := audio.Save(filename)
	if !ok {
		return fmt.Errorf("failed to save audio")
	}

	if err := playWavFile(filename); err != nil {
		logError("Failed to play audio file %s: %v", filename, err)
		return fmt.Errorf("failed to play audio: %w", err)
	}

	return nil
}

func playWavFile(filename string) error {
	logDebug("Playing audio...")

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open WAV file: %w", err)
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return fmt.Errorf("invalid WAV file: %s", filename)
	}
	format := decoder.Format()
	sampleRate := int(format.SampleRate)
	channels := int(format.NumChannels)
	logDebug("Playing WAV file: %s (Sample Rate: %d, Channels: %d)", filename, sampleRate, channels)

	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return fmt.Errorf("failed to decode WAV file: %w", err)
	}

	// Convert audio data to bytes (16-bit PCM, little-endian)
	audioData := make([]byte, len(buf.Data)*2)
	for i, sample := range buf.Data {
		s16 := int16(sample)
		audioData[i*2] = byte(s16)
		audioData[i*2+1] = byte(s16 >> 8)
	}

	// Initialize the audio context, create a reader from the audio data, and play!
	ctx, ready, err := oto.NewContext(sampleRate, channels, oto.FormatSignedInt16LE)
	if err != nil {
		return fmt.Errorf("failed to create audio context: %w", err)
	}
	<-ready

	reader := bytes.NewReader(audioData)
	player := ctx.NewPlayer(reader)
	defer player.Close()
	player.Play()

	// Wait for playback to complete, duration is based on sample rate and data length + small buffer
	// this can be improved by using actual audio-device-specific timing information, but fine for now!
	duration := time.Duration(len(buf.Data)) * time.Second / time.Duration(sampleRate*channels)
	time.Sleep(duration + 100*time.Millisecond)

	return nil
}
