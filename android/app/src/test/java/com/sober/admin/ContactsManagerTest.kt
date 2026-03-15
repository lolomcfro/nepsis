package com.sober.admin

import android.content.ContentProviderOperation
import android.provider.ContactsContract
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import java.io.File

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [28])
class ContactsManagerTest {

    private lateinit var context: android.content.Context
    private lateinit var manager: ContactsManager

    @Before
    fun setUp() {
        context = ApplicationProvider.getApplicationContext()
        manager = ContactsManager(context)
    }

    @Test
    fun `exportToVcf writes empty string when no contacts`() {
        val file = File(context.cacheDir, "test_export.vcf")
        manager.exportToVcf(file)
        assertEquals("", file.readText())
    }

    @Test
    fun `exportToVcf includes contact name`() {
        insertContact(context, "Alice Smith", listOf("+15550001111"), emptyList())
        val file = File(context.cacheDir, "test_export2.vcf")
        manager.exportToVcf(file)
        val vcf = file.readText()
        assertTrue("Expected FN:Alice Smith in VCF", vcf.contains("FN:Alice Smith"))
        assertTrue("Expected BEGIN:VCARD", vcf.contains("BEGIN:VCARD"))
        assertTrue("Expected END:VCARD", vcf.contains("END:VCARD"))
    }

    @Test
    fun `exportToVcf includes phone number`() {
        insertContact(context, "Bob Jones", listOf("+15550002222"), emptyList())
        val file = File(context.cacheDir, "test_export3.vcf")
        manager.exportToVcf(file)
        assertTrue(file.readText().contains("+15550002222"))
    }

    @Test
    fun `exportToVcf includes email`() {
        insertContact(context, "Carol", emptyList(), listOf("carol@example.com"))
        val file = File(context.cacheDir, "test_export4.vcf")
        manager.exportToVcf(file)
        assertTrue(file.readText().contains("carol@example.com"))
    }

    @Test
    fun `importFromVcf round-trip restores contact`() {
        insertContact(context, "Dave Export", listOf("+15550003333"), listOf("dave@example.com"))
        val exportFile = File(context.cacheDir, "roundtrip.vcf")
        manager.exportToVcf(exportFile)

        // Wipe all contacts
        context.contentResolver.delete(ContactsContract.RawContacts.CONTENT_URI, null, null)

        val count = manager.importFromVcf(exportFile)
        assertTrue("Expected at least 1 import, got $count", count >= 1)

        val cursor = context.contentResolver.query(
            ContactsContract.Contacts.CONTENT_URI,
            arrayOf(ContactsContract.Contacts.DISPLAY_NAME_PRIMARY),
            null, null, null
        )
        var found = false
        cursor?.use { while (it.moveToNext()) { if (it.getString(0) == "Dave Export") found = true } }
        assertTrue("Expected Dave Export to be restored", found)
    }

    @Test
    fun `importFromVcf returns 0 for empty file`() {
        val empty = File(context.cacheDir, "empty.vcf")
        empty.writeText("")
        val count = manager.importFromVcf(empty)
        assertEquals(0, count)
    }

    @Test
    fun `importFromVcf skips cards with no FN`() {
        val vcf = "BEGIN:VCARD\r\nVERSION:3.0\r\nTEL:+15550004444\r\nEND:VCARD\r\n"
        val file = File(context.cacheDir, "nofn.vcf")
        file.writeText(vcf)
        val count = manager.importFromVcf(file)
        assertEquals(0, count)
    }

    @Test
    fun `importFromVcf does not match non-phone typed lines`() {
        // ADR and PHOTO lines must not be treated as phone numbers
        val vcf = "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Eve\r\n" +
                  "ADR;TYPE=HOME:;;123 Main St;City;ST;12345;US\r\n" +
                  "TEL:+15550005555\r\n" +
                  "END:VCARD\r\n"
        val file = File(context.cacheDir, "nonphone.vcf")
        file.writeText(vcf)
        manager.importFromVcf(file)

        val phoneCursor = context.contentResolver.query(
            ContactsContract.CommonDataKinds.Phone.CONTENT_URI,
            arrayOf(ContactsContract.CommonDataKinds.Phone.NUMBER),
            null, null, null
        )
        var count = 0
        phoneCursor?.use { while (it.moveToNext()) count++ }
        assertEquals("Expected exactly 1 phone number (not the ADR line)", 1, count)
    }

    private fun insertContact(
        ctx: android.content.Context,
        name: String,
        phones: List<String>,
        emails: List<String>
    ) {
        val ops = ArrayList<ContentProviderOperation>()
        ops.add(
            ContentProviderOperation.newInsert(ContactsContract.RawContacts.CONTENT_URI)
                .withValue(ContactsContract.RawContacts.ACCOUNT_TYPE, null)
                .withValue(ContactsContract.RawContacts.ACCOUNT_NAME, null)
                .build()
        )
        ops.add(
            ContentProviderOperation.newInsert(ContactsContract.Data.CONTENT_URI)
                .withValueBackReference(ContactsContract.Data.RAW_CONTACT_ID, 0)
                .withValue(ContactsContract.Data.MIMETYPE, ContactsContract.CommonDataKinds.StructuredName.CONTENT_ITEM_TYPE)
                .withValue(ContactsContract.CommonDataKinds.StructuredName.DISPLAY_NAME, name)
                .build()
        )
        for (phone in phones) {
            ops.add(
                ContentProviderOperation.newInsert(ContactsContract.Data.CONTENT_URI)
                    .withValueBackReference(ContactsContract.Data.RAW_CONTACT_ID, 0)
                    .withValue(ContactsContract.Data.MIMETYPE, ContactsContract.CommonDataKinds.Phone.CONTENT_ITEM_TYPE)
                    .withValue(ContactsContract.CommonDataKinds.Phone.NUMBER, phone)
                    .build()
            )
        }
        for (email in emails) {
            ops.add(
                ContentProviderOperation.newInsert(ContactsContract.Data.CONTENT_URI)
                    .withValueBackReference(ContactsContract.Data.RAW_CONTACT_ID, 0)
                    .withValue(ContactsContract.Data.MIMETYPE, ContactsContract.CommonDataKinds.Email.CONTENT_ITEM_TYPE)
                    .withValue(ContactsContract.CommonDataKinds.Email.ADDRESS, email)
                    .build()
            )
        }
        ctx.contentResolver.applyBatch(ContactsContract.AUTHORITY, ops)
    }
}
