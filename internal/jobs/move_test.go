package jobs

import (
	"errors"
	"slices"
	"testing"

	"callback/internal/store"
)

// assertPositions checks that a column's stored positions run 0, 1, 2, ... with no gaps.
func assertPositions(t *testing.T, h *harness, status Status) {
	t.Helper()
	var positions []int64
	for _, j := range h.store.jobs {
		if Status(j.Status) == status {
			positions = append(positions, j.Position)
		}
	}
	slices.Sort(positions)
	for i, p := range positions {
		if p != int64(i) {
			t.Errorf("%s positions = %v, want 0 to %d", status, positions, len(positions)-1)
			return
		}
	}
}

// @scenario [features/job-board.feature#move-card-forward] Move a card to the next status
func TestMoveCardForward(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
	id := h.acme(StatusSaved)
	h.seed(store.Job{Company: "Globex"})

	// @when [features/job-board.feature] I move the card to "Applied"
	mustMove(t, h, Move{JobID: id, Status: StatusApplied})

	// @then [features/job-board.feature] the card appears in the "Applied" column
	if got := h.column(t, StatusApplied); !slices.Equal(got, []string{"Acme"}) {
		t.Errorf("Applied = %v, want [Acme]", got)
	}
	// @and [features/job-board.feature] the card no longer appears in the "Saved" column
	if got := h.column(t, StatusSaved); !slices.Equal(got, []string{"Globex"}) {
		t.Errorf("Saved = %v, want [Globex]", got)
	}
	assertPositions(t, h, StatusSaved)
}

// @scenario [features/job-board.feature#move-card-any-status] Move a card to any other status
func TestMoveCardToAnyStatus(t *testing.T) {
	tests := []struct{ from, to string }{
		{"Saved", "Interviewing"},
		{"Applied", "Interviewing"},
		{"Interviewing", "Offer"},
		{"Offer", "Interviewing"},
		{"Closed", "Applied"},
	}
	for _, tt := range tests {
		t.Run(tt.from+" to "+tt.to, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "<from>"
			id := h.acme(statusFromLabel(t, tt.from))

			// @when [features/job-board.feature] I move the card to "<to>"
			mustMove(t, h, Move{JobID: id, Status: statusFromLabel(t, tt.to)})

			// @then [features/job-board.feature] the card appears in the "<to>" column
			if _, status := h.card(t, id); status.Label() != tt.to {
				t.Errorf("card is in %q, want %q", status.Label(), tt.to)
			}
		})
	}
}

// @scenario [features/job-board.feature#move-keeps-details] Moving a card keeps its details and documents
func TestMoveKeepsDetails(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Applied"
	id := h.seed(store.Job{Company: "Acme", Status: string(StatusApplied), DateApplied: nullString("2026-10-01"),
		// @and [features/job-board.feature] the job has a job description, a link, and an attached resume
		Description: "Build and maintain internal tools.", Link: "https://acme.example.com/jobs/123"})
	h.seedDoc(id, "resume-2026.pdf", KindResume)
	before := h.job(t, id)

	// @when [features/job-board.feature] I move the card to "Interviewing"
	mustMove(t, h, Move{JobID: id, Status: StatusInterviewing})

	// @then [features/job-board.feature] the card still shows the company "Acme" and the role "Software Engineer"
	card, _ := h.card(t, id)
	if card.Company != "Acme" || card.Role != "Software Engineer" {
		t.Errorf("card = %+v", card)
	}
	// @and [features/job-board.feature] the job description, link, and attached resume are unchanged
	after := h.job(t, id)
	if after.Description != before.Description || after.Link != before.Link || !slices.Equal(after.Documents, before.Documents) {
		t.Errorf("after the move: %+v\nbefore: %+v", after, before)
	}
}

// @scenario [features/job-board.feature#move-to-applied-sets-date] Moving a card to Applied fills in today's date when no date is set
func TestMoveToAppliedSetsDate(t *testing.T) {
	// @given [features/job-board.feature] today is 2026-10-09
	h := newHarness(t)
	// @and [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Saved" with no date applied
	id := h.acme(StatusSaved)

	// @when [features/job-board.feature] I move the card to "Applied"
	mustMove(t, h, Move{JobID: id, Status: StatusApplied})

	// @then [features/job-board.feature] the job's date applied is 2026-10-09
	if got := h.job(t, id).DateApplied; got != "2026-10-09" {
		t.Errorf("date applied = %q, want 2026-10-09", got)
	}
}

