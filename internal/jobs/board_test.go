package jobs

import (
	"slices"
	"testing"

	"callback/internal/store"
)

// @scenario [features/job-board.feature#board-status-columns] Board shows the status columns in order
func TestBoardShowsStatusColumnsInOrder(t *testing.T) {
	h := newHarness(t)

	// @given [features/job-board.feature] I open the job search board
	board := h.board(t)

	// @then [features/job-board.feature] I see the columns "Saved", "Applied", "Interviewing", "Offer", and "Closed" in that order
	var labels []string
	for _, c := range board.Columns {
		labels = append(labels, c.Label)
	}
	want := []string{"Saved", "Applied", "Interviewing", "Offer", "Closed"}
	if !slices.Equal(labels, want) {
		t.Errorf("columns = %v, want %v", labels, want)
	}
}

// @scenario [features/job-board.feature#empty-board] Empty board when no jobs have been added
func TestEmptyBoard(t *testing.T) {
	// @given [features/job-board.feature] I have not added any jobs
	h := newHarness(t)

	// @when [features/job-board.feature] I open the job search board
	board := h.board(t)

	// @then [features/job-board.feature] every status column is empty
	for _, c := range board.Columns {
		// An empty slice, not nil, so the frontend gets [] rather than null.
		if c.Cards == nil || len(c.Cards) != 0 || c.Count != 0 {
			t.Errorf("%s column = %d cards (count %d, nil %v), want empty", c.Label, len(c.Cards), c.Count, c.Cards == nil)
		}
	}
	// "And I am prompted to add my first job" is the frontend's to show.
}

// @scenario [features/job-board.feature#column-card-count] Each column shows how many cards it holds
func TestColumnCardCount(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] I have 3 jobs in "Saved" and 1 job in "Applied"
	for _, company := range []string{"Acme", "Globex", "Initech"} {
		h.seed(store.Job{Company: company})
	}
	h.seed(store.Job{Company: "Hooli", Status: string(StatusApplied)})

	// @when [features/job-board.feature] I open the job search board
	board := h.board(t)

	counts := map[Status]int{}
	for _, c := range board.Columns {
		counts[c.Status] = c.Count
	}
	// @then [features/job-board.feature] the "Saved" column shows a count of 3
	if counts[StatusSaved] != 3 {
		t.Errorf("Saved count = %d, want 3", counts[StatusSaved])
	}
	// @and [features/job-board.feature] the "Applied" column shows a count of 1
	if counts[StatusApplied] != 1 {
		t.Errorf("Applied count = %d, want 1", counts[StatusApplied])
	}
	// @and [features/job-board.feature] the "Interviewing" column shows a count of 0
	if counts[StatusInterviewing] != 0 {
		t.Errorf("Interviewing count = %d, want 0", counts[StatusInterviewing])
	}
}

// @scenario [features/job-documents.feature#card-shows-document-count] The card shows that documents are attached
func TestCardShowsDocumentCount(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme" with 2 attached documents
	id := h.acme(StatusSaved)
	h.seedDoc(id, "resume-2026.pdf", KindResume)
	h.seedDoc(id, "acme-cover.docx", KindCoverLetter)
	other := h.seed(store.Job{Company: "Globex"})

	// @when [features/job-documents.feature] I look at the card on the board
	card, _ := h.card(t, id)

	// @then [features/job-documents.feature] the card shows that it has 2 documents
	if card.DocumentCount != 2 {
		t.Errorf("document count = %d, want 2", card.DocumentCount)
	}
	if card, _ := h.card(t, other); card.DocumentCount != 0 {
		t.Errorf("job with no documents shows %d", card.DocumentCount)
	}
}

func TestBoardCardCarriesClosingReasonText(t *testing.T) {
	h := newHarness(t)
	rejected := h.seed(store.Job{Company: "Acme", Status: string(StatusClosed), ClosingReason: nullString(string(ReasonRejected))})
	other := h.seed(store.Job{Company: "Globex", Status: string(StatusClosed), ClosingReason: nullString(string(ReasonOther)), ClosingReasonOther: nullString("Relocated")})

	if card, _ := h.card(t, rejected); card.ClosingReason != ReasonRejected || card.ClosingReasonText != "Rejected" {
		t.Errorf("rejected card = %+v", card)
	}
	if card, _ := h.card(t, other); card.ClosingReason != ReasonOther || card.ClosingReasonText != "Relocated" {
		t.Errorf("other card = %+v", card)
	}
}
