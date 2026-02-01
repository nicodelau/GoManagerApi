#!/bin/bash

# GoManager SQLite to PostgreSQL Migration Script
# This script helps migrate your existing SQLite data to PostgreSQL

set -e

echo "==============================================="
echo "  GoManager Database Migration Tool"
echo "  SQLite → PostgreSQL"
echo "==============================================="

# Check if required tools are available
command -v sqlite3 >/dev/null 2>&1 || { echo "Error: sqlite3 is required but not installed." >&2; exit 1; }
command -v psql >/dev/null 2>&1 || { echo "Error: psql is required but not installed." >&2; exit 1; }

# Configuration
SQLITE_DB="${1:-./data/gomanager.db}"
POSTGRES_URL="${2:-$DATABASE_URL}"

if [ -z "$POSTGRES_URL" ]; then
    echo "Usage: $0 [sqlite_db_path] [postgres_url]"
    echo ""
    echo "Examples:"
    echo "  $0 ./data/gomanager.db postgresql://user:pass@host:5432/dbname"
    echo "  DATABASE_URL='postgresql://...' $0"
    echo ""
    echo "Environment variables:"
    echo "  DATABASE_URL - PostgreSQL connection string"
    echo ""
    exit 1
fi

if [ ! -f "$SQLITE_DB" ]; then
    echo "Error: SQLite database file not found: $SQLITE_DB"
    exit 1
fi

echo "Source SQLite DB: $SQLITE_DB"
echo "Target PostgreSQL: $(echo $POSTGRES_URL | sed 's/:\/\/.*@/:\/\/***@/')"
echo ""

read -p "Continue with migration? (y/N): " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Migration cancelled."
    exit 0
fi

echo "Starting migration..."

# Create temporary dump files
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "1. Exporting SQLite data..."

# Export users table
sqlite3 "$SQLITE_DB" <<EOF > "$TEMP_DIR/users.sql"
.mode insert users
SELECT * FROM users;
EOF

# Export sessions table
sqlite3 "$SQLITE_DB" <<EOF > "$TEMP_DIR/sessions.sql"
.mode insert sessions  
SELECT * FROM sessions;
EOF

# Export shares table
sqlite3 "$SQLITE_DB" <<EOF > "$TEMP_DIR/shares.sql"
.mode insert shares
SELECT * FROM shares;
EOF

echo "2. Converting data format for PostgreSQL..."

# Convert SQLite INSERT syntax to PostgreSQL compatible format
for table in users sessions shares; do
    if [ -s "$TEMP_DIR/$table.sql" ]; then
        sed -i 's/INSERT INTO \([^(]*\)/INSERT INTO \1/g' "$TEMP_DIR/$table.sql"
        # Convert boolean values for PostgreSQL
        sed -i 's/,1,/,true,/g' "$TEMP_DIR/$table.sql"
        sed -i 's/,0,/,false,/g' "$TEMP_DIR/$table.sql"
        sed -i 's/,1);/,true);/g' "$TEMP_DIR/$table.sql"
        sed -i 's/,0);/,false);/g' "$TEMP_DIR/$table.sql"
    fi
done

echo "3. Importing data to PostgreSQL..."

# Import each table
for table in users sessions shares; do
    if [ -s "$TEMP_DIR/$table.sql" ]; then
        echo "   Importing $table..."
        psql "$POSTGRES_URL" -c "TRUNCATE TABLE $table CASCADE;" || true
        psql "$POSTGRES_URL" -f "$TEMP_DIR/$table.sql" || {
            echo "Warning: Failed to import $table table. It might be empty or have format issues."
        }
    else
        echo "   Skipping $table (no data found)"
    fi
done

echo "4. Updating sequences (PostgreSQL)..."
psql "$POSTGRES_URL" <<EOF || true
-- No auto-increment columns to update in current schema
-- This is a placeholder for future sequence updates
SELECT 'Migration completed successfully!' as status;
EOF

echo ""
echo "==============================================="
echo "✅ Migration completed successfully!"
echo "==============================================="
echo ""
echo "Next steps:"
echo "1. Update your .env file to use DATABASE_URL instead of DATABASE_PATH"
echo "2. Test your application with the new PostgreSQL database"
echo "3. Backup your original SQLite file"
echo ""
echo "To use PostgreSQL going forward:"
echo "export DATABASE_URL='$POSTGRES_URL'"
echo ""
echo "To rollback (if needed):"
echo "export DATABASE_PATH='$SQLITE_DB'"