// @scenario [features/job-board.feature#move-to-applied-keeps-date] Moving a card to Applied keeps a date applied that is already set
func TestMoveToAppliedKeepsDate(t *testing.T) {
	// @given [features/job-board.feature] today is 2026-10-09
	h := newHarness(t)
	// @and [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing" with a date applied of 2026-10-01
	id := h.seed(store.Job{Company: "Acme", Status: string(StatusInterviewing), DateApplied: nullString("2026-10-01")})

	// @when [features/job-board.feature] I move the card to "Applied"
	mustMove(t, h, Move{JobID: id, Status: StatusApplied})

	// @then [features/job-board.feature] the job's date applied is still 2026-10-01
	if got := h.job(t, id).DateApplied; got != "2026-10-01" {
		t.Errorf("date applied = %q, want 2026-10-01", got)
	}
}

// @scenario [features/job-board.feature#move-to-saved-clears-date] Moving a card back to Saved clears the date applied
func TestMoveToSavedClearsDate(t *testing.T) {
	for _, from := range []string{"Applied", "Interviewing", "Offer", "Closed"} {
		t.Run(from, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "<from>" with a date applied of 2026-10-01
			id := h.seed(store.Job{Company: "Acme", Status: string(statusFromLabel(t, from)), DateApplied: nullString("2026-10-01")})

			// @when [features/job-board.feature] I move the card to "Saved"
			mustMove(t, h, Move{JobID: id, Status: StatusSaved})

			// @then [features/job-board.feature] the job has no date applied
			if got := h.store.jobs[id].DateApplied; got.Valid {
				t.Errorf("date applied = %q, want NULL", got.String)
			}
		})
	}
}

// @scenario [features/job-board.feature#reorder-saved-keeps-date] Reordering a card within Saved keeps its date applied
func TestReorderWithinSavedKeepsDate(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] the "Saved" column holds the jobs at "Globex" and "Acme" in that order
	h.seed(store.Job{Company: "Globex"})
	// @and [features/job-board.feature] the job at "Acme" has a date applied of 2026-10-01
	acme := h.seed(store.Job{Company: "Acme", DateApplied: nullString("2026-10-01")})

	// @when [features/job-board.feature] I move the "Acme" card above the "Globex" card
	mustMove(t, h, Move{JobID: acme, Status: StatusSaved, Index: 0})

	// @then [features/job-board.feature] the job's date applied is still 2026-10-01
	if got := h.job(t, acme).DateApplied; got != "2026-10-01" {
		t.Errorf("date applied = %q, want 2026-10-01", got)
	}
	if got := h.column(t, StatusSaved); !slices.Equal(got, []string{"Acme", "Globex"}) {
		t.Errorf("Saved = %v", got)
	}
}

// @scenario [features/job-board.feature#reorder-applied-no-date] Reordering a card within Applied doesn't fill in a date applied
func TestReorderWithinAppliedDoesNotFillDate(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] the "Applied" column holds the jobs at "Globex" and "Acme" in that order
	h.seed(store.Job{Company: "Globex", Status: string(StatusApplied)})
	// @and [features/job-board.feature] the job at "Acme" has no date applied
	acme := h.acme(StatusApplied)

	// @when [features/job-board.feature] I move the "Acme" card above the "Globex" card
	mustMove(t, h, Move{JobID: acme, Status: StatusApplied, Index: 0})

	// @then [features/job-board.feature#reorder-applied-no-date] the job has no date applied
	if got := h.store.jobs[acme].DateApplied; got.Valid {
		t.Errorf("date applied = %q, want NULL", got.String)
	}
}

