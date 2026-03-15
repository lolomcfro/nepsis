package com.sober.admin

import android.content.ContentProviderOperation
import android.content.Context
import android.provider.ContactsContract
import java.io.File

class ContactsManager(private val context: Context) {

    fun exportToVcf(outputFile: File) {
        val sb = StringBuilder()
        val contactsCursor = context.contentResolver.query(
            ContactsContract.Contacts.CONTENT_URI,
            arrayOf(ContactsContract.Contacts._ID, ContactsContract.Contacts.DISPLAY_NAME_PRIMARY),
            null, null, null
        )
        contactsCursor?.use { c ->
            while (c.moveToNext()) {
                val id = c.getString(c.getColumnIndexOrThrow(ContactsContract.Contacts._ID))
                val name = c.getString(c.getColumnIndexOrThrow(ContactsContract.Contacts.DISPLAY_NAME_PRIMARY)) ?: ""
                if (name.isBlank()) continue  // skip contacts with no display name
                sb.append("BEGIN:VCARD\r\nVERSION:3.0\r\n")
                sb.append("FN:${escapeVcf(name)}\r\n")

                val phoneCursor = context.contentResolver.query(
                    ContactsContract.CommonDataKinds.Phone.CONTENT_URI,
                    arrayOf(ContactsContract.CommonDataKinds.Phone.NUMBER),
                    "${ContactsContract.CommonDataKinds.Phone.CONTACT_ID} = ?",
                    arrayOf(id), null
                )
                phoneCursor?.use { pc ->
                    while (pc.moveToNext()) {
                        val num = pc.getString(pc.getColumnIndexOrThrow(ContactsContract.CommonDataKinds.Phone.NUMBER))
                        if (!num.isNullOrBlank()) sb.append("TEL:${escapeVcf(num)}\r\n")
                    }
                }

                val emailCursor = context.contentResolver.query(
                    ContactsContract.CommonDataKinds.Email.CONTENT_URI,
                    arrayOf(ContactsContract.CommonDataKinds.Email.ADDRESS),
                    "${ContactsContract.CommonDataKinds.Email.CONTACT_ID} = ?",
                    arrayOf(id), null
                )
                emailCursor?.use { ec ->
                    while (ec.moveToNext()) {
                        val addr = ec.getString(ec.getColumnIndexOrThrow(ContactsContract.CommonDataKinds.Email.ADDRESS))
                        if (!addr.isNullOrBlank()) sb.append("EMAIL:${escapeVcf(addr)}\r\n")
                    }
                }

                sb.append("END:VCARD\r\n")
            }
        }
        outputFile.writeText(sb.toString())
    }

    /** Imports contacts from a VCF 3.0 file. Returns the number of contacts imported. */
    fun importFromVcf(vcfFile: File): Int {
        val content = vcfFile.readText()
        val cards = content.split("END:VCARD").filter { it.contains("BEGIN:VCARD") }
        var count = 0
        for (card in cards) {
            val name = card.lines()
                .firstOrNull { it.startsWith("FN:") }
                ?.removePrefix("FN:")?.trim()
                ?.let { unescapeVcf(it) }
                ?.takeIf { it.isNotBlank() }
                ?: continue

            val phones = card.lines()
                // Match bare TEL: and typed TEL;TYPE=...: lines. The contains(":") guard prevents
                // matching TEL; lines that have no value portion. Parentheses are mandatory here —
                // && binds tighter than || without them.
                .filter { line -> line.startsWith("TEL:") || (line.startsWith("TEL;") && line.contains(":")) }
                .mapNotNull { line ->
                    val colon = line.lastIndexOf(':')
                    if (colon >= 0) unescapeVcf(line.substring(colon + 1).trim()) else null
                }
                .filter { it.isNotBlank() }

            val emails = card.lines()
                .filter { it.startsWith("EMAIL:") }
                .map { unescapeVcf(it.removePrefix("EMAIL:").trim()) }
                .filter { it.isNotBlank() }

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
            try {
                context.contentResolver.applyBatch(ContactsContract.AUTHORITY, ops)
                count++
            } catch (e: Exception) {
                android.util.Log.w("ContactsManager", "Failed to import contact '$name': ${e.message}")
            }
        }
        return count
    }

    private fun escapeVcf(s: String): String = s.replace("\\", "\\\\").replace(",", "\\,").replace("\n", "\\n")
    private fun unescapeVcf(s: String): String = s.replace("\\\\", "\\").replace("\\,", ",").replace("\\n", "\n")
}
