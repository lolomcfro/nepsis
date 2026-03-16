package com.sober.admin

import android.content.ContentProvider
import android.content.ContentProviderOperation
import android.content.ContentProviderResult
import android.content.ContentValues
import android.database.Cursor
import android.database.sqlite.SQLiteDatabase
import android.database.sqlite.SQLiteOpenHelper
import android.net.Uri
import android.provider.ContactsContract

/**
 * Minimal in-memory SQLite ContentProvider for ContactsContract — used in Robolectric tests.
 *
 * Key constants (from android.jar inspection):
 *   ContactsContract.Contacts.DISPLAY_NAME_PRIMARY  = "display_name"
 *   ContactsContract.CommonDataKinds.Phone.NUMBER   = "data1"
 *   ContactsContract.CommonDataKinds.Phone.CONTACT_ID = "contact_id"
 *   ContactsContract.CommonDataKinds.Email.ADDRESS  = "data1"
 *   ContactsContract.CommonDataKinds.Email.CONTACT_ID = "contact_id"
 *   ContactsContract.CommonDataKinds.StructuredName.DISPLAY_NAME = "data1"
 *
 * The data table stores data1 for all typed data (just like the real provider).
 * raw_contact._id is used as the "contact_id" in phone/email queries.
 */
class FakeContactsProvider : ContentProvider() {

    private lateinit var db: SQLiteDatabase

    private class DbHelper(ctx: android.content.Context?) : SQLiteOpenHelper(ctx, null, null, 1) {
        override fun onCreate(db: SQLiteDatabase) {
            db.execSQL("""
                CREATE TABLE raw_contacts (
                    _id INTEGER PRIMARY KEY AUTOINCREMENT,
                    account_type TEXT,
                    account_name TEXT,
                    deleted INTEGER DEFAULT 0
                )
            """.trimIndent())
            // data1 stores the primary value for all mimetypes (name, phone number, email address)
            db.execSQL("""
                CREATE TABLE data (
                    _id INTEGER PRIMARY KEY AUTOINCREMENT,
                    raw_contact_id INTEGER NOT NULL,
                    mimetype TEXT NOT NULL,
                    data1 TEXT,
                    data2 TEXT,
                    data3 TEXT,
                    data4 TEXT
                )
            """.trimIndent())
        }
        override fun onUpgrade(db: SQLiteDatabase, old: Int, new: Int) {}
    }

    override fun onCreate(): Boolean {
        db = DbHelper(context).writableDatabase
        return true
    }

    override fun getType(uri: Uri): String? = null

    override fun insert(uri: Uri, values: ContentValues?): Uri? {
        if (values == null) return null
        return when {
            isRawContacts(uri) -> {
                val cv = ContentValues()
                cv.put("account_type", values.getAsString(ContactsContract.RawContacts.ACCOUNT_TYPE))
                cv.put("account_name", values.getAsString(ContactsContract.RawContacts.ACCOUNT_NAME))
                cv.put("deleted", 0)
                val id = db.insert("raw_contacts", null, cv)
                Uri.withAppendedPath(ContactsContract.RawContacts.CONTENT_URI, id.toString())
            }
            isData(uri) -> {
                val cv = ContentValues()
                cv.put("raw_contact_id", values.getAsLong(ContactsContract.Data.RAW_CONTACT_ID))
                val mimetype = values.getAsString(ContactsContract.Data.MIMETYPE) ?: ""
                cv.put("mimetype", mimetype)
                // data1 is the primary column for all mimetypes (DISPLAY_NAME, NUMBER, ADDRESS all = "data1")
                cv.put("data1", values.getAsString("data1"))
                val id = db.insert("data", null, cv)
                Uri.withAppendedPath(ContactsContract.Data.CONTENT_URI, id.toString())
            }
            else -> null
        }
    }

    override fun query(
        uri: Uri, projection: Array<String>?,
        selection: String?, selectionArgs: Array<String>?, sortOrder: String?
    ): Cursor? {
        return when {
            isContactsView(uri) -> queryContactsView(projection)
            isRawContacts(uri) -> db.query(
                "raw_contacts", projection,
                if (selection != null) "$selection AND deleted=0" else "deleted=0",
                selectionArgs, null, null, sortOrder
            )
            isPhoneData(uri) -> queryTypedData(
                ContactsContract.CommonDataKinds.Phone.CONTENT_ITEM_TYPE, selection, selectionArgs
            )
            isEmailData(uri) -> queryTypedData(
                ContactsContract.CommonDataKinds.Email.CONTENT_ITEM_TYPE, selection, selectionArgs
            )
            isData(uri) -> db.query("data", projection, selection, selectionArgs, null, null, sortOrder)
            else -> null
        }
    }

