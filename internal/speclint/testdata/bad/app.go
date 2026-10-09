package main

// @scenario [features/call-reminders.feature#cancelled-no-reminder] No reminder for cancelled calls
// @scenario [features/call-reminders.feature] No reminder for a cancelled call
// @scenario [features/call-reminders.feature#nope] Unknown
// @scenario [features/missing.feature#x] Missing file
// @scenario [features\call-reminders.feature#cancelled-no-reminder] No reminder for a cancelled call
// @given [features/call-reminders.feature] I had a call scheduled for 3 PM
// @given [features/call-reminders.feature] I cancelled the call
// @then [features/call-reminders.feature] I receive an SMS reminder
// @then [features/call-reminders.feature#sms-reminder] I receive an SMS reminder
func app() {}
