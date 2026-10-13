package jobs

import (
	"testing"
	"time"

	"callback/internal/store"
)

// seedNextAction seeds a job at company in status with a next action expected on date, or with no
// expected date when date is "".
func (h *harness) seedNextAction(company string, status Status, text, date string) int64 {
	return h.seed(store.Job{
		Company: company, Status: string(status), NextAction: nullString(text), NextActionDate: nullString(date),
	})
}

func (h *harness) mustSetNextAction(t *testing.T, id int64, text, date string) Job {
	t.Helper()
	job, err := h.svc.SetNextAction(h.ctx, id, text, date)
	if err != nil {
		t.Fatal(err)
	}
	return job
}

// nextActionText is a job's or card's next action text, or "" when it has none.
func nextActionText(a *NextAction) string {
	if a == nil {
		return ""
	}
	return a.Text
}

func nextActionDate(a *NextAction) string {
	if a == nil {
		return ""
	}
	return a.ExpectedDate
}

func nextActionDueOf(a *NextAction) NextActionDue {
	if a == nil {
		return DueNone
	}
	return a.Due
}

// @scenario [features/job-next-action.feature#set-next-action] Set a next action with an expected date
func TestSetNextAction(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with no next action
	id := h.acme(StatusApplied)

	// @when [features/job-next-action.feature] I set the next action to "Waiting for a call back from the recruiter" expected on 2026-10-14
	h.mustSetNextAction(t, id, "Waiting for a call back from the recruiter", "2026-10-14")

	got := h.job(t, id).NextAction
	// @then [features/job-next-action.feature] the job's next action is "Waiting for a call back from the recruiter"
	if nextActionText(got) != "Waiting for a call back from the recruiter" {
		t.Errorf("next action = %+v", got)
	}
	// @and [features/job-next-action.feature#set-next-action] its expected date is 2026-10-14
	if nextActionDate(got) != "2026-10-14" {
		t.Errorf("expected date = %q, want 2026-10-14", nextActionDate(got))
	}
}

// @scenario [features/job-next-action.feature#set-next-action-no-date] Set a next action without an expected date
func TestSetNextActionWithoutDate(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with no next action
	id := h.acme(StatusApplied)

	// @when [features/job-next-action.feature] I set the next action to "Send a thank-you email" with no expected date
	h.mustSetNextAction(t, id, "Send a thank-you email", "")

	got := h.job(t, id).NextAction
	// @then [features/job-next-action.feature] the job's next action is "Send a thank-you email"
	if nextActionText(got) != "Send a thank-you email" {
		t.Errorf("next action = %+v", got)
	}
	// @and [features/job-next-action.feature#set-next-action-no-date] it has no expected date
	if nextActionDate(got) != "" {
		t.Errorf("expected date = %q, want none", nextActionDate(got))
	}
}

// @scenario [features/job-next-action.feature#next-action-needs-text] A next action can't be saved without saying what it is
func TestNextActionNeedsText(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with no next action
	id := h.acme(StatusApplied)

	// @when [features/job-next-action.feature] I try to set a next action expected on 2026-10-14 with nothing but spaces written
	_, err := h.svc.SetNextAction(h.ctx, id, "   ", "2026-10-14")

	// @then [features/job-next-action.feature] the next action is not saved
	if got := h.job(t, id).NextAction; got != nil {
		t.Errorf("next action = %+v, want none", got)
	}
	// @and [features/job-next-action.feature] I am told to write what the next action is
	wantValidation(t, err, "write what the next action is")
}

// @scenario [features/job-next-action.feature#next-action-past-date] The expected date can be in the past
func TestNextActionPastDate(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-next-action.feature] today is 2026-10-09
	h.setToday(2026, time.October, 9)
	// @and [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with no next action
	id := h.acme(StatusApplied)

	// @when [features/job-next-action.feature] I set the next action to "Follow up on application" expected on 2026-10-05
	h.mustSetNextAction(t, id, "Follow up on application", "2026-10-05")

	got := h.job(t, id).NextAction
	// @then [features/job-next-action.feature] the job's next action is "Follow up on application"
	if nextActionText(got) != "Follow up on application" {
		t.Errorf("next action = %+v", got)
	}
	// @and [features/job-next-action.feature] its expected date is 2026-10-05
	if nextActionDate(got) != "2026-10-05" {
		t.Errorf("expected date = %q, want 2026-10-05", nextActionDate(got))
	}
}

