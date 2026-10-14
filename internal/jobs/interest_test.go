package jobs

import (
	"database/sql"
	"fmt"
	"slices"
	"testing"

	"callback/internal/store"
)

// stars returns a pointer to an interest rating, for Details and Job literals.
func stars(n int) *int { return &n }

// rated seeds Acme with an interest rating in a status column.
func (h *harness) rated(status Status, n int) int64 {
	return h.seed(store.Job{Company: "Acme", Status: string(status), Interest: sql.NullInt64{Int64: int64(n), Valid: true}})
}

// rate saves a job's details as they are, with the interest rating changed to interest.
func (h *harness) rate(t *testing.T, id int64, interest *int) error {
	t.Helper()
	j := h.job(t, id)
	_, err := h.svc.UpdateJob(h.ctx, id, Details{
		Company: j.Company, Role: j.Role, Description: j.Description, Link: j.Link, Salary: j.Salary,
		Location: j.Location, Notes: j.Notes, DateApplied: j.DateApplied, Interest: interest,
	})
	return err
}

func (h *harness) mustRate(t *testing.T, id int64, interest *int) {
	t.Helper()
	if err := h.rate(t, id, interest); err != nil {
		t.Fatal(err)
	}
}

// interestOf is a job's interest rating as text, for comparing and printing.
func interestOf(interest *int) string {
	if interest == nil {
		return "unrated"
	}
	return fmt.Sprintf("%d stars", *interest)
}

// @scenario [#rate-interest]
func TestRateInterest(t *testing.T) {
	for n := MinInterest; n <= MaxInterest; n++ {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with no interest rating
			id := h.acme(StatusSaved)

			// @when [features/job-interest.feature] I answer "How interested are you in this job?" with <stars> stars
			h.mustRate(t, id, stars(n))

			// @then [features/job-interest.feature] the job's interest rating is <stars> stars
			if got := h.job(t, id).Interest; interestOf(got) != interestOf(stars(n)) {
				t.Errorf("interest = %s, want %d stars", interestOf(got), n)
			}
		})
	}
}

// @scenario [#rate-interest-when-adding]
func TestRateInterestWhenAdding(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interest.feature] I open the job search board
	h.board(t)

	// @when [features/job-interest.feature] I add a job with company "Acme" and role "Software Engineer"
	details := Details{Company: "Acme", Role: "Software Engineer"}
	// @and [features/job-interest.feature#rate-interest-when-adding] I answer "How interested are you in this job?" with 4 stars
	details.Interest = stars(4)
	added, err := h.svc.AddJob(h.ctx, details, "")
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-interest.feature] a card for "Software Engineer" at "Acme" appears with an interest rating of 4 stars
	card, _ := h.card(t, added.ID)
	if card.Company != "Acme" || card.Role != "Software Engineer" || interestOf(card.Interest) != "4 stars" {
		t.Errorf("card = %+v with %s, want Acme Software Engineer with 4 stars", card, interestOf(card.Interest))
	}
}

// @scenario [#unrated-by-default]
func TestUnratedByDefault(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interest.feature] I open the job search board
	h.board(t)

	// @when [features/job-interest.feature] I add a job with company "Acme" and role "Software Engineer" and nothing else
	added, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Software Engineer"}, "")
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-interest.feature] the job has no interest rating
	if got := h.job(t, added.ID).Interest; got != nil {
		t.Errorf("interest = %s, want unrated", interestOf(got))
	}
}

// @scenario [#interest-rating-range]
func TestInterestRatingRange(t *testing.T) {
	for _, n := range []int{0, 6} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with an interest rating of 3 stars
			id := h.rated(StatusSaved, 3)

			// @when [features/job-interest.feature] I try to set the interest rating to <stars> stars
			err := h.rate(t, id, stars(n))

			// @then [features/job-interest.feature] the interest rating is not saved
			if err == nil {
				t.Fatalf("rating %d stars was saved", n)
			}
			// @and [features/job-interest.feature] the job's interest rating is still 3 stars
			if got := h.job(t, id).Interest; interestOf(got) != "3 stars" {
				t.Errorf("interest = %s, want 3 stars", interestOf(got))
			}
			// @and [features/job-interest.feature] I am told the rating must be from 1 to 5 stars
			wantValidation(t, err, "the interest rating must be from 1 to 5 stars")
		})
	}
}

func TestAddJobRejectsAnInterestOutOfRange(t *testing.T) {
	h := newHarness(t)
	_, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Software Engineer", Interest: stars(-1)}, "")
	wantValidation(t, err, "the interest rating must be from 1 to 5 stars")
	if got := h.column(t, StatusSaved); len(got) != 0 {
		t.Errorf("Saved = %v, want nothing added", got)
	}
}