// @scenario [features/job-board.feature#close-asks-reason] Moving a card to Closed asks for a reason
func TestCloseWithReason(t *testing.T) {
	reasons := map[string]ClosingReason{
		"Rejected":        ReasonRejected,
		"Withdrew":        ReasonWithdrew,
		"No response":     ReasonNoResponse,
		"Declined offer":  ReasonDeclinedOffer,
		"Position filled": ReasonPositionFilled,
	}
	for label, reason := range reasons {
		t.Run(label, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
			id := h.acme(StatusInterviewing)

			// @when [features/job-board.feature] I move the card to "Closed"
			// @and [features/job-board.feature] I choose the closing reason "<reason>"
			mustMove(t, h, Move{JobID: id, Status: StatusClosed, ClosingReason: reason})

			card, status := h.card(t, id)
			// @then [features/job-board.feature] the card appears in the "Closed" column
			if status != StatusClosed {
				t.Errorf("card is in %q, want Closed", status)
			}
			// @and [features/job-board.feature] the card shows the closing reason "<reason>"
			if card.ClosingReasonText != label {
				t.Errorf("closing reason = %q, want %q", card.ClosingReasonText, label)
			}
		})
	}
}

// @scenario [features/job-board.feature#close-other-reason] Write my own reason when closing with Other
func TestCloseWithOtherReason(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
	id := h.acme(StatusInterviewing)

	// @when [features/job-board.feature] I move the card to "Closed"
	// @and [features/job-board.feature] I choose the closing reason "Other" and write "Company went through layoffs"
	mustMove(t, h, Move{JobID: id, Status: StatusClosed, ClosingReason: ReasonOther, ClosingReasonOther: "  Company went through layoffs "})

	card, status := h.card(t, id)
	// @then [features/job-board.feature] the card appears in the "Closed" column
	if status != StatusClosed {
		t.Errorf("card is in %q, want Closed", status)
	}
	// @and [features/job-board.feature] the card shows the closing reason "Company went through layoffs"
	if card.ClosingReasonText != "Company went through layoffs" {
		t.Errorf("closing reason = %q", card.ClosingReasonText)
	}
}

func TestClosingNeedsAValidReason(t *testing.T) {
	tests := []struct {
		name   string
		reason ClosingReason
		other  string
	}{
		{"no reason", "", ""},
		{"unknown reason", "bored", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			id := h.acme(StatusInterviewing)

			err := h.svc.MoveJob(h.ctx, Move{JobID: id, Status: StatusClosed, ClosingReason: tt.reason, ClosingReasonOther: tt.other})

			var v *ValidationError
			if !errors.As(err, &v) {
				t.Errorf("err = %v, want a ValidationError", err)
			}
			if _, status := h.card(t, id); status != StatusInterviewing {
				t.Errorf("card moved to %q, want it to stay in Interviewing", status)
			}
		})
	}
}

// @scenario [features/job-board.feature#close-other-needs-text] Closing with Other needs a written reason
func TestCloseWithOtherNeedsText(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
	id := h.acme(StatusInterviewing)

	// @when [features/job-board.feature] I move the card to "Closed"
	// @and [features/job-board.feature] I choose the closing reason "Other" without writing a reason
	err := h.svc.MoveJob(h.ctx, Move{JobID: id, Status: StatusClosed, ClosingReason: ReasonOther, ClosingReasonOther: "  "})

	// @then [features/job-board.feature#close-other-needs-text] the card stays in the "Interviewing" column
	if _, status := h.card(t, id); status != StatusInterviewing {
		t.Errorf("card moved to %q, want it to stay in Interviewing", status)
	}
	// @and [features/job-board.feature] I am told to write why the job was closed
	wantValidation(t, err, "write why the job was closed")
}

func TestOtherTextIsDroppedForANamedReason(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusOffer)
	mustMove(t, h, Move{JobID: id, Status: StatusClosed, ClosingReason: ReasonRejected, ClosingReasonOther: "stray text"})
	if got := h.store.jobs[id].ClosingReasonOther; got.Valid {
		t.Errorf("closing reason other = %q, want NULL", got.String)
	}
}

