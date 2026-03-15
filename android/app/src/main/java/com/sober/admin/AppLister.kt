package com.sober.admin

import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.graphics.Bitmap
import android.graphics.Canvas
import android.util.Base64
import java.io.ByteArrayOutputStream

class AppLister(private val context: Context) {

    private val pm: PackageManager = context.packageManager

    private fun encodeIcon(pkg: String): String {
        return try {
            val drawable = pm.getApplicationIcon(pkg)
            val bitmap = Bitmap.createBitmap(48, 48, Bitmap.Config.ARGB_8888)
            val canvas = Canvas(bitmap)
            drawable.setBounds(0, 0, 48, 48)
            drawable.draw(canvas)
            val out = ByteArrayOutputStream()
            bitmap.compress(Bitmap.CompressFormat.PNG, 100, out)
            Base64.encodeToString(out.toByteArray(), Base64.NO_WRAP)
        } catch (e: Exception) {
            ""
        }
    }

    /**
     * Returns JSON for all apps that have a launcher intent (user-facing apps).
     * Icons are encoded as 48×48 PNG base64 strings. Call this from a background
     * thread (e.g. via goAsync) to avoid ANR on the main broadcast receiver thread.
     */
    fun listAppsAsJson(hiddenChecker: (String) -> Boolean): String {
        val launcherIntent = Intent(Intent.ACTION_MAIN).apply {
            addCategory(Intent.CATEGORY_LAUNCHER)
        }
        val resolvedApps = pm.queryIntentActivities(launcherIntent, 0)
            .map { it.activityInfo.packageName }
            .distinct()
            .sorted()

        val entries = resolvedApps.map { pkg ->
            val label = try {
                pm.getApplicationLabel(pm.getApplicationInfo(pkg, 0)).toString()
            } catch (e: PackageManager.NameNotFoundException) {
                pkg
            }
            val hidden = hiddenChecker(pkg)
            buildAppEntry(pkg, label, hidden, encodeIcon(pkg))
        }

        return buildJsonArray(entries)
    }

    fun buildAppEntry(pkg: String, label: String, hidden: Boolean, icon: String): String {
        val escapedLabel = label.replace("\\", "\\\\").replace("\"", "\\\"")
        return """{"package":"$pkg","label":"$escapedLabel","icon":"$icon","hidden":$hidden}"""
    }

    fun buildJsonArray(entries: List<String>): String = "[${entries.joinToString(",")}]"

}
