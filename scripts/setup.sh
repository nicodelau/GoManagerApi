#!/bin/bash

# GoManager Quick Setup Script
# Sets up the environment and runs the first migration

echo "==============================================="
echo "  GoManager Quick Setup"
echo "==============================================="

# Create necessary directories
mkdir -p data storage/.avatars scripts

echo "✅ Created directories"

# Copy environment template if .env doesn't exist
if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        cp .env.example .env
        echo "✅ Created .env from template"
        echo "⚠️  Please edit .env with your actual configuration"
    else
        echo "⚠️  .env.example not found - please create .env manually"
    fi
else
    echo "✅ .env already exists"
fi

# Install Go dependencies
echo "📦 Installing dependencies..."
go mod tidy
echo "✅ Dependencies installed"

# Build the application
echo "🔨 Building application..."
go build -o gomanager
echo "✅ Application built successfully"

echo ""
echo "==============================================="
echo "🎉 Setup completed!"
echo "==============================================="
echo ""
echo "Next steps:"
echo "1. Edit .env with your database and API credentials"
echo "2. For PostgreSQL: Set DATABASE_URL"
echo "3. For SQLite: Set DATABASE_PATH (default: ./data/gomanager.db)"
echo "4. Configure Google OAuth credentials"
echo "5. Run: ./gomanager"
echo ""
echo "Database options:"
echo "• SQLite (development):   DATABASE_PATH=./data/gomanager.db"
echo "• PostgreSQL (production): DATABASE_URL=postgresql://..."
echo ""
echo "Migration:"
echo "• To migrate from SQLite to PostgreSQL: ./scripts/migrate_to_postgres.sh"
echo ""