// @scenario [features/job-next-action.feature#next-action-any-status] A next action can be set on a job in any status
func TestNextActionAnyStatus(t *testing.T) {
	for _, status := range Statuses {
		t.Run(status.Label(), func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" in "<status>"
			id := h.acme(status)
			h.seed(store.Job{Company: "Globex", Status: string(status)})

			// @when [features/job-next-action.feature] I set the next action to "Check the posting is still open" expected on 2026-10-14
			h.mustSetNextAction(t, id, "Check the posting is still open", "2026-10-14")

			// @then [features/job-next-action.feature] the job's next action is "Check the posting is still open"
			if got := h.job(t, id).NextAction; nextActionText(got) != "Check the posting is still open" {
				t.Errorf("next action = %+v", got)
			}
			// @and [features/job-next-action.feature] the card stays in the "<status>" column
			if _, in := h.card(t, id); in != status {
				t.Errorf("card is in %q, want %q", in, status)
			}
			if got := h.column(t, status); len(got) != 2 || got[0] != "Acme" {
				t.Errorf("%s column = %v, want Acme still first", status.Label(), got)
			}
		})
	}
}

// @scenario [features/job-next-action.feature#edit-next-action] Change the next action and its expected date
func TestEditNextAction(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
	id := h.seedNextAction("Acme", StatusOffer, "Waiting for a call back", "2026-10-14")

	// @when [features/job-next-action.feature] I change the next action to "Expecting an email with the offer details" expected on 2026-10-16
	h.mustSetNextAction(t, id, "Expecting an email with the offer details", "2026-10-16")

	got := h.job(t, id).NextAction
	// @then [features/job-next-action.feature] the job's next action is "Expecting an email with the offer details"
	if nextActionText(got) != "Expecting an email with the offer details" {
		t.Errorf("next action = %+v", got)
	}
	// @and [features/job-next-action.feature] its expected date is 2026-10-16
	if nextActionDate(got) != "2026-10-16" {
		t.Errorf("expected date = %q, want 2026-10-16", nextActionDate(got))
	}
}

// @scenario [features/job-next-action.feature#clear-next-action-date] Clear only the expected date
func TestClearNextActionDate(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
	id := h.seedNextAction("Acme", StatusApplied, "Waiting for a call back", "2026-10-14")

	// @when [features/job-next-action.feature#clear-next-action-date] I clear the expected date
	h.mustSetNextAction(t, id, "Waiting for a call back", "")

	got := h.job(t, id).NextAction
	// @then [features/job-next-action.feature#clear-next-action-date] the job's next action is still "Waiting for a call back"
	if nextActionText(got) != "Waiting for a call back" {
		t.Errorf("next action = %+v", got)
	}
	// @and [features/job-next-action.feature#clear-next-action-date] it has no expected date
	if nextActionDate(got) != "" {
		t.Errorf("expected date = %q, want none", nextActionDate(got))
	}
}

// @scenario [features/job-next-action.feature#clear-next-action] Clear the next action once it's done
func TestClearNextAction(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
	id := h.seedNextAction("Acme", StatusApplied, "Waiting for a call back", "2026-10-14")

	// @when [features/job-next-action.feature#clear-next-action] I clear the next action
	if _, err := h.svc.ClearNextAction(h.ctx, id); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-next-action.feature#clear-next-action] the job has no next action
	if got := h.job(t, id).NextAction; got != nil {
		t.Errorf("next action = %+v, want none", got)
	}
	// @and [features/job-next-action.feature#clear-next-action] the card no longer shows a next action
	if card, _ := h.card(t, id); card.NextAction != nil {
		t.Errorf("card's next action = %+v, want none", card.NextAction)
	}
}

func TestNextActionIsTrimmed(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusApplied)
	job := h.mustSetNextAction(t, id, "  Send a thank-you email \n", "")
	if job.NextAction == nil || job.NextAction.Text != "Send a thank-you email" {
		t.Errorf("next action = %+v, want it trimmed", job.NextAction)
	}
}