// @scenario [features/job-board.feature#reorder-closed-keeps-reason] Reordering a card within Closed keeps its closing reason
func TestReorderWithinClosedKeepsReason(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] the "Closed" column holds the jobs at "Globex" and "Acme" in that order
	h.seed(store.Job{Company: "Globex", Status: string(StatusClosed)})
	// @and [features/job-board.feature] the job at "Acme" has the closing reason "Rejected"
	id := h.seed(store.Job{Company: "Acme", Status: string(StatusClosed), ClosingReason: nullString(string(ReasonRejected))})

	// @when [features/job-board.feature] I move the "Acme" card above the "Globex" card
	mustMove(t, h, Move{JobID: id, Status: StatusClosed, Index: 0})

	// @then [features/job-board.feature#reorder-closed-keeps-reason] the card shows the closing reason "Rejected"
	if card, _ := h.card(t, id); card.ClosingReasonText != "Rejected" {
		t.Errorf("closing reason = %q, want Rejected", card.ClosingReasonText)
	}
	if got := h.column(t, StatusClosed); !slices.Equal(got, []string{"Acme", "Globex"}) {
		t.Errorf("Closed = %v", got)
	}
}

func TestReorderWithinClosedKeepsOtherText(t *testing.T) {
	h := newHarness(t)
	h.seed(store.Job{Company: "Globex", Status: string(StatusClosed)})
	id := h.seed(store.Job{Company: "Acme", Status: string(StatusClosed), ClosingReason: nullString(string(ReasonOther)), ClosingReasonOther: nullString("Relocated")})

	mustMove(t, h, Move{JobID: id, Status: StatusClosed, Index: 0})

	if card, _ := h.card(t, id); card.ClosingReason != ReasonOther || card.ClosingReasonText != "Relocated" {
		t.Errorf("card = %+v, want the Other reason kept", card)
	}
}

// @scenario [features/job-board.feature#change-closing-reason] Change the closing reason of a closed job
func TestChangeClosingReason(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Closed" with the closing reason "No response"
	id := h.seed(store.Job{Company: "Acme", Status: string(StatusClosed), ClosingReason: nullString(string(ReasonNoResponse))})

	// @when [features/job-board.feature] I change the closing reason to "Rejected"
	if err := h.svc.SetClosingReason(h.ctx, id, ReasonRejected, ""); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-board.feature] the card shows the closing reason "Rejected"
	if card, _ := h.card(t, id); card.ClosingReasonText != "Rejected" {
		t.Errorf("closing reason = %q, want Rejected", card.ClosingReasonText)
	}
}

func TestSetClosingReasonOnlyForClosedJobs(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusInterviewing)
	if err := h.svc.SetClosingReason(h.ctx, id, ReasonRejected, ""); err == nil {
		t.Error("setting a reason on an open job succeeded")
	}
	if err := h.svc.SetClosingReason(h.ctx, 99, ReasonRejected, ""); !errors.Is(err, ErrJobNotFound) {
		t.Errorf("err = %v, want ErrJobNotFound", err)
	}
}

// @scenario [features/job-board.feature#reopen-clears-reason] Moving a card out of Closed clears the closing reason
func TestReopenClearsReason(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Closed" with the closing reason "No response"
	id := h.seed(store.Job{Company: "Acme", Status: string(StatusClosed), ClosingReason: nullString(string(ReasonNoResponse))})

	// @when [features/job-board.feature] I move the card to "Interviewing"
	mustMove(t, h, Move{JobID: id, Status: StatusInterviewing})

	// @then [features/job-board.feature] the card appears in the "Interviewing" column
	if _, status := h.card(t, id); status != StatusInterviewing {
		t.Errorf("card is in %q, want Interviewing", status)
	}
	// @and [features/job-board.feature] the job has no closing reason
	if j := h.store.jobs[id]; j.ClosingReason.Valid || j.ClosingReasonOther.Valid {
		t.Errorf("closing reason = %+v, %+v; want NULL", j.ClosingReason, j.ClosingReasonOther)
	}
}

