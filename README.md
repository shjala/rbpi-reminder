# PiVoiceReminder

A Go-based extremely simple reminder app designed for people with ADHD, helping them stay on track and manage their time. The app integrates with iCloud Calendar to load your events and provides timely reminders, including periodic notifications about how much time you have left to complete tasks. Setup is simple, just add your tasks as events in your iCloud Calendar. Features include web-based configuration management and AI-powered text-to-speech reminders.

# Why❓

I often find myself forgetting when something has already started, or how much time I have left to finish it, drifting into other tasks. Traditional calendar reminders only alert you once, and then it’s too easy to lose track.

This project was built to solve that problem: a simple voice-assisted reminder system that not only announces when a task begins, but also periodically reminds you how much time you have left, making it easier to stay on track, manage time effectively, and return focus to the task at hand.

## 🎤 Model Installation

The application requires TTS (Text-to-Speech) models for voice functionality. You can install either or both of the following models. Or you can run the installation script to automate the process. The installation process involves downloading and extracting the models from their respective sources.

### GLaDOS Voice Model

```bash
cd resources/models/tts
wget https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-piper-en_US-glados.tar.bz2
tar xvf vits-piper-en_US-glados.tar.bz2
rm vits-piper-en_US-glados.tar.bz2
```

### Kokoro Voice Model

```bash
cd resources/models/tts
wget https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-en-v0_19.tar.bz2
tar xf kokoro-en-v0_19.tar.bz2
rm kokoro-en-v0_19.tar.bz2
```

## 🚀 Quick Start

### Build the Application

**Note:** Cross-compilation is not currently supported. You must build directly on the target Raspberry Pi. Before building, install the required system dependencies:

```bash
# On Raspberry Pi (Debian/Ubuntu)
sudo apt update
sudo apt install -y libasound2-dev build-essential
```

#### Build Commands

```bash
# Build the main application
make build

# Or build manually
go build -o bin/simple-reminder src/*.go
go build -o tools/hash-password tools/hash-password.go
```

The application depends on the following shared libraries:
- `libasound.so.2` - ALSA sound library
- `libsherpa-onnx-c-api.so` - TTS engine (included with Go module)
- `libonnxruntime.so` - ONNX runtime library (included with Go module)
- Standard system libraries (libc, libm, libpthread, etc.)

### Installation

```bash
# Install the application, voice models, and dependencies
make install

# Or install manually
./install.sh
```

### Access Web Interface

Navigate to <http://localhost:8080> and log in with your password. The web interface provides a very simple interface to manage the configurations and logs.

## ⚙️ Configuration

- `resources/configs/config.yml`  - Application settings
- `resources/configs/secrets.yml` - Credentials
