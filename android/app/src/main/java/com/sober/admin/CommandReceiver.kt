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
            "com.sober.EXPORT_CONTACTS" -> {
                val result = goAsync()
                Thread {
                    val outFile = File(context.cacheDir, "sober_contacts.vcf")
                    try {
                        val manager = ContactsManager(context)
                        manager.exportToVcf(outFile)
                    } catch (e: Exception) {
                        outFile.writeText("""{"error":${escapeJson(e.toString())}}""")
                    } finally {
                        result.finish()
                    }
                }.start()
            }
            "com.sober.IMPORT_CONTACTS" -> {
                val result = goAsync()
                Thread {
                    val resultFile = File(context.cacheDir, "sober_import_result.json")
                    try {
                        // Use the app's external files directory — no storage permission required,
                        // accessible to both ADB push and this app on all API levels (26+).
                        val vcfFile = File(context.getExternalFilesDir(null), "sober_contacts_restore.vcf")
                        if (!vcfFile.exists()) throw Exception("source file not found: ${vcfFile.absolutePath}")
                        val manager = ContactsManager(context)
                        val count = manager.importFromVcf(vcfFile)
                        vcfFile.delete()
                        resultFile.writeText("""{"success":true,"count":$count}""")
                    } catch (e: Exception) {
                        resultFile.writeText("""{"error":${escapeJson(e.toString())}}""")
                    } finally {
                        result.finish()
                    }
                }.start()
            }
            "com.sober.CLEAR_DEVICE_OWNER" -> {
                // goAsync() is required — without it the process is eligible for reclamation
                // before clearDeviceOwnerApp() executes on Android 8+ (API 26, this app's minSdk).
                // No result file is written: the desktop confirms success by polling
                // `dpm list-owners` until com.sober.admin no longer appears (10s deadline).
                val result = goAsync()
                Thread {
                    try {
                        dpm.clearDeviceOwnerApp(context.packageName)
                    } finally {
                        result.finish()
                    }
                }.start()
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
