#!/bin/bash

set -e

echo "Installing EdgeTTS dependencies..."

if command -v pip3 &> /dev/null; then
    pip3 install edge-tts
elif command -v pip &> /dev/null; then
    pip install edge-tts
else
    echo "pip not found. Installing python3-pip..."
    apt-get update
    apt-get install -y python3-pip
    pip3 install edge-tts
fi

echo "Verifying installation..."
edge-tts --version

echo "EdgeTTS installed successfully!"
