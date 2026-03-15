package com.sober.admin

import android.app.Application
import android.provider.ContactsContract
import org.robolectric.shadows.ShadowContentResolver

/**
 * Custom Robolectric Application that registers FakeContactsProvider for ContactsContract tests.
 */
class TestApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        val provider = FakeContactsProvider()
        provider.attachInfo(this, null)
        ShadowContentResolver.registerProviderInternal(ContactsContract.AUTHORITY, provider)
    }
}
