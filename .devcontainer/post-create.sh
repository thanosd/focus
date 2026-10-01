#!/bin/bash

# Make sure we're in the workspace directory
cd /workspace || exit

# Install Go dependencies and tooling
if [[ -f "go.mod" ]]; then
	echo "Installing Go dependencies..."
	go mod download
	go install golang.org/x/tools/gopls@latest
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install github.com/air-verse/air@latest
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
fi

# Install frontend Node.js dependencies
if [[ -d "frontend" ]] && [[ -f "frontend/package.json" ]]; then
	echo "Installing frontend dependencies..."
	cd frontend || exit
	npm install
	cd ..
fi

# Set up git configuration (if not already set)
git_user_name=$(git config --global user.name)
if [[ -z ${git_user_name} ]]; then
	git config --global user.name "Developer"
	git config --global user.email "developer@example.com"
fi

echo "Development environment setup complete!"
echo "Next steps:"
echo "1. Copy .env.sample to backend/.env and fill in Google OAuth + Claude keys"
echo "2. make dc-migrate"
echo "3. make dc-backend / make dc-frontend"
