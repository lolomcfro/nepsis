package com.sober.admin

import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.drawable.Drawable
import android.content.pm.PackageManager
import android.util.Base64
import java.io.ByteArrayOutputStream

class AppLister(private val context: Context) {

    private val pm: PackageManager = context.packageManager

    /**
     * Returns JSON for all apps that have a launcher intent (user-facing apps).
     * Icons are scaled to 48dp and base64-encoded.
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
            val icon = try {
                encodeIcon(pm.getApplicationIcon(pkg))
            } catch (e: Exception) {
                ""
            }
            val hidden = hiddenChecker(pkg)
            buildAppEntry(pkg, label, hidden, icon)
        }

        return buildJsonArray(entries)
    }

    fun buildAppEntry(pkg: String, label: String, hidden: Boolean, icon: String): String {
        val escapedLabel = label.replace("\\", "\\\\").replace("\"", "\\\"")
        return """{"package":"$pkg","label":"$escapedLabel","icon":"$icon","hidden":$hidden}"""
    }

    fun buildJsonArray(entries: List<String>): String = "[${entries.joinToString(",")}]"

    private fun encodeIcon(drawable: Drawable): String {
        val sizePx = (48 * context.resources.displayMetrics.density).toInt()
        val bitmap = Bitmap.createBitmap(sizePx, sizePx, Bitmap.Config.ARGB_8888)
        val canvas = Canvas(bitmap)
        drawable.setBounds(0, 0, sizePx, sizePx)
        drawable.draw(canvas)

        val bos = ByteArrayOutputStream()
        bitmap.compress(Bitmap.CompressFormat.PNG, 80, bos)
        return Base64.encodeToString(bos.toByteArray(), Base64.NO_WRAP)
    }
}
