# 🚀 LOCAL DEVELOPMENT DASHBOARD
**Status**: March 8, 2026  
**All Services Running** ✅

---

## 📊 SISTEMA OPERATIVO

### ✅ **1. PostgreSQL Database**
- **Status**: Running
- **Container**: klyra_postgres
- **Port**: 5433 (host) → 5432 (container)
- **Image**: pgvector/pgvector:pg15
- **Connection**: postgres://user:password@localhost:5433/klyra_db

**Verificar:**
```bash
docker-compose ps
```

**Conectarse con psql:**
```bash
psql -h localhost -p 5433 -U user -d klyra_db -W
# Password: password
```

---

### ✅ **2. Backend API (Go)**
- **Status**: Running
- **Port**: 8080
- **Environment**: local
- **Database Mode**: local (PostgreSQL on 5433)
- **Storage Mode**: local (./storage)

**URL Base**: `http://localhost:8080`

**Health Check:**
```bash
curl http://localhost:8080/health
# Response: {"status":"ok"}
```

**Endpoints Disponibles:**
```
POST   /auth/google-signin-mock          → Login sin Google (local)
POST   /courses                          → Crear curso
GET    /courses                          → Listar cursos del usuario
GET    /courses/{id}                    → Detalle del curso
POST   /courses/{id}/materials           → Upload de material
GET    /courses/{id}/materials           → Listar materiales
GET    /courses/{id}/context?query=...   → RAG context (similarity search)
```

**En desarrollo, para login test:**
```bash
curl -X POST http://localhost:8080/auth/google-signin-mock \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "name": "Test User"
  }'

# Response (guarda el token JWT):
# {
#   "token": "eyJhbGciOiJIUzI1NiI...",
#   "user": {
#     "id": "550e8400-e29b-41d4-a716...",
#     "email": "test@example.com",
#     "name": "Test User"
#   }
# }
```

**Crear curso con token:**
```bash
curl -X POST http://localhost:8080/courses \
  -H "Authorization: Bearer <TOKEN_FROM_ABOVE>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Cálculo Diferencial",
    "description": "Fundamentos de cálculo",
    "imageUrl": "https://example.com/image.jpg"
  }'
```

---

### ✅ **3. Frontend (Flutter)**
- **Status**: Running
- **Platform**: Windows Desktop
- **URL**: App abierta automáticamente
- **Connect to Backend**: http://localhost:8080

**Funcionalidades**:
- Dashboard: Ver cursos
- Login con mock de Google
- Crear/editar cursos
- Upload de materiales
- Búsqueda y contexto RAG

**Hot Reload**: Presiona `r` en la terminal para recargar sin reiniciar

---

## 🔄 FLUJO DE DESARROLLO LOCAL

### **1. Inicia una nueva sesión de cliente (Flutter)**

Abre `http://localhost:8080/auth/google-signin-mock` (o usa el botón en el app):

```json
{
  "email": "dev-user@example.com",
  "name": "Developer"
}
```

**Respuesta**: Recibe JWT token (se guarda en `SharedPreferences` automáticamente)

### **2. Crea un curso**

```bash
POST /courses
Header: Authorization: Bearer <TOKEN>
Body: {
  "name": "Mi Curso",
  "description": "Descripción",
  "imageUrl": "https://..."
}
```

### **3. Sube un material (PDF/TXT)**

```bash
POST /courses/{courseId}/materials
Header: Authorization: Bearer <TOKEN>
        Content-Type: multipart/form-data
Body: file=<tu-archivo.pdf>
      topicId=<topic-uuid>
```

### **4. Sistema RAG se ejecuta automáticamente**

La API:
1. Extrae texto del PDF
2. Divide en chunks (800 runes, 100 overlap)
3. Genera embeddings (Vertex AI o mock en local)
4. Guarda en pgvector con topicID scoping
5. Calcula similarity scores

### **5. Consulta RAG context**

```bash
GET /courses/{courseId}/context?query=¿Qué es derivada?
Header: Authorization: Bearer <TOKEN>

# Respuesta contiene top-3 chunks más relevantes
{
  "context": "Un fragmento del material...",
  "chunks": [
    {
      "id": "...",
      "content": "...",
      "similarity": 0.87
    }
  ]
}
```

---

## 📝 COMANDOS ÚTILES

### **Backend**
```bash
# Tests locales (no requiere DB)
cd backend && go test ./... -v

# Tests con integración (requiere Docker + PostgreSQL)
go test -tags=integration ./... -v

# Logs del backend (en otra terminal)
tail -f backend/logs.txt

# Recarga caliente en desarrollo (con air)
cd backend && air
```

### **Frontend**
```bash
# Hot reload en línea de comandos
# Presiona 'r' en la terminal

# Rebuild completo
flutter clean && flutter run -d windows

# Tests de UI
flutter test

# Build Web (para despliegue)
flutter build web
```

### **Database**
```bash
# Conectar a PostgreSQL
psql -h localhost -p 5433 -U user -d klyra_db -W

# Ver esquema
\dt
\d material_chunks

# Query rápida - contar chunks
SELECT COUNT(*) FROM material_chunks;

# Query rápida - ver por topic
SELECT topic_id, COUNT(*) FROM material_chunks GROUP BY topic_id;

# Similarity search manual
SELECT id, similarity, content 
FROM (
  SELECT id, content, 
         1 - (embedding <=> '[0.1, 0.2, ...]'::vector) AS similarity
  FROM material_chunks 
  WHERE topic_id = '...'
  ORDER BY embedding <=> '[0.1, 0.2, ...]'::vector
  LIMIT 3
) AS ranked;
```

---

## 🛠️ TROUBLESHOOTING

### **Backend no inicia**

```bash
# Verifica que PostgreSQL está corriendo
docker-compose ps

# Verifica el log de PostgreSQL
docker-compose logs postgres

# Reinicia PostgreSQL
docker-compose down
docker-compose up -d postgres

# Intenta backend nuevamente
cd backend && go run ./cmd/api/main.go
```

### **Flutter no compila**

```bash
# Limpia build cache
flutter clean

# Get dependencies
flutter pub get

# Intenta compilar
flutter run -d windows
```

### **Database connection error**

Verifica que el puerto es **5433** (no 5432):
- Docker expone: 5433:5432
- Tu conexión debe ir a localhost:5433

```bash
# Reconecta en .env
DB_PORT=5433
```

### **Tests fallan**

```bash
# Ejecuta solo tests unitarios (no integración)
go test ./... -v -short

# O con integración (requiere DB)
go test -tags=integration ./... -v
```

---

## 🎯 SIGUIENTE PASO

### Local Development está listo para:
✅ Crear cursos y temas  
✅ Subir materiales (PDF, TXT)  
✅ Probar RAG pipeline (chunking → embedding → search)  
✅ Desarrollar features del frontend  
✅ Escribir tests  

### Para Staging Deployment:
→ Ref: [SPRINT-5-KICKOFF.md](.agent/reports/SPRINT-5-KICKOFF.md)

---

## 📞 CONTACTO RÁPIDO

**Logs en tiempo real:**
```bash
# Backend
docker-compose logs -f postgres

# Metrics DB
SELECT NOW();
SELECT COUNT(*) FROM material_chunks;
SELECT COUNT(*) FROM courses WHERE created_at > NOW() - INTERVAL '1 hour';
```

**Resetear todo (limpio):**
```bash
docker-compose down -v  # Elimina volúmenes
docker-compose up -d postgres
# Backend y Frontend reinician automáticamente
```

---

**Ready to code!** 🚀
