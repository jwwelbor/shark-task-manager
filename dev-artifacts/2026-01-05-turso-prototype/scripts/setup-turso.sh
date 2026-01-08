#!/bin/bash
# Turso Setup Script for Prototype
# This script guides you through setting up a Turso database for testing

set -e

echo "=== Turso Prototype Setup ==="
echo ""

# Check if turso CLI is installed
if ! command -v turso &> /dev/null; then
    echo "❌ Turso CLI not found"
    echo ""
    echo "Install with:"
    echo "  curl -sSfL https://get.tur.so/install.sh | bash"
    echo ""
    echo "After installation, restart your terminal and run this script again."
    exit 1
fi

echo "✅ Turso CLI found: $(turso --version)"
echo ""

# Check if authenticated
if ! turso auth token &> /dev/null; then
    echo "❌ Not authenticated with Turso"
    echo ""
    echo "Run: turso auth login"
    echo ""
    echo "Then run this script again."
    exit 1
fi

echo "✅ Authenticated with Turso"
echo ""

# Database name
DB_NAME="shark-tasks-prototype"

# Check if database exists
if turso db show "$DB_NAME" &> /dev/null; then
    echo "⚠️  Database '$DB_NAME' already exists"
    echo ""
    read -p "Do you want to use the existing database? (y/n): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Please choose a different database name or delete the existing one:"
        echo "  turso db destroy $DB_NAME"
        exit 1
    fi
else
    echo "Creating database '$DB_NAME'..."
    turso db create "$DB_NAME"
    echo "✅ Database created"
    echo ""
fi

# Get database URL
echo "Fetching database URL..."
DB_URL=$(turso db show "$DB_NAME" --url)
echo "✅ Database URL: $DB_URL"
echo ""

# Create auth token
echo "Creating authentication token..."
AUTH_TOKEN=$(turso db tokens create "$DB_NAME")
echo "✅ Auth token created (stored securely)"
echo ""

# Create .env file
ENV_FILE="dev-artifacts/2026-01-05-turso-prototype/prototype/.env"
echo "Creating .env file at $ENV_FILE..."
cat > "$ENV_FILE" << EOF
# Turso Connection Details
# Generated: $(date)
export TURSO_DATABASE_URL="$DB_URL"
export TURSO_AUTH_TOKEN="$AUTH_TOKEN"
EOF

echo "✅ Credentials saved to $ENV_FILE"
echo ""

echo "=== Setup Complete ==="
echo ""
echo "Database Details:"
echo "  Name: $DB_NAME"
echo "  URL:  $DB_URL"
echo ""
echo "Next Steps:"
echo "  1. cd dev-artifacts/2026-01-05-turso-prototype/prototype"
echo "  2. source .env"
echo "  3. go run main.go"
echo ""
echo "Or run the prototype from project root:"
echo "  cd dev-artifacts/2026-01-05-turso-prototype/prototype && go run main.go"
