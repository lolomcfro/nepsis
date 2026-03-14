package com.sober.admin

import android.app.admin.DevicePolicyManager
import android.content.ComponentName
import android.content.Context
import android.os.UserManager
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.mockito.Mock
import org.mockito.Mockito.mock
import org.mockito.Mockito.verify
import org.mockito.MockitoAnnotations
import org.mockito.junit.MockitoJUnitRunner
import org.mockito.kotlin.eq

@RunWith(MockitoJUnitRunner::class)
class PolicyManagerTest {

    @Mock
    private lateinit var context: Context

    @Mock
    private lateinit var dpm: DevicePolicyManager

    @Mock
    private lateinit var admin: ComponentName

    private lateinit var policyManager: PolicyManager

    @Before
    fun setUp() {
        policyManager = PolicyManager(context, dpm, admin)
    }

    @Test
    fun `hideApp calls setApplicationHidden with true`() {
        policyManager.hideApp("com.reddit.frontpage")
        verify(dpm).setApplicationHidden(eq(admin), eq("com.reddit.frontpage"), eq(true))
    }

    @Test
    fun `showApp calls setApplicationHidden with false`() {
        policyManager.showApp("com.reddit.frontpage")
        verify(dpm).setApplicationHidden(eq(admin), eq("com.reddit.frontpage"), eq(false))
    }

    @Test
    fun `applyRestrictions calls addUserRestriction with DISALLOW_INSTALL_UNKNOWN_SOURCES`() {
        policyManager.applyRestrictions()
        verify(dpm).addUserRestriction(
            eq(admin),
            eq(UserManager.DISALLOW_INSTALL_UNKNOWN_SOURCES)
        )
    }
}