// @scenario [features/job-board.feature#reorder-card] Reorder a card within its column
func TestReorderCard(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] the "Saved" column holds the jobs at "Acme", "Globex", and "Initech" in that order
	h.seed(store.Job{Company: "Acme"})
	h.seed(store.Job{Company: "Globex"})
	initech := h.seed(store.Job{Company: "Initech"})

	// @when [features/job-board.feature] I move the "Initech" card above the "Acme" card
	mustMove(t, h, Move{JobID: initech, Status: StatusSaved, Index: 0})

	// @then [features/job-board.feature] the "Saved" column shows "Initech", "Acme", and "Globex" in that order
	if got, want := h.column(t, StatusSaved), []string{"Initech", "Acme", "Globex"}; !slices.Equal(got, want) {
		t.Errorf("Saved = %v, want %v", got, want)
	}
	assertPositions(t, h, StatusSaved)
}

func TestReorderDownward(t *testing.T) {
	h := newHarness(t)
	acme := h.seed(store.Job{Company: "Acme"})
	h.seed(store.Job{Company: "Globex"})
	h.seed(store.Job{Company: "Initech"})

	// Index counts as if Acme had already left, so 1 lands it between Globex and Initech.
	mustMove(t, h, Move{JobID: acme, Status: StatusSaved, Index: 1})

	if got, want := h.column(t, StatusSaved), []string{"Globex", "Acme", "Initech"}; !slices.Equal(got, want) {
		t.Errorf("Saved = %v, want %v", got, want)
	}
}

// @scenario [features/job-board.feature#move-card-to-position] Place a moved card at a chosen position in another column
func TestMoveCardToPosition(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] the "Applied" column holds the jobs at "Globex" and "Initech" in that order
	h.seed(store.Job{Company: "Globex", Status: string(StatusApplied)})
	h.seed(store.Job{Company: "Initech", Status: string(StatusApplied)})
	// @and [features/job-board.feature] I have a job at "Acme" in "Saved"
	acme := h.acme(StatusSaved)

	// @when [features/job-board.feature] I move the "Acme" card into "Applied" between "Globex" and "Initech"
	mustMove(t, h, Move{JobID: acme, Status: StatusApplied, Index: 1})

	// @then [features/job-board.feature] the "Applied" column shows "Globex", "Acme", and "Initech" in that order
	if got, want := h.column(t, StatusApplied), []string{"Globex", "Acme", "Initech"}; !slices.Equal(got, want) {
		t.Errorf("Applied = %v, want %v", got, want)
	}
	assertPositions(t, h, StatusApplied)
}

func TestMoveIndexOutOfRangeLandsAtTheEnds(t *testing.T) {
	h := newHarness(t)
	h.seed(store.Job{Company: "Globex", Status: string(StatusApplied)})
	acme := h.acme(StatusSaved)
	initech := h.seed(store.Job{Company: "Initech"})

	mustMove(t, h, Move{JobID: acme, Status: StatusApplied, Index: 99})
	mustMove(t, h, Move{JobID: initech, Status: StatusApplied, Index: -5})

	if got, want := h.column(t, StatusApplied), []string{"Initech", "Globex", "Acme"}; !slices.Equal(got, want) {
		t.Errorf("Applied = %v, want %v", got, want)
	}
	assertPositions(t, h, StatusApplied)
}

func TestMoveRollsBackWhenAQueryFails(t *testing.T) {
	h := newHarness(t)
	acme := h.acme(StatusSaved)
	h.seed(store.Job{Company: "Globex", Status: string(StatusApplied)})
	h.store.fail = "UpdateJobPlacement"

	if err := h.svc.MoveJob(h.ctx, Move{JobID: acme, Status: StatusApplied}); !errors.Is(err, errFake) {
		t.Fatalf("err = %v, want errFake", err)
	}
	h.store.fail = ""
	if got := h.column(t, StatusSaved); !slices.Equal(got, []string{"Acme"}) {
		t.Errorf("Saved = %v, want [Acme]", got)
	}
	if got := h.column(t, StatusApplied); !slices.Equal(got, []string{"Globex"}) {
		t.Errorf("Applied = %v, want [Globex]", got)
	}
}

func TestMoveToUnknownStatus(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	if err := h.svc.MoveJob(h.ctx, Move{JobID: id, Status: "ghosted"}); err == nil {
		t.Error("moving to an unknown status succeeded")
	}
}