func TestNextActionRejectsBadDates(t *testing.T) {
	for _, date := range []string{"10/14/2026", "2026-02-30", "tomorrow"} {
		t.Run(date, func(t *testing.T) {
			h := newHarness(t)
			id := h.acme(StatusApplied)
			_, err := h.svc.SetNextAction(h.ctx, id, "Call", date)
			wantValidation(t, err, "the expected date must be a date like 2026-10-14")
			if got := h.job(t, id).NextAction; got != nil {
				t.Errorf("next action = %+v, want none", got)
			}
		})
	}
}

func TestNextActionUnknownJob(t *testing.T) {
	h := newHarness(t)
	if _, err := h.svc.SetNextAction(h.ctx, 99, "Call", ""); err != ErrJobNotFound {
		t.Errorf("SetNextAction err = %v, want ErrJobNotFound", err)
	}
	if _, err := h.svc.ClearNextAction(h.ctx, 99); err != ErrJobNotFound {
		t.Errorf("ClearNextAction err = %v, want ErrJobNotFound", err)
	}
}

// @scenario [features/job-next-action.feature#card-shows-next-action] The card shows the next action and its expected date
func TestCardShowsNextAction(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-next-action.feature] today is 2026-10-09
	h.setToday(2026, time.October, 9)
	// @and [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
	id := h.seedNextAction("Acme", StatusApplied, "Waiting for a call back", "2026-10-14")

	// @when [features/job-next-action.feature] I open the job search board
	card, _ := h.card(t, id)

	// @then [features/job-next-action.feature] the card shows "Waiting for a call back" expected on 2026-10-14
	want := NextAction{Text: "Waiting for a call back", ExpectedDate: "2026-10-14"}
	if card.NextAction == nil || *card.NextAction != want {
		t.Errorf("card's next action = %+v, want %+v", card.NextAction, want)
	}
}

// @scenario [features/job-next-action.feature#card-next-action-no-date] The card shows a next action with no expected date
func TestCardShowsNextActionWithoutDate(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Send a thank-you email" and no expected date
	id := h.seedNextAction("Acme", StatusApplied, "Send a thank-you email", "")

	// @when [features/job-next-action.feature] I open the job search board
	card, _ := h.card(t, id)

	// @then [features/job-next-action.feature] the card shows "Send a thank-you email" with no expected date
	if nextActionText(card.NextAction) != "Send a thank-you email" || nextActionDate(card.NextAction) != "" {
		t.Errorf("card's next action = %+v", card.NextAction)
	}
	// @and [features/job-next-action.feature] the next action is not marked as due or overdue
	if due := nextActionDueOf(card.NextAction); due != DueNone {
		t.Errorf("due = %q, want none", due)
	}
}

func TestCardMarksNextActionDue(t *testing.T) {
	// @scenario [features/job-next-action.feature#card-next-action-due-today] A next action expected today is marked as due
	t.Run("due today", func(t *testing.T) {
		h := newHarness(t)
		// @given [features/job-next-action.feature] today is 2026-10-14
		h.setToday(2026, time.October, 14)
		// @and [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
		id := h.seedNextAction("Acme", StatusApplied, "Waiting for a call back", "2026-10-14")

		// @when [features/job-next-action.feature] I open the job search board
		card, _ := h.card(t, id)

		// @then [features/job-next-action.feature] the card marks the next action as due today
		if due := nextActionDueOf(card.NextAction); due != DueToday {
			t.Errorf("due = %q, want %q", due, DueToday)
		}
	})

	// @scenario [features/job-next-action.feature#card-next-action-overdue] A next action whose expected date has passed is marked as overdue
	t.Run("overdue", func(t *testing.T) {
		h := newHarness(t)
		// @given [features/job-next-action.feature] today is 2026-10-15
		h.setToday(2026, time.October, 15)
		// @and [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
		id := h.seedNextAction("Acme", StatusApplied, "Waiting for a call back", "2026-10-14")

		// @when [features/job-next-action.feature] I open the job search board
		card, _ := h.card(t, id)

		// @then [features/job-next-action.feature] the card marks the next action as overdue
		if due := nextActionDueOf(card.NextAction); due != DueOverdue {
			t.Errorf("due = %q, want %q", due, DueOverdue)
		}
	})

	// @scenario [features/job-next-action.feature#card-next-action-upcoming] A next action expected later isn't marked
	t.Run("upcoming", func(t *testing.T) {
		h := newHarness(t)
		// @given [features/job-next-action.feature] today is 2026-10-09
		h.setToday(2026, time.October, 9)
		// @and [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
		id := h.seedNextAction("Acme", StatusApplied, "Waiting for a call back", "2026-10-14")

		// @when [features/job-next-action.feature] I open the job search board
		card, _ := h.card(t, id)

		// @then [features/job-next-action.feature] the next action is not marked as due or overdue
		if due := nextActionDueOf(card.NextAction); due != DueNone {
			t.Errorf("due = %q, want none", due)
		}
	})
}