// @scenario [#interest-any-status]
func TestInterestInAnyStatus(t *testing.T) {
	for _, status := range Statuses {
		t.Run(status.Label(), func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" in "<status>"
			id := h.acme(statusFromLabel(t, status.Label()))

			// @when [features/job-interest.feature#interest-any-status] I answer "How interested are you in this job?" with 4 stars
			h.mustRate(t, id, stars(4))

			// @then [features/job-interest.feature] the job's interest rating is 4 stars
			if got := h.job(t, id).Interest; interestOf(got) != "4 stars" {
				t.Errorf("interest = %s, want 4 stars", interestOf(got))
			}
		})
	}
}

// @scenario [#change-interest]
func TestChangeInterest(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with an interest rating of 2 stars
	id := h.rated(StatusSaved, 2)

	// @when [features/job-interest.feature] I change the interest rating to 5 stars
	h.mustRate(t, id, stars(5))

	// @then [features/job-interest.feature] the job's interest rating is 5 stars
	if got := h.job(t, id).Interest; interestOf(got) != "5 stars" {
		t.Errorf("interest = %s, want 5 stars", interestOf(got))
	}
}

// @scenario [#clear-interest]
func TestClearInterest(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with an interest rating of 3 stars
	id := h.rated(StatusSaved, 3)

	// @when [features/job-interest.feature] I clear the interest rating
	h.mustRate(t, id, nil)

	// @then [features/job-interest.feature] the job has no interest rating
	if got := h.job(t, id).Interest; got != nil {
		t.Errorf("interest = %s, want unrated", interestOf(got))
	}
}

// @scenario [#card-shows-interest]
func TestCardShowsInterest(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with an interest rating of 4 stars
	id := h.rated(StatusSaved, 4)

	// @when [features/job-interest.feature] I open the job search board
	card, _ := h.card(t, id)

	// @then [features/job-interest.feature] the card shows 4 of 5 stars
	if interestOf(card.Interest) != "4 stars" {
		t.Errorf("card interest = %s, want 4 stars", interestOf(card.Interest))
	}
}

// @scenario [#card-unrated]
func TestCardUnrated(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with no interest rating
	id := h.acme(StatusSaved)

	// @when [features/job-interest.feature] I open the job search board
	card, _ := h.card(t, id)

	// @then [features/job-interest.feature] the card doesn't show an interest rating
	if card.Interest != nil {
		t.Errorf("card interest = %s, want unrated", interestOf(card.Interest))
	}
}

// @scenario [#view-interest]
func TestViewInterest(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with an interest rating of 4 stars
	id := h.rated(StatusSaved, 4)

	// @when [features/job-interest.feature] I open the card
	got := h.job(t, id)

	// @then [features/job-interest.feature] I see "How interested are you in this job?" answered with 4 stars
	if interestOf(got.Interest) != "4 stars" {
		t.Errorf("interest = %s, want 4 stars", interestOf(got.Interest))
	}
}

// @scenario [#rating-doesnt-reorder]
func TestRatingDoesntReorder(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interest.feature] the "Saved" column holds the jobs at "Acme" and "Globex" in that order
	h.acme(StatusSaved)
	globex := h.seed(store.Job{Company: "Globex"})

	// @when [features/job-interest.feature] I give the job at "Globex" an interest rating of 5 stars
	h.mustRate(t, globex, stars(5))

	// @then [features/job-interest.feature] the "Saved" column still shows "Acme" and "Globex" in that order
	if got, want := h.column(t, StatusSaved), []string{"Acme", "Globex"}; !slices.Equal(got, want) {
		t.Errorf("Saved = %v, want %v", got, want)
	}
}

// @scenario [#move-keeps-interest]
func TestMoveKeepsInterest(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" in "Applied" with an interest rating of 4 stars
	id := h.rated(StatusApplied, 4)

	// @when [features/job-interest.feature] I move the card to "Interviewing"
	mustMove(t, h, Move{JobID: id, Status: StatusInterviewing})

	// @then [features/job-interest.feature] the job's interest rating is still 4 stars
	if got := h.job(t, id).Interest; interestOf(got) != "4 stars" {
		t.Errorf("interest = %s, want 4 stars", interestOf(got))
	}
}

func TestEditingOtherDetailsKeepsInterest(t *testing.T) {
	h := newHarness(t)
	id := h.rated(StatusSaved, 4)
	j := h.job(t, id)
	_, err := h.svc.UpdateJob(h.ctx, id, Details{Company: j.Company, Role: j.Role, Notes: "Call back Friday.", Interest: j.Interest})
	if err != nil {
		t.Fatal(err)
	}
	if got := h.job(t, id).Interest; interestOf(got) != "4 stars" {
		t.Errorf("interest = %s, want 4 stars", interestOf(got))
	}
}
