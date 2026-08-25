package be.heister.simplepresent

import android.app.AlarmManager
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import org.json.JSONArray

class ReminderBootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val pendingResult = goAsync()
        try {
            val preferences = context.getSharedPreferences("simple_present_reminders", Context.MODE_PRIVATE)
            val payloads = JSONArray(preferences.getString("payloads", "[]"))
            val alarmManager = context.getSystemService(Context.ALARM_SERVICE) as AlarmManager
            for (index in 0 until payloads.length()) {
                val payload = payloads.optJSONObject(index) ?: continue
                val triggerAt = payload.optLong("triggerAtMillis", 0L)
                if (triggerAt <= System.currentTimeMillis()) continue
                val taskId = payload.optString("id")
                val reminderIntent = Intent(context, ReminderReceiver::class.java).apply {
                    putExtra(ReminderReceiver.EXTRA_TASK_ID, taskId)
                    putExtra(ReminderReceiver.EXTRA_TITLE, payload.optString("title"))
                    putExtra(ReminderReceiver.EXTRA_BODY, payload.optString("body"))
                }
                val pending = PendingIntent.getBroadcast(
                    context, ReminderReceiver.requestCode(taskId), reminderIntent,
                    PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
                )
                try { alarmManager.setExactAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, triggerAt, pending) }
                catch (_: SecurityException) { alarmManager.setAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, triggerAt, pending) }
            }
        } finally {
            pendingResult.finish()
        }
    }
}