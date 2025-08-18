#!/bin/bash
set -e

echo "Installing PiVoiceReminder to user home directory..."

# Check if installation directory already exists
if [ -d "$HOME/srm" ]; then
    echo "Warning: Installation directory ~/srm already exists."
    echo "This will overwrite existing files, except for the configuration files."
    read -p "Do you want to proceed? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Installation cancelled."
        exit 1
    fi
fi

# Check if service is running and stop it
echo "Checking if service is running..."
if systemctl --user is-active --quiet simple-reminder-user; then
    echo "Service is running. Stopping it..."
    systemctl --user stop simple-reminder-user
    echo "Service stopped."
else
    echo "Service is not running."
fi

# Create directories
echo "Creating directories..."
mkdir -p ~/srm/
mkdir -p ~/srm/resources
mkdir -p ~/srm/web
mkdir -p ~/.config/systemd/user

# Copy binaries
echo "Copying binaries..."
cp bin/simple-reminder ~/srm/
cp bin/hash-password ~/srm/

# Copy web files
echo "Copying web files..."
cp -r web/* ~/srm/web/


# Copy resources
echo "Copying resources..."
if [ -d "resources" ]; then
    # Check if config files exist and ask for confirmation
    config_exists=false
    if [ -f "$HOME/srm/resources/configs/config.yml" ] || [ -f "$HOME/srm/resources/configs/secrets.yml" ]; then
        config_exists=true
        echo "Warning: Configuration files already exist in ~/srm/resources/configs/"
        echo "Overwriting them will replace your current settings."
        read -p "Do you want to overwrite existing config files? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo "Copying resources..."
            cp -r resources/* ~/srm/resources/
        else
            echo "Copying resources except config files..."
            # Copy everything except configs directory
            find resources -type f ! -path "resources/configs/*" -exec cp --parents {} ~/srm/ \;
            # Create configs directory if it doesn't exist
            mkdir -p ~/srm/resources/configs
        fi
    else
        echo "No existing config files found. Copying all resources..."
        cp -r resources/* ~/srm/resources/
    fi

    if [ ! -f "$HOME/srm/resources/configs/secrets.yml" ]; then
        echo "Creating secrets.yml from template..."
        cp resources/configs/secrets.yml.template ~/srm/resources/configs/secrets.yml
        echo -e "\033[32mDefault web interface password is: admin\033[0m"
    fi
fi

# Ask user if they want to download voice models
echo ""
echo "Voice Model Download"
echo "==================="
echo "The application requires TTS (Text-to-Speech) models for voice functionality."
echo "Available models:"
echo "  - GLaDOS Voice Model (~65MB)"
echo "  - Kokoro Voice Model (~300MB)"
echo ""
read -p "Do you want to download voice models now? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Downloading voice models..."

    # Create models directory
    mkdir -p ~/srm/resources/models/tts
    cd ~/srm/resources/models/tts

    # Download GLaDOS model
    echo "Downloading GLaDOS voice model..."
    if wget -q --show-progress https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/vits-piper-en_US-glados.tar.bz2; then
        echo "Extracting GLaDOS model..."
        tar xf vits-piper-en_US-glados.tar.bz2
        rm vits-piper-en_US-glados.tar.bz2
        echo "GLaDOS model installed successfully."
    else
        echo "Warning: Failed to download GLaDOS model."
    fi

    # Download Kokoro model
    echo "Downloading Kokoro voice model..."
    if wget -q --show-progress https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-en-v0_19.tar.bz2; then
        echo "Extracting Kokoro model..."
        tar xf kokoro-en-v0_19.tar.bz2
        rm kokoro-en-v0_19.tar.bz2
        echo "Kokoro model installed successfully."
    else
        echo "Warning: Failed to download Kokoro model."
    fi

    cd - > /dev/null
    echo "Voice model download completed."
else
    echo "Voice models skipped. You can download them later by following the instructions in README.md"
fi

# Install systemd service
echo "Installing systemd user service..."
sudo cp systemd/simple-reminder-user.service /etc/systemd/user
systemctl --user daemon-reload

echo "Installation complete!"
echo ""
echo "To start the service:"
echo "  systemctl --user enable simple-reminder-user"
echo "  systemctl --user start simple-reminder-user"
echo ""
echo "For boot startup:"
echo "  sudo loginctl enable-linger $(whoami)"
echo ""
echo "To check status:"
echo "  systemctl --user status simple-reminder-user"