// @scenario [features/job-next-action.feature#closed-card-no-due-marks] A closed job's next action is never marked as due or overdue
func TestClosedCardNoDueMarks(t *testing.T) {
	for _, expected := range []string{"2026-10-14", "2026-10-10"} {
		t.Run(expected, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-next-action.feature] today is 2026-10-14
			h.setToday(2026, time.October, 14)
			// @and [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" in "Closed" with the next action "Ask for feedback" expected on <expected>
			id := h.seedNextAction("Acme", StatusClosed, "Ask for feedback", expected)

			// @when [features/job-next-action.feature] I open the job search board
			card, _ := h.card(t, id)

			// @then [features/job-next-action.feature] the card shows "Ask for feedback"
			if nextActionText(card.NextAction) != "Ask for feedback" {
				t.Errorf("card's next action = %+v", card.NextAction)
			}
			// @but [features/job-next-action.feature] the next action is not marked as due or overdue
			if due := nextActionDueOf(card.NextAction); due != DueNone {
				t.Errorf("due = %q, want none", due)
			}
			// The job, opened, isn't marked either.
			if due := nextActionDueOf(h.job(t, id).NextAction); due != DueNone {
				t.Errorf("job's due = %q, want none", due)
			}
		})
	}
}

// @scenario [features/job-next-action.feature#card-no-next-action] A job with no next action shows none on its card
func TestCardNoNextAction(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with no next action
	id := h.acme(StatusApplied)

	// @when [features/job-next-action.feature] I open the job search board
	card, _ := h.card(t, id)

	// @then [features/job-next-action.feature] the card doesn't show a next action
	if card.NextAction != nil {
		t.Errorf("card's next action = %+v, want none", card.NextAction)
	}
}

// @scenario [features/job-next-action.feature#view-next-action] See the next action when opening the card
func TestViewNextAction(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
	id := h.seedNextAction("Acme", StatusApplied, "Waiting for a call back", "2026-10-14")

	// @when [features/job-next-action.feature] I open the card
	job := h.job(t, id)

	// @then [features/job-next-action.feature] I see the next action "Waiting for a call back" and its expected date 2026-10-14
	if nextActionText(job.NextAction) != "Waiting for a call back" || nextActionDate(job.NextAction) != "2026-10-14" {
		t.Errorf("next action = %+v", job.NextAction)
	}
}

