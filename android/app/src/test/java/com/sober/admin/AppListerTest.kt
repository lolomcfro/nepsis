package com.sober.admin

import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment

@RunWith(RobolectricTestRunner::class)
class AppListerTest {

    @Test
    fun `buildAppEntry produces valid JSON entry`() {
        val context = RuntimeEnvironment.getApplication()
        val lister = AppLister(context)
        val entry = lister.buildAppEntry("com.android.dialer", "Phone", false, "")
        assertTrue(entry.contains("\"package\":\"com.android.dialer\""))
        assertTrue(entry.contains("\"label\":\"Phone\""))
        assertTrue(entry.contains("\"hidden\":false"))
    }

    @Test
    fun `buildJsonArray wraps entries in array`() {
        val context = RuntimeEnvironment.getApplication()
        val lister = AppLister(context)
        val entries = listOf(
            lister.buildAppEntry("com.foo", "Foo", false, ""),
            lister.buildAppEntry("com.bar", "Bar", true, "")
        )
        val json = lister.buildJsonArray(entries)
        assertTrue(json.startsWith("["))
        assertTrue(json.endsWith("]"))
        assertTrue(json.contains("com.foo"))
        assertTrue(json.contains("com.bar"))
    }

    @Test
    fun `buildAppEntry escapes quotes in label`() {
        val context = RuntimeEnvironment.getApplication()
        val lister = AppLister(context)
        val entry = lister.buildAppEntry("com.foo", "App \"Name\"", false, "")
        assertTrue("Label with quotes should be escaped", entry.contains("\\\"Name\\\""))
    }
}
