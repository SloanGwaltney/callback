package main

// @scenario [features/call-reminders.feature#cancelled-no-reminder] No reminder for a cancelled call
// @scenario [features/call-reminders.feature#default-lead-time] Reminder fires at the default lead time
func TestOnTick() {
	// @given [features/call-reminders.feature] I had a call scheduled for 3:00 PM
	// @given [features/call-reminders.feature] I cancelled the call
	// @when [features/call-reminders.feature] the time reaches 2:45 PM
	// @then [features/call-reminders.feature] I do not receive a reminder
	// @then [features/call-reminders.feature#default-lead-time] I receive a reminder
}