    /**
     * Returns contacts view: _id and display_name (DISPLAY_NAME_PRIMARY = "display_name").
     * Honors the requested projection so callers using getString(0) with a single-column
     * projection get the expected value.
     */
    private fun queryContactsView(projection: Array<String>?): Cursor {
        // Build SELECT columns from the projection (default: _id and display_name)
        val selectCols = if (projection.isNullOrEmpty()) {
            "rc._id AS _id, COALESCE(d.data1, '') AS display_name"
        } else {
            projection.joinToString(", ") { col ->
                when (col) {
                    "_id" -> "rc._id AS _id"
                    "display_name" -> "COALESCE(d.data1, '') AS display_name"
                    else -> "''"  // unknown columns return empty
                }
            }
        }
        val sql = """
            SELECT $selectCols
            FROM raw_contacts rc
            LEFT JOIN data d ON d.raw_contact_id = rc._id
                AND d.mimetype = '${ContactsContract.CommonDataKinds.StructuredName.CONTENT_ITEM_TYPE}'
            WHERE rc.deleted = 0
            GROUP BY rc._id
        """.trimIndent()
        return db.rawQuery(sql, null)
    }

    /**
     * Queries data rows filtered by mimetype. Handles "contact_id = ?" selection by
     * translating to "raw_contact_id = ?" (our raw_contact._id doubles as contact_id).
     * Returns rows with data1 aliased as "data1" and raw_contact_id aliased as "contact_id".
     */
    private fun queryTypedData(
        mimetype: String, selection: String?, selectionArgs: Array<String>?
    ): Cursor {
        val where = buildString {
            append("mimetype = ?")
            if (!selection.isNullOrBlank()) {
                // Translate "contact_id = ?" to "raw_contact_id = ?"
                val xlated = selection.replace("contact_id", "raw_contact_id")
                append(" AND $xlated")
            }
        }
        val args: Array<String> = if (selectionArgs != null) {
            arrayOf(mimetype, *selectionArgs)
        } else {
            arrayOf(mimetype)
        }
        // Return data1 column (used for both NUMBER and ADDRESS) plus raw_contact_id as contact_id
        return db.rawQuery(
            "SELECT data1, raw_contact_id AS contact_id FROM data WHERE $where", args
        )
    }

    override fun delete(uri: Uri, selection: String?, selectionArgs: Array<String>?): Int {
        return when {
            isRawContacts(uri) -> {
                // Soft-delete all raw contacts (ignore selection for simplicity in tests)
                val cv = ContentValues()
                cv.put("deleted", 1)
                val count = db.update("raw_contacts", cv, null, null)
                db.delete("data", null, null)
                count
            }
            isData(uri) -> db.delete("data", selection, selectionArgs)
            else -> 0
        }
    }

    override fun update(uri: Uri, values: ContentValues?, selection: String?, selectionArgs: Array<String>?): Int = 0

    override fun applyBatch(operations: ArrayList<ContentProviderOperation>): Array<ContentProviderResult> {
        val results = Array<ContentProviderResult>(operations.size) { ContentProviderResult(0) }
        for ((i, op) in operations.withIndex()) {
            results[i] = op.apply(this, results, i)
        }
        return results
    }

    // URI matchers — order matters: check more specific paths first
    private fun isContactsView(uri: Uri) =
        uri.authority == ContactsContract.AUTHORITY &&
        uri.path?.let { it.startsWith("/contacts") && !it.startsWith("/contacts_corp") } == true

    private fun isRawContacts(uri: Uri) =
        uri.authority == ContactsContract.AUTHORITY && uri.path?.startsWith("/raw_contacts") == true

    private fun isPhoneData(uri: Uri) =
        uri.authority == ContactsContract.AUTHORITY && uri.path?.startsWith("/data/phones") == true

    private fun isEmailData(uri: Uri) =
        uri.authority == ContactsContract.AUTHORITY && uri.path?.startsWith("/data/emails") == true

    private fun isData(uri: Uri) =
        uri.authority == ContactsContract.AUTHORITY && uri.path?.startsWith("/data") == true
}
