package jobs

import (
	"errors"
	"slices"
	"testing"
	"time"

	"callback/internal/store"
)

// addAcmeOn adds a job at Acme through the service with today set to date, so its history starts
// on that day.
func (h *harness) addAcmeOn(t *testing.T, status Status, year int, month time.Month, day int) int64 {
	t.Helper()
	h.setToday(year, month, day)
	job, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Software Engineer"}, status)
	if err != nil {
		t.Fatal(err)
	}
	return job.ID
}

// movedOn seeds a job at Acme in status whose history says it moved there on date.
func (h *harness) movedOn(status Status, date string) int64 {
	id := h.acme(status)
	h.seedStatusChange(id, status, date)
	return id
}

// daysInStatusOf returns a card's days in status, failing if it has none.
func (h *harness) daysInStatusOf(t *testing.T, id int64) int {
	t.Helper()
	card, _ := h.card(t, id)
	if card.DaysInStatus == nil {
		t.Fatalf("card %d has no days in status", id)
	}
	return *card.DaysInStatus
}

// @scenario [features/job-status-history.feature#history-starts-when-added] A new job's history starts with the status it was added to
func TestHistoryStartsWhenAdded(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] today is 2026-10-09
	h.setToday(2026, time.October, 9)

	// @when [features/job-status-history.feature] I add a job for "Software Engineer" at "Acme" to the "Saved" column
	job, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Software Engineer"}, StatusSaved)
	if err != nil {
		t.Fatal(err)
	}
	// @when [features/job-status-history.feature] I open the card
	got := h.history(t, job.ID)

	// @then [features/job-status-history.feature] the status history shows "Saved" on 2026-10-09
	if want := []string{"Saved 2026-10-09"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

func TestAddJobToAnotherColumnStartsHistoryThere(t *testing.T) {
	h := newHarness(t)
	job, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Software Engineer"}, StatusApplied)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := h.history(t, job.ID), []string{"Applied 2026-10-09"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

func TestAddJobRollsBackWhenHistoryFails(t *testing.T) {
	h := newHarness(t)
	h.store.fail = "AddStatusChange"
	_, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Software Engineer"}, "")
	if !errors.Is(err, errFake) {
		t.Fatalf("err = %v, want errFake", err)
	}
	if len(h.store.jobs) != 0 {
		t.Errorf("%d jobs saved, want none", len(h.store.jobs))
	}
}

// @scenario [features/job-status-history.feature#history-records-move] Moving a card to another status adds it to the history
func TestHistoryRecordsMove(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] I added a job for "Software Engineer" at "Acme" to "Saved" on 2026-10-01
	id := h.addAcmeOn(t, StatusSaved, 2026, time.October, 1)
	// @given [features/job-status-history.feature] today is 2026-10-09
	h.setToday(2026, time.October, 9)

	// @when [features/job-status-history.feature] I move the card to "Applied"
	mustMove(t, h, Move{JobID: id, Status: StatusApplied})
	// @when [features/job-status-history.feature] I open the card
	got := h.history(t, id)

	// @then [features/job-status-history.feature] the status history shows "Saved" on 2026-10-01, then "Applied" on 2026-10-09
	if want := []string{"Saved 2026-10-01", "Applied 2026-10-09"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

func TestMoveRollsBackWhenHistoryFails(t *testing.T) {
	h := newHarness(t)
	id := h.movedOn(StatusSaved, "2026-10-01")
	h.store.fail = "AddStatusChange"
	if err := h.svc.MoveJob(h.ctx, Move{JobID: id, Status: StatusApplied}); !errors.Is(err, errFake) {
		t.Fatalf("err = %v, want errFake", err)
	}
	h.store.fail = ""
	if _, status := h.card(t, id); status != StatusSaved {
		t.Errorf("card is in %q, want Saved", status)
	}
}

// @scenario [features/job-status-history.feature#history-oldest-first] The history lists every change from oldest to newest
func TestHistoryOldestFirst(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] I added a job for "Software Engineer" at "Acme" to "Saved" on 2026-10-01
	id := h.addAcmeOn(t, StatusSaved, 2026, time.October, 1)
	// @and [features/job-status-history.feature] I moved it to "Applied" on 2026-10-02
	h.setToday(2026, time.October, 2)
	mustMove(t, h, Move{JobID: id, Status: StatusApplied})
	// @and [features/job-status-history.feature] I moved it to "Interviewing" on 2026-10-06
	h.setToday(2026, time.October, 6)
	mustMove(t, h, Move{JobID: id, Status: StatusInterviewing})
	// @and [features/job-status-history.feature] I moved it to "Offer" on 2026-10-09
	h.setToday(2026, time.October, 9)
	mustMove(t, h, Move{JobID: id, Status: StatusOffer})

	// @when [features/job-status-history.feature] I open the card
	history := h.job(t, id).StatusHistory

	// @then [features/job-status-history.feature] the status history shows "Saved", "Applied", "Interviewing", and "Offer" in that order
	var statuses []Status
	for _, c := range history {
		statuses = append(statuses, c.Status)
	}
	if want := []Status{StatusSaved, StatusApplied, StatusInterviewing, StatusOffer}; !slices.Equal(statuses, want) {
		t.Errorf("statuses = %v, want %v", statuses, want)
	}
	// @and [features/job-status-history.feature] each status shows the date the job moved to it
	var dates []string
	for _, c := range history {
		dates = append(dates, c.Date)
	}
	if want := []string{"2026-10-01", "2026-10-02", "2026-10-06", "2026-10-09"}; !slices.Equal(dates, want) {
		t.Errorf("dates = %v, want %v", dates, want)
	}
}

// @scenario [features/job-status-history.feature#history-same-day-order] Several changes on the same day keep the order they happened in
func TestHistorySameDayOrder(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] I added a job for "Software Engineer" at "Acme" to "Saved" on 2026-10-09
	id := h.addAcmeOn(t, StatusSaved, 2026, time.October, 9)
	// @and [features/job-status-history.feature] I moved it to "Applied" and then to "Interviewing" on 2026-10-09
	mustMove(t, h, Move{JobID: id, Status: StatusApplied})
	mustMove(t, h, Move{JobID: id, Status: StatusInterviewing})

	// @when [features/job-status-history.feature] I open the card
	got := h.history(t, id)

	// @then [features/job-status-history.feature] the status history shows "Saved", "Applied", and "Interviewing" in that order
	if want := []string{"Saved 2026-10-09", "Applied 2026-10-09", "Interviewing 2026-10-09"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

// @scenario [features/job-status-history.feature#history-records-moving-back] Moving a card back to an earlier status is recorded too
func TestHistoryRecordsMovingBack(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] I have a job for "Software Engineer" at "Acme" that moved from "Applied" to "Interviewing"
	id := h.acme(StatusInterviewing)
	h.seedStatusChange(id, StatusApplied, "2026-10-01")
	h.seedStatusChange(id, StatusInterviewing, "2026-10-05")

	// @when [features/job-status-history.feature] I move the card back to "Applied"
	mustMove(t, h, Move{JobID: id, Status: StatusApplied})
	// @when [features/job-status-history.feature] I open the card
	got := h.history(t, id)

	// @then [features/job-status-history.feature] the status history shows "Applied", "Interviewing", and "Applied" in that order
	if want := []string{"Applied 2026-10-01", "Interviewing 2026-10-05", "Applied 2026-10-09"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

// @scenario [features/job-status-history.feature#history-records-closing-reason] Closing a job records its closing reason in the history
func TestHistoryRecordsClosingReason(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
	id := h.movedOn(StatusInterviewing, "2026-10-05")

	// @when [features/job-status-history.feature#history-records-closing-reason] I move the card to "Closed"
	// @and [features/job-status-history.feature] I choose the closing reason "Rejected"
	mustMove(t, h, Move{JobID: id, Status: StatusClosed, ClosingReason: ReasonRejected})
	// @when [features/job-status-history.feature] I open the card
	got := h.history(t, id)

	// @then [features/job-status-history.feature] the status history ends with "Closed" and the reason "Rejected"
	if want := []string{"Interviewing 2026-10-05", "Closed 2026-10-09 Rejected"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

func TestHistoryRecordsOtherClosingReasonText(t *testing.T) {
	h := newHarness(t)
	id := h.movedOn(StatusOffer, "2026-10-05")
	mustMove(t, h, Move{JobID: id, Status: StatusClosed, ClosingReason: ReasonOther, ClosingReasonOther: "  Company went through layoffs "})
	history := h.job(t, id).StatusHistory
	last := history[len(history)-1]
	if last.ClosingReason != ReasonOther || last.ClosingReasonText != "Company went through layoffs" {
		t.Errorf("last entry = %+v, want Other with the trimmed text", last)
	}
}

// @scenario [features/job-status-history.feature#history-keeps-original-closing-reason] Changing the closing reason later doesn't rewrite the history
func TestHistoryKeepsOriginalClosingReason(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] I closed a job for "Software Engineer" at "Acme" with the closing reason "No response"
	id := h.movedOn(StatusApplied, "2026-10-01")
	mustMove(t, h, Move{JobID: id, Status: StatusClosed, ClosingReason: ReasonNoResponse})

	// @when [features/job-status-history.feature] I change the closing reason to "Rejected"
	if err := h.svc.SetClosingReason(h.ctx, id, ReasonRejected, ""); err != nil {
		t.Fatal(err)
	}
	// @when [features/job-status-history.feature] I open the card
	job := h.job(t, id)

	// @then [features/job-status-history.feature] the card shows the closing reason "Rejected"
	if job.ClosingReasonText != "Rejected" {
		t.Errorf("closing reason = %q, want Rejected", job.ClosingReasonText)
	}
	// @but [features/job-status-history.feature] the status history still shows "Closed" with the reason "No response"
	if got, want := h.history(t, id), []string{"Applied 2026-10-01", "Closed 2026-10-09 No response"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

// @scenario [features/job-status-history.feature#history-ignores-reorder] Reordering a card within its column doesn't change the history
func TestHistoryIgnoresReorder(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] the "Saved" column holds the jobs at "Globex" and "Acme" in that order
	h.seed(store.Job{Company: "Globex"})
	id := h.acme(StatusSaved)
	// @and [features/job-status-history.feature] the job at "Acme" has "Saved" as its only status history
	h.seedStatusChange(id, StatusSaved, "2026-10-01")

	// @when [features/job-status-history.feature] I move the "Acme" card above the "Globex" card
	mustMove(t, h, Move{JobID: id, Status: StatusSaved, Index: 0})
	if got := h.column(t, StatusSaved); !slices.Equal(got, []string{"Acme", "Globex"}) {
		t.Fatalf("Saved = %v, want [Acme Globex]", got)
	}
	// @and [features/job-status-history.feature] I open the "Acme" card
	got := h.history(t, id)

	// @then [features/job-status-history.feature] the status history still shows only "Saved"
	if want := []string{"Saved 2026-10-01"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

func TestReorderWithinClosedDoesntChangeHistory(t *testing.T) {
	h := newHarness(t)
	h.seed(store.Job{Company: "Globex", Status: string(StatusClosed)})
	id := h.movedOn(StatusClosed, "2026-10-01")
	mustMove(t, h, Move{JobID: id, Status: StatusClosed, Index: 0})
	if got, want := h.history(t, id), []string{"Closed 2026-10-01 No response"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

// @scenario [features/job-status-history.feature#history-ignores-cancelled-close] Cancelling a move to Closed doesn't change the history
func TestHistoryIgnoresCancelledClose(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] I have a job for "Software Engineer" at "Acme" whose history ends with "Interviewing"
	id := h.acme(StatusInterviewing)
	h.seedStatusChange(id, StatusSaved, "2026-10-01")
	h.seedStatusChange(id, StatusInterviewing, "2026-10-05")

	// Cancelling the closing reason never calls MoveJob (Board.test.tsx, "leaves the history alone
	// when the close is cancelled"). A move to Closed with no reason is refused and records nothing
	// either.
	wantValidation(t, h.svc.MoveJob(h.ctx, Move{JobID: id, Status: StatusClosed}), "choose why the job was closed")

	// @when [features/job-status-history.feature] I open the card
	got := h.history(t, id)

	// @then [features/job-status-history.feature] the status history still ends with "Interviewing"
	if want := []string{"Saved 2026-10-01", "Interviewing 2026-10-05"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

// @scenario [features/job-status-history.feature#history-ignores-date-applied] Changing the date applied doesn't change the history
func TestHistoryIgnoresDateApplied(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] I moved a job for "Software Engineer" at "Acme" to "Applied" on 2026-10-09
	id := h.movedOn(StatusSaved, "2026-10-01")
	mustMove(t, h, Move{JobID: id, Status: StatusApplied})

	// @when [features/job-status-history.feature] I set the date applied to 2026-10-01
	_, err := h.svc.UpdateJob(h.ctx, id, Details{Company: "Acme", Role: "Software Engineer", DateApplied: "2026-10-01"})
	if err != nil {
		t.Fatal(err)
	}
	// @when [features/job-status-history.feature] I open the card
	got := h.history(t, id)

	// @then [features/job-status-history.feature] the status history still shows "Applied" on 2026-10-09
	if want := []string{"Saved 2026-10-01", "Applied 2026-10-09"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

// @scenario [features/job-status-history.feature#card-shows-days-in-status] The card shows how many days the job has been in its current status
func TestCardShowsDaysInStatus(t *testing.T) {
	tests := []struct {
		moved string
		shown string
		days  int
	}{
		{"2026-10-09", "Today", 0},
		{"2026-10-08", "1 day", 1},
		{"2026-10-01", "8 days", 8},
	}
	for _, tt := range tests {
		t.Run(tt.shown, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-status-history.feature] I moved a job for "Software Engineer" at "Acme" to "Applied" on <moved>
			id := h.movedOn(StatusApplied, tt.moved)
			// @given [features/job-status-history.feature] today is 2026-10-09
			h.setToday(2026, time.October, 9)

			// @when [features/job-status-history.feature] I open the job search board
			got := h.daysInStatusOf(t, id)

			// The card turns the count into "<shown>" (JobCard.test.tsx).
			// @then [features/job-status-history.feature] the card shows "<shown>"
			if got != tt.days {
				t.Errorf("days in status = %d, want %d", got, tt.days)
			}
		})
	}
}

// @scenario [features/job-status-history.feature#days-in-status-from-added] A job that hasn't moved counts from when it was added
func TestDaysInStatusFromAdded(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] I added a job for "Software Engineer" at "Acme" to "Saved" on 2026-10-06
	id := h.addAcmeOn(t, StatusSaved, 2026, time.October, 6)
	// @given [features/job-status-history.feature] today is 2026-10-09
	h.setToday(2026, time.October, 9)

	// @when [features/job-status-history.feature] I open the job search board
	got := h.daysInStatusOf(t, id)

	// @then [features/job-status-history.feature] the card shows "3 days"
	if got != 3 {
		t.Errorf("days in status = %d, want 3", got)
	}
}

// @scenario [features/job-status-history.feature#days-in-status-resets-on-move] Moving a card to another status starts the count again
func TestDaysInStatusResetsOnMove(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] I moved a job for "Software Engineer" at "Acme" to "Applied" on 2026-10-01
	id := h.movedOn(StatusApplied, "2026-10-01")
	// @given [features/job-status-history.feature] today is 2026-10-09
	h.setToday(2026, time.October, 9)
	if got := h.daysInStatusOf(t, id); got != 8 {
		t.Fatalf("before the move, days in status = %d, want 8", got)
	}

	// @when [features/job-status-history.feature] I move the card to "Interviewing"
	mustMove(t, h, Move{JobID: id, Status: StatusInterviewing})

	// @then [features/job-status-history.feature] the card shows "Today"
	if got := h.daysInStatusOf(t, id); got != 0 {
		t.Errorf("days in status = %d, want 0", got)
	}
}

// @scenario [features/job-status-history.feature#days-in-status-ignores-reorder] Reordering a card within its column doesn't reset the count
func TestDaysInStatusIgnoresReorder(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature] today is 2026-10-09
	h.setToday(2026, time.October, 9)
	// @and [features/job-status-history.feature] the "Applied" column holds the jobs at "Globex" and "Acme" in that order
	h.seed(store.Job{Company: "Globex", Status: string(StatusApplied)})
	id := h.acme(StatusApplied)
	// @and [features/job-status-history.feature] the job at "Acme" moved to "Applied" on 2026-10-01
	h.seedStatusChange(id, StatusApplied, "2026-10-01")

	// @when [features/job-status-history.feature] I move the "Acme" card above the "Globex" card
	mustMove(t, h, Move{JobID: id, Status: StatusApplied, Index: 0})

	// @then [features/job-status-history.feature] the "Acme" card shows "8 days"
	if got := h.column(t, StatusApplied); !slices.Equal(got, []string{"Acme", "Globex"}) {
		t.Fatalf("Applied = %v, want [Acme Globex]", got)
	}
	if got := h.daysInStatusOf(t, id); got != 8 {
		t.Errorf("days in status = %d, want 8", got)
	}
}

// @scenario [features/job-status-history.feature#days-in-status-any-column] Every column's cards show their days in status
func TestDaysInStatusAnyColumn(t *testing.T) {
	for _, status := range []string{"Saved", "Applied", "Interviewing", "Offer", "Closed"} {
		t.Run(status, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-status-history.feature] I moved a job for "Software Engineer" at "Acme" to "<status>" on 2026-10-01
			id := h.movedOn(statusFromLabel(t, status), "2026-10-01")
			// @given [features/job-status-history.feature] today is 2026-10-09
			h.setToday(2026, time.October, 9)

			// @when [features/job-status-history.feature] I open the job search board
			got := h.daysInStatusOf(t, id)

			// @then [features/job-status-history.feature] the card shows "8 days"
			if got != 8 {
				t.Errorf("days in status = %d, want 8", got)
			}
		})
	}
}

func TestDaysInStatusCountsFromTheLatestChange(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusInterviewing)
	h.seedStatusChange(id, StatusApplied, "2026-09-20")
	h.seedStatusChange(id, StatusInterviewing, "2026-10-04")
	if got := h.daysInStatusOf(t, id); got != 5 {
		t.Errorf("days in status = %d, want 5", got)
	}
}

func TestCardWithoutHistoryHasNoDaysInStatus(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	if card, _ := h.card(t, id); card.DaysInStatus != nil {
		t.Errorf("days in status = %d, want nil", *card.DaysInStatus)
	}
}

func TestBoardFailsWhenDaysInStatusFail(t *testing.T) {
	h := newHarness(t)
	h.acme(StatusSaved)
	h.store.fail = "ListCurrentStatusDates"
	if _, err := h.svc.Board(h.ctx); !errors.Is(err, errFake) {
		t.Errorf("err = %v, want errFake", err)
	}
}

// @scenario [features/job-status-history.feature#history-deleted-with-job] Deleting a job deletes its status history
func TestDeleteJobDeletesItsStatusHistory(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-status-history.feature#history-deleted-with-job] I have a job for "Software Engineer" at "Acme" with a status history
	id := h.movedOn(StatusSaved, "2026-10-01")
	mustMove(t, h, Move{JobID: id, Status: StatusApplied})
	other := h.movedOn(StatusSaved, "2026-10-01")

	// The frontend asks for confirmation before calling DeleteJob.
	// @when [features/job-status-history.feature] I delete the job
	if err := h.svc.DeleteJob(h.ctx, id); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-status-history.feature] its status history is removed too
	for _, c := range h.store.history {
		if c.JobID == id {
			t.Errorf("status change %+v outlived its job", c)
		}
	}
	if got := h.history(t, other); len(got) != 1 {
		t.Errorf("the other job's history = %v, want its one entry", got)
	}
}
