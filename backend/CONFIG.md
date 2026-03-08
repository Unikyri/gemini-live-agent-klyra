# Klyra Backend Configuration

## Environment Variables

### Database
- `DB_HOST`: PostgreSQL hostname (default: localhost)
- `DB_PORT`: PostgreSQL port (default: 5433 for local)
- `DB_USER`: Database user (default: klyra)
- `DB_PASSWORD`: Database password
- `DB_NAME`: Database name (default: klyra_dev)
- `DB_SSLMODE`: SSL mode for cloud (production)

### Authentication
- `JWT_ACCESS_SECRET`: Secret for access tokens (min 32 chars)
- `JWT_REFRESH_SECRET`: Secret for refresh tokens (min 32 chars)
- `GOOGLE_CLIENT_ID`: OAuth client ID (from Google Cloud)
- `OAUTH_DISCOVERY_URL`: Google OIDC discovery endpoint

### Cloud Services
- `GCP_PROJECT_ID`: Google Cloud project ID
- `GCS_BUCKET`: Google Cloud Storage bucket for file uploads
- `VERTEX_AI_API_KEY`: Vertex AI embeddings API key (or use Application Default Credentials)
- `CLOUD_SQL_INSTANCE`: Cloud SQL instance connection name

### Development
- `ENV`: Environment ("local", "staging", "production")
- `LOG_LEVEL`: Logging level ("debug", "info", "warn", "error")
- `CORS_ALLOWED_ORIGINS`: Comma-separated CORS origins

## Local Development Setup

### Quick Start
```bash
# Start PostgreSQL + pgvector
docker-compose up -d

# Run migrations
go run cmd/api/main.go --migrate=true

# Start backend
go run cmd/api/main.go
```

### Environment File (.env.local)
```env
ENV=local
DB_HOST=localhost
DB_PORT=5433
DB_USER=klyra
DB_PASSWORD=klyra_dev_password
DB_NAME=klyra_dev
JWT_ACCESS_SECRET=your-secret-32-chars-or-longer-1234567890
JWT_REFRESH_SECRET=your-secret-32-chars-or-longer-9876543210
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
LOG_LEVEL=debug
```

## Deployment

### Google Cloud Run
```bash
# Build container
docker build -t gcr.io/klyra-project/backend:latest .

# Deploy
gcloud run deploy backend --image gcr.io/klyra-project/backend:latest \
  --platform managed \
  --region us-central1 \
  --set-env-vars ENV=production,GCP_PROJECT_ID=klyra-project,CLOUD_SQL_INSTANCE=...
```

### Cloud SQL Connection
For Cloud Run to Cloud SQL connection via Unix socket:
```go
// Connection string format
dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=/cloudsql/%s",
    dbUser, dbPassword, dbName, cloudSQLInstance)
```

## Database Schema

Migrations auto-run on startup:
1. `000001_create_users_table.up.sql` - Users with Google OAuth IDs
2. `000002_create_courses_and_topics.up.sql` - Learning hierarchy
3. `000003_create_materials_table.up.sql` - Course materials
4. `000004_add_pgvector_and_chunks.up.sql` - Vector embeddings for RAG

View schema: [backend/migrations/](./migrations/)

## Performance Tuning

### pgvector Indices
```sql
-- Create IVFFlat index for fast similarity search
CREATE INDEX ON material_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Vacuum analyze for statistics
VACUUM ANALYZE material_chunks;
```

### Connection Pool
- Max connections: 25 (default GORM)
- Connection timeout: 10s
- Idle timeout: 5 minutes

## Monitoring

### Health Check
```bash
curl http://localhost:8080/health
```

### Logs
- Local: stdout (JSON format)
- Cloud Run: Cloud Logging
- Log level controlled via `LOG_LEVEL` env var

## Testing

### Run All Tests
```bash
./test-all.sh          # Linux/macOS
.\test-all.ps1         # Windows
```

### Test Coverage
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Troubleshooting

### Database Connection Error
- Check `DB_HOST`, `DB_PORT` match docker-compose
- Verify database container is running: `docker ps`
- Reset database: `docker-compose down -v && docker-compose up`

### Embeddings API Error
- Verify `VERTEX_AI_API_KEY` is set or ADC credentials available
- Check API quota in Google Cloud console
- Fallback to mock embeddings in dev mode

### Guest Login Not Working
- Verify `ENV=local` is set
- Check backend is running on port 8080
- Verify Flutter base URL points to `http://localhost:8080/api/v1`