// The board filter runs in the frontend (Board.test.tsx and boardView.test.ts), over each card's
// due mark. These tests check the backend marks the cards those scenarios start from.
func TestOverdueMarksForTheFilter(t *testing.T) {
	// dues returns each company's card's due mark.
	dues := func(t *testing.T, h *harness) map[string]NextActionDue {
		t.Helper()
		got := map[string]NextActionDue{}
		for _, c := range h.board(t).Columns {
			for _, card := range c.Cards {
				got[card.Company] = nextActionDueOf(card.NextAction)
			}
		}
		return got
	}
	check := func(t *testing.T, got, want map[string]NextActionDue) {
		t.Helper()
		for company, due := range want {
			if got[company] != due {
				t.Errorf("%s is %q, want %q", company, got[company], due)
			}
		}
	}

	// @scenario [features/job-next-action.feature#filter-overdue] Show only overdue jobs
	// @scenario [features/job-next-action.feature#filter-not-overdue] Show only jobs that aren't overdue
	t.Run("overdue, later, and none", func(t *testing.T) {
		h := newHarness(t)
		// @given [features/job-next-action.feature] today is 2026-10-14
		h.setToday(2026, time.October, 14)
		// @and [features/job-next-action.feature] I have a job at "Acme" with a next action expected on 2026-10-10
		h.seedNextAction("Acme", StatusApplied, "Follow up", "2026-10-10")
		// @and [features/job-next-action.feature] a job at "Globex" with a next action expected on 2026-10-20
		h.seedNextAction("Globex", StatusApplied, "Follow up", "2026-10-20")
		// @and [features/job-next-action.feature] a job at "Initech" with no next action
		h.seed(store.Job{Company: "Initech", Status: string(StatusApplied)})

		check(t, dues(t, h), map[string]NextActionDue{"Acme": DueOverdue, "Globex": DueNone, "Initech": DueNone})
	})

	// @scenario [features/job-next-action.feature#filter-show-all] Show every job again
	t.Run("overdue and none", func(t *testing.T) {
		h := newHarness(t)
		// @given [features/job-next-action.feature] today is 2026-10-14
		h.setToday(2026, time.October, 14)
		// @and [features/job-next-action.feature] I have a job at "Acme" with a next action expected on 2026-10-10
		h.seedNextAction("Acme", StatusApplied, "Follow up", "2026-10-10")
		// @and [features/job-next-action.feature] a job at "Globex" with no next action
		h.seed(store.Job{Company: "Globex", Status: string(StatusApplied)})

		check(t, dues(t, h), map[string]NextActionDue{"Acme": DueOverdue, "Globex": DueNone})
	})

	// @scenario [features/job-next-action.feature#filter-due-today-not-overdue] A job due today isn't overdue yet
	t.Run("due today", func(t *testing.T) {
		h := newHarness(t)
		// @given [features/job-next-action.feature] today is 2026-10-14
		h.setToday(2026, time.October, 14)
		// @and [features/job-next-action.feature] I have a job at "Acme" with a next action expected on 2026-10-14
		h.seedNextAction("Acme", StatusApplied, "Follow up", "2026-10-14")

		check(t, dues(t, h), map[string]NextActionDue{"Acme": DueToday})
	})

	// @scenario [features/job-next-action.feature#filter-closed-never-overdue] A closed job is never overdue
	t.Run("closed", func(t *testing.T) {
		h := newHarness(t)
		// @given [features/job-next-action.feature] today is 2026-10-14
		h.setToday(2026, time.October, 14)
		// @and [features/job-next-action.feature] I have a job at "Acme" in "Closed" with a next action expected on 2026-10-10
		h.seedNextAction("Acme", StatusClosed, "Ask for feedback", "2026-10-10")

		check(t, dues(t, h), map[string]NextActionDue{"Acme": DueNone})
	})

	// @scenario [features/job-next-action.feature#filter-keeps-columns] Filtered cards stay in their own columns and the counts show how many are shown
	t.Run("several columns", func(t *testing.T) {
		h := newHarness(t)
		// @given [features/job-next-action.feature] today is 2026-10-14
		h.setToday(2026, time.October, 14)
		// @and [features/job-next-action.feature] I have 2 overdue jobs in "Applied" and 1 overdue job in "Interviewing"
		h.seedNextAction("Acme", StatusApplied, "Follow up", "2026-10-10")
		h.seedNextAction("Globex", StatusApplied, "Follow up", "2026-10-13")
		h.seedNextAction("Initech", StatusInterviewing, "Follow up", "2026-10-01")
		// @and [features/job-next-action.feature] 3 jobs in "Applied" that aren't overdue
		h.seedNextAction("Hooli", StatusApplied, "Follow up", "2026-10-14")
		h.seedNextAction("Umbrella", StatusApplied, "Follow up", "2026-10-30")
		h.seed(store.Job{Company: "Vandelay", Status: string(StatusApplied)})

		check(t, dues(t, h), map[string]NextActionDue{
			"Acme": DueOverdue, "Globex": DueOverdue, "Initech": DueOverdue,
			"Hooli": DueToday, "Umbrella": DueNone, "Vandelay": DueNone,
		})
	})

	// @scenario [features/job-next-action.feature#filter-with-search] The filter and the search both apply
	t.Run("search", func(t *testing.T) {
		h := newHarness(t)
		// @given [features/job-next-action.feature] today is 2026-10-14
		h.setToday(2026, time.October, 14)
		// @and [features/job-next-action.feature] I have an overdue job for "Software Engineer" at "Acme"
		h.seedNextAction("Acme", StatusApplied, "Follow up", "2026-10-10")
		// @and [features/job-next-action.feature] an overdue job for "Designer" at "Globex"
		h.seed(store.Job{Company: "Globex", Role: "Designer", Status: string(StatusApplied), NextAction: nullString("Follow up"), NextActionDate: nullString("2026-10-10")})
		// @and [features/job-next-action.feature] a job for "Software Engineer" at "Initech" that isn't overdue
		h.seed(store.Job{Company: "Initech", Status: string(StatusApplied)})

		check(t, dues(t, h), map[string]NextActionDue{"Acme": DueOverdue, "Globex": DueOverdue, "Initech": DueNone})
	})

	// @scenario [features/job-next-action.feature#filter-no-matches] Told when no jobs match the filter
	t.Run("none overdue", func(t *testing.T) {
		h := newHarness(t)
		// @given [features/job-next-action.feature] today is 2026-10-14
		h.setToday(2026, time.October, 14)
		// @and [features/job-next-action.feature] none of my jobs are overdue
		h.seedNextAction("Acme", StatusApplied, "Follow up", "2026-10-20")
		h.seed(store.Job{Company: "Globex", Status: string(StatusApplied)})

		check(t, dues(t, h), map[string]NextActionDue{"Acme": DueNone, "Globex": DueNone})
	})
}

