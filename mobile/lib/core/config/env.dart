class EnvInfo {
  // Hardcoded for Sprint 2 local dev; later we should use a .env package (e.g., flutter_dotenv)
  // Remember that Android emulator uses 10.0.2.2 to access localhost, 
  // and iOS simulator uses 127.0.0.1. We assume Windows local testing might use localhost.
  static const String backendBaseUrl = 'http://localhost:8080/api/v1';
}
