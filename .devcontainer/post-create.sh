#!/bin/bash
set -e

echo "Setting up workspace permissions..."
sudo chown vscode:vscode /workspace

echo "Installing Go development tools..."
cd /workspace/web-console-backend
make setup

echo "Installing global npm packages..."
npm install -g @google/gemini-cli @openai/codex

echo "Setup completed successfully!"
