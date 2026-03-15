package com.sober.admin

import android.app.admin.DevicePolicyManager
import android.content.BroadcastReceiver
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import java.io.File

class CommandReceiver : BroadcastReceiver() {

    private fun escapeJson(s: String): String =
        "\"" + s.replace("\\", "\\\\").replace("\"", "\\\"") + "\""

    override fun onReceive(context: Context, intent: Intent) {
        val dpm = context.getSystemService(Context.DEVICE_POLICY_SERVICE) as DevicePolicyManager
        val admin = ComponentName(context, AdminReceiver::class.java)
        val policyManager = PolicyManager(context, dpm, admin)

        when (intent.action) {
            "com.sober.HIDE_APP" -> {
                val pkg = intent.getStringExtra("package") ?: return
                policyManager.hideApp(pkg)
            }
            "com.sober.SHOW_APP" -> {
                val pkg = intent.getStringExtra("package") ?: return
                policyManager.showApp(pkg)
            }
            "com.sober.APPLY_RESTRICTIONS" -> {
                policyManager.applyRestrictions()
            }
            "com.sober.LIST_APPS" -> {
                val result = goAsync()
                Thread {
                    val outFile = File(context.cacheDir, "sober_apps.json")
                    try {
                        val lister = AppLister(context)
                        val json = lister.listAppsAsJson { pkg -> policyManager.isHidden(pkg) }
                        outFile.writeText(json)
                    } catch (e: Exception) {
                        outFile.writeText("""{"error":${escapeJson(e.toString())}}""")
                    } finally {
                        result.finish()
                    }
                }.start()
            }
            "android.intent.action.PACKAGE_REPLACED" -> {
                // No-op: stateless design means we cannot know which apps should be hidden.
                // If a background update resets a hidden app's visibility, the user
                // will see it reappear and can re-hide it from the Sober desktop app.
            }
        }
    }
}
