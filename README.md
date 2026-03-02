# Klyra - Tutoría Interactiva por IA

Klyra es una aplicación móvil innovadora respaldada por inteligencia artificial (Gemini) que transforma apuntes y textos estáticos en "clases magistrales" interactivas con un Tutor Avatar utilizando "barge-in" (interrupciones) en tiempo real, inyección de fondos visuales (Graph RAG), y memoria persistente del estudiante.

## Visión del Producto
Para estudiantes de nivel medio y superior que necesitan un soporte de aprendizaje personalizado, Klyra ofrece explicaciones ultra-rápidas e inmersivas en tiempo real.

## Arquitectura (MVP)
* **Frontend:** Flutter (Mobile) con soporte de renderizado multicapa para inyección de fondos.
* **Backend:** Go (Monolito Modular con Clean Architecture) alojado en Google Cloud Run.
* **Componentes IA:** Gemini Live API (barge-in / audio), Imagen (generación de sprites/avatares), y Vertex AI Vector Search (Graph RAG).
* **Base de Datos:** PostgreSQL para persistencia de usuarios, perfiles de aprendizaje y referenciación RAG.
* **Conexiones:** WebSockets (WSS) cliente-servidor para los eventos asíncronos en vivo (fondos), y API REST para gestión de CRUD y carga de archivos.

## Configuración del Entorno de Desarrollo

### Requisitos Previos
* [Go 1.22+](https://go.dev/doc/install)
* [Flutter 3.22+](https://docs.flutter.dev/get-started/install)
* [PostgreSQL 16+](https://www.postgresql.org/download/)
* Cuenta en [Google Cloud](https://cloud.google.com/) con Vertex AI y Gemini API habilitados.

### Instalación Local
1. Clona este repositorio:
   ```bash
   git clone https://github.com/Unikyri/gemini-live-agent-klyra.git
   cd gemini-live-agent-klyra
   ```
2. Configura las variables de entorno basándote en `.env.example`.
3. Inicia la base de datos local y corre las migraciones correspondientes (a definir en `/backend`).
4. (Opcional) Instala dependencias para web/mobile dentro de `/mobile`.

_Este repositorio emplea convenciones estrictas de seguridad (ver Threat Model) y control de versión con Trunk Based Development (o GitHub Flow)._
