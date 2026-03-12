package com.klyra.klyra

import android.content.Context
import android.media.AudioManager
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.embedding.android.FlutterActivity
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
  private val channelName = "klyra/audio_session"
  private var previousMode: Int? = null

  override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
    super.configureFlutterEngine(flutterEngine)
    MethodChannel(flutterEngine.dartExecutor.binaryMessenger, channelName)
      .setMethodCallHandler { call, result ->
        when (call.method) {
          "setVoiceChatMode" -> {
            try {
              val am = getSystemService(Context.AUDIO_SERVICE) as AudioManager
              if (previousMode == null) {
                previousMode = am.mode
              }
              am.mode = AudioManager.MODE_IN_COMMUNICATION
              result.success(true)
            } catch (e: Exception) {
              result.error("AEC_SET_FAILED", e.message, null)
            }
          }
          "resetAudioMode" -> {
            try {
              val am = getSystemService(Context.AUDIO_SERVICE) as AudioManager
              val mode = previousMode
              if (mode != null) {
                am.mode = mode
              }
              previousMode = null
              result.success(true)
            } catch (e: Exception) {
              result.error("AEC_RESET_FAILED", e.message, null)
            }
          }
          else -> result.notImplemented()
        }
      }
  }
}
