# GoManager API - Production PostgreSQL + Google Integration

This project has been upgraded to support production-grade PostgreSQL databases and enhanced Google integrations including Google Drive and Google Ads API.

## 🚀 New Features

### Database Migration
- ✅ **PostgreSQL Support**: Production-ready database with connection pooling
- ✅ **Backward Compatibility**: Still supports SQLite for development
- ✅ **Auto-detection**: Database type is automatically detected from connection string

### Google Drive Integration
- ✅ **Folder Management**: Create and manage specific folders in Google Drive
- ✅ **File Operations**: Upload, download, delete files directly to/from Drive
- ✅ **Permissions**: Share files and manage access permissions
- ✅ **Search**: Find files within your Drive folders

### Google Ads API Integration
- ✅ **Campaign Management**: View and manage Google Ads campaigns
- ✅ **Performance Metrics**: Get campaign performance data
- ✅ **Account Info**: Access Google Ads account information
- 📝 **Note**: Google Ads API requires special setup and developer token

## 🛠️ Setup Instructions

### 1. Database Configuration

#### For PostgreSQL (Production):
```bash
# Set your DATABASE_URL environment variable
export DATABASE_URL="postgresql://neondb_owner:npg_e3KazGXVux4P@ep-young-frog-acw7jt2v-pooler.sa-east-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
```

#### For SQLite (Development):
```bash
# Set your DATABASE_PATH environment variable
export DATABASE_PATH="./data/gomanager.db"
```

### 2. Google Services Setup

#### Google OAuth & APIs:
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create or select a project
3. Enable these APIs:
   - Google Calendar API
   - Google Tasks API
   - Google Drive API
   - Google Ads API (optional)
4. Create OAuth 2.0 credentials
5. Set authorized redirect URIs: `http://localhost:8005/api/auth/google/callback`

#### Google Drive Configuration:
```bash
export GOOGLE_DRIVE_FOLDER="GoManager"  # Folder name in your Drive
```

#### Google Ads Configuration (Optional):
```bash
export GOOGLE_ADS_CUSTOMER_ID="your_customer_id"
export GOOGLE_ADS_DEVELOPER_TOKEN="your_developer_token"
```

### 3. Environment Configuration

Copy the example environment file and configure:
```bash
cp .env.example .env
# Edit .env with your actual credentials
```

### 4. Install Dependencies

```bash
go mod tidy
```

### 5. Run the Application

```bash
go run main.go
```

## 📡 API Endpoints

### Google Drive Endpoints
```
GET    /api/google/drive/files              - List files in Drive folder
POST   /api/google/drive/folders            - Create new folder
POST   /api/google/drive/upload             - Upload file to Drive
DELETE /api/google/drive/delete             - Delete file from Drive
```

### Google Ads Endpoints
```
GET    /api/google/ads/status               - Check Ads API connection
GET    /api/google/ads/campaigns            - List campaigns
POST   /api/google/ads/campaigns/create     - Create new campaign
GET    /api/google/ads/campaigns/performance - Get performance metrics
```

### Existing Endpoints
All previous endpoints remain unchanged:
- File management: `/api/files`, `/api/upload`, `/api/download`
- Authentication: `/api/auth/*`
- Google Calendar: `/api/google/calendar/*`
- Google Tasks: `/api/google/tasks/*`

## 🔧 Database Migration

The application automatically detects your database type and runs appropriate migrations:

- **PostgreSQL**: Detected by `postgresql://` or `postgres://` prefix
- **SQLite**: Default for file paths

### Migration Features:
- ✅ Creates new tables for Google Drive folders
- ✅ Creates new tables for Google Ads campaigns
- ✅ Backward compatible with existing databases
- ✅ Automatic index creation for performance

## 📁 Google Drive Folder Structure

When you connect your Google account, the application will:
1. Create or locate the specified folder in your Google Drive
2. Store all uploaded files in this dedicated folder
3. Maintain folder permissions and sharing settings
4. Sync file operations between local storage and Drive

## 🎯 Google Ads Integration

### Requirements:
1. Google Ads account with API access
2. Developer token (apply via Google Ads)
3. Customer ID from your Ads account
4. OAuth consent with `adwords` scope

### Features:
- Campaign listing and creation
- Performance metrics retrieval
- Account information access
- Budget and bidding management

## 🔐 Security Considerations

### Database Security:
- Connection pooling with limits
- SQL injection protection
- Encrypted connections (SSL/TLS)

### API Security:
- OAuth 2.0 with proper scopes
- Token refresh handling
- Rate limiting (recommended for production)

### File Security:
- Secure file upload validation
- Proper permission checking
- Encrypted file storage options

## 🐛 Troubleshooting

### Database Connection Issues:
1. Verify your connection string format
2. Check network connectivity to database
3. Ensure SSL certificates are valid
4. Verify database user permissions

### Google API Issues:
1. Check API quotas and limits
2. Verify OAuth consent screen setup
3. Ensure proper scopes are requested
4. Check for API key restrictions

### Common Errors:
- `Google account not connected`: Re-authenticate via `/api/auth/google`
- `Database connection failed`: Check DATABASE_URL format
- `Permission denied`: Verify Google API scopes and permissions

## 📊 Performance Considerations

### Database:
- Connection pooling: 25 max open, 5 max idle
- Query optimization with proper indexes
- Prepared statements for security

### File Operations:
- Multipart upload for large files
- Streaming for download operations
- Background sync for Drive operations

## 🚀 Production Deployment

1. Set up PostgreSQL database (like Neon, AWS RDS, etc.)
2. Configure environment variables for production
3. Set up proper CORS for your frontend domain
4. Enable HTTPS for security
5. Configure monitoring and logging
6. Set up backup strategies for database and files

## 📝 Next Steps

1. **Frontend Integration**: Update your frontend to use new Drive endpoints
2. **Monitoring**: Add application monitoring and logging
3. **Backup**: Implement backup strategies for both database and files
4. **Scaling**: Consider horizontal scaling options
5. **Security**: Implement rate limiting and additional security measures

---

This upgrade makes your GoManager API production-ready with enterprise-grade database support and enhanced Google integrations! 🎉