// @scenario [features/job-next-action.feature#move-keeps-next-action] Moving a card keeps its next action
func TestMoveKeepsNextAction(t *testing.T) {
	for _, to := range []Status{StatusOffer, StatusApplied, StatusClosed} {
		t.Run(to.Label(), func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing" with the next action "Waiting for a call back" expected on 2026-10-14
			id := h.seedNextAction("Acme", StatusInterviewing, "Waiting for a call back", "2026-10-14")

			// @when [features/job-next-action.feature] I move the card to "<to>"
			move := Move{JobID: id, Status: to}
			if to == StatusClosed {
				move.ClosingReason = ReasonRejected
			}
			mustMove(t, h, move)

			got := h.job(t, id).NextAction
			// @then [features/job-next-action.feature#move-keeps-next-action] the job's next action is still "Waiting for a call back"
			if nextActionText(got) != "Waiting for a call back" {
				t.Errorf("next action = %+v", got)
			}
			// @and [features/job-next-action.feature] its expected date is still 2026-10-14
			if nextActionDate(got) != "2026-10-14" {
				t.Errorf("expected date = %q, want 2026-10-14", nextActionDate(got))
			}
		})
	}
}

func TestEditingDetailsKeepsNextAction(t *testing.T) {
	h := newHarness(t)
	id := h.seedNextAction("Acme", StatusApplied, "Waiting for a call back", "2026-10-14")
	if _, err := h.svc.UpdateJob(h.ctx, id, Details{Company: "Acme", Role: "Engineer"}); err != nil {
		t.Fatal(err)
	}
	if got := h.job(t, id).NextAction; nextActionText(got) != "Waiting for a call back" || nextActionDate(got) != "2026-10-14" {
		t.Errorf("next action = %+v after editing details", got)
	}
}

func TestNextActionDue(t *testing.T) {
	const today = "2026-10-14"
	for _, tt := range []struct {
		date   string
		status Status
		want   NextActionDue
	}{
		{"", StatusApplied, DueNone},
		{"2026-10-13", StatusApplied, DueOverdue},
		{"2026-10-14", StatusApplied, DueToday},
		{"2026-10-15", StatusApplied, DueNone},
		{"2026-10-13", StatusSaved, DueOverdue},
		{"2026-10-13", StatusClosed, DueNone},
		{"2026-10-14", StatusClosed, DueNone},
	} {
		if got := nextActionDue(tt.date, tt.status, today); got != tt.want {
			t.Errorf("nextActionDue(%q, %s) = %q, want %q", tt.date, tt.status, got, tt.want)
		}
	}
}
