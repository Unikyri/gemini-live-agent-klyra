#!/bin/bash
# Instrucciones para Configurar Google Sign In en Android (Sprint 3)
# ================================================================

## PASO 1: Descargar google-services.json desde GCP

1. Ve a: https://console.firebase.google.com/
2. Selecciona el proyecto **agent-klyra**
3. En la izquierda, ve a "Project Settings" (rueda dentada)
4. En la pestaña "Your apps", busca la app Android **com.klyra.klyra**
5. Si no está, clic en "Agregar app" → Selecciona **Android**
   - Package name: `com.klyra.klyra`
   - SHA-1: Tu SHA-1 fingerprint (ya configurado)
6. Descarga el archivo `google-services.json`
7. Colócalo en: `mobile/android/app/google-services.json`

## PASO 2: Verificar la Configuración en build.gradle (ya debe estar)

En `mobile/android/app/build.gradle.kts`, verifica:

```gradle
plugins {
    id("com.google.gms.google-services") // ← Debe estar presente
}
```

En `mobile/android/build.gradle.kts`, verifica:

```gradle
plugins {
    id("com.google.gms.google-services") version "4.4.0" apply false
}
```

## PASO 3: Verificar Credenciales OAuth en GCP

1. Ve a: https://console.cloud.google.com/apis/credentials
2. Proyecto: **agent-klyra**
3. Busca la credencial OAuth 2.0 "Android"
4. Verifica que el SHA-1 y package `com.klyra.klyra` estén registrados

## PASO 4: Compilar y Probar

```bash
cd mobile
flutter clean
flutter pub get
flutter run
```

Si el error persiste:
- `flutter clean` + `flutter pub cache clean`
- Elimina `build/` y `.dart_tool/`
- Vuelve a compilar

## NOTA DE SEGURIDAD

- **NO commits `google-services.json` a Git** (está en .gitignore)
- Cada developer debe descargar su propio desde Firebase
- En CI/CD, el archivo debe agregarse desde secretos del pipeline
