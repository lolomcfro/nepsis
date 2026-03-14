package com.sober.admin

import android.app.admin.DevicePolicyManager
import android.content.ComponentName
import android.content.Context
import android.os.UserManager

class PolicyManager(
    private val context: Context,
    private val dpm: DevicePolicyManager,
    private val admin: ComponentName
) {

    fun hideApp(packageName: String) {
        dpm.setApplicationHidden(admin, packageName, true)
    }

    fun showApp(packageName: String) {
        dpm.setApplicationHidden(admin, packageName, false)
    }

    fun isHidden(packageName: String): Boolean {
        return dpm.isApplicationHidden(admin, packageName)
    }

    fun applyRestrictions() {
        dpm.addUserRestriction(admin, UserManager.DISALLOW_INSTALL_UNKNOWN_SOURCES)
    }
}
