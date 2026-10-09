package main

// @scenario [features/call-reminders.feature#cancelled-no-reminder] No reminder for a cancelled call
// @scenario [features/call-reminders.feature#default-lead-time] Reminder fires at the default lead time
// @when [features/call-reminders.feature] the time reaches 2:45 PM
func onTick() {
	// @given [features/call-reminders.feature] I had a call scheduled for 3:00 PM
	// @and [features/call-reminders.feature] I cancelled the call
	// @then [features/call-reminders.feature] I do not receive a reminder
	// @then [features/call-reminders.feature#default-lead-time] I receive a reminder
	_ = "// @then [features/call-reminders.feature] not a comment, so it's ignored"
}
