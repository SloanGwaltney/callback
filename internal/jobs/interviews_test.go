package jobs

import (
	"errors"
	"slices"
	"testing"
	"time"

	"callback/internal/store"
)

// The harness's today is 2026-10-09.

func (h *harness) mustAddInterview(t *testing.T, jobID int64, d InterviewDetails) Interview {
	t.Helper()
	i, err := h.svc.AddInterview(h.ctx, jobID, d)
	if err != nil {
		t.Fatal(err)
	}
	return i
}

// interviewsOf lists a job's interviews as "Kind on date" or "Kind on date at time", in the order
// the job shows them.
func (h *harness) interviewsOf(t *testing.T, jobID int64) []string {
	t.Helper()
	var out []string
	for _, i := range h.job(t, jobID).Interviews {
		out = append(out, describeInterview(i))
	}
	return out
}

func describeInterview(i Interview) string {
	s := i.Kind.Label() + " on " + i.Date
	if i.Time != "" {
		s += " at " + i.Time
	}
	return s
}

// onlyInterview returns the job's one interview.
func (h *harness) onlyInterview(t *testing.T, jobID int64) Interview {
	t.Helper()
	interviews := h.job(t, jobID).Interviews
	if len(interviews) != 1 {
		t.Fatalf("job has %d interviews, want 1", len(interviews))
	}
	return interviews[0]
}

func kindFromLabel(t *testing.T, label string) InterviewKind {
	t.Helper()
	for k, l := range interviewKindLabels {
		if l == label {
			return k
		}
	}
	t.Fatalf("no interview kind labelled %q", label)
	return ""
}

// @scenario [#add-interview]
func TestAddInterviewWithAllDetails(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
	id := h.acme(StatusInterviewing)

	// @when [features/job-interviews.feature] I add an interview with:
	h.mustAddInterview(t, id, InterviewDetails{
		Date: "2026-10-14", Time: "14:00", Kind: InterviewTechnical, Notes: "Live coding with two engineers.",
	})

	// @then [features/job-interviews.feature] the job's interviews show a "Technical" interview on 2026-10-14 at 2:00 PM
	if got := h.interviewsOf(t, id); !slices.Equal(got, []string{"Technical on 2026-10-14 at 14:00"}) {
		t.Errorf("interviews = %v", got)
	}
	// @and [features/job-interviews.feature] the interview shows the notes "Live coding with two engineers."
	if got := h.onlyInterview(t, id).Notes; got != "Live coding with two engineers." {
		t.Errorf("notes = %q", got)
	}
}

// @scenario [#add-interview-minimum]
func TestAddInterviewMinimum(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
	id := h.acme(StatusInterviewing)

	// @when [features/job-interviews.feature] I add a "Phone screen" interview on 2026-10-14 with no time and no notes
	h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-14", Kind: InterviewPhoneScreen})

	// @then [features/job-interviews.feature] the job's interviews show a "Phone screen" interview on 2026-10-14
	if got := h.interviewsOf(t, id); !slices.Equal(got, []string{"Phone screen on 2026-10-14"}) {
		t.Errorf("interviews = %v", got)
	}
	if i := h.onlyInterview(t, id); i.Time != "" || i.Notes != "" {
		t.Errorf("interview = %+v, want no time and no notes", i)
	}
}

// @scenario [#interview-kinds]
func TestInterviewKinds(t *testing.T) {
	for _, label := range []string{"Phone screen", "Technical", "Behavioral", "Panel", "Onsite", "Other"} {
		t.Run(label, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
			id := h.acme(StatusInterviewing)

			// @when [features/job-interviews.feature] I add a "<kind>" interview on 2026-10-14
			h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-14", Kind: kindFromLabel(t, label)})

			// @then [features/job-interviews.feature] the job's interviews show a "<kind>" interview on 2026-10-14
			if got := h.interviewsOf(t, id); !slices.Equal(got, []string{label + " on 2026-10-14"}) {
				t.Errorf("interviews = %v", got)
			}
		})
	}
}

// @scenario [#interview-missing-required]
func TestInterviewMissingRequired(t *testing.T) {
	for _, tt := range []struct {
		date    string
		kind    InterviewKind
		missing string
	}{
		{"", InterviewTechnical, "date"},
		{"2026-10-14", "", "kind"},
	} {
		t.Run(tt.missing, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
			id := h.acme(StatusInterviewing)

			// @when [features/job-interviews.feature] I try to add an interview with date "<date>" and kind "<kind>"
			_, err := h.svc.AddInterview(h.ctx, id, InterviewDetails{Date: tt.date, Kind: tt.kind})

			// @then [features/job-interviews.feature] the interview is not added
			if got := h.job(t, id).Interviews; len(got) != 0 {
				t.Errorf("interviews = %v, want none", got)
			}
			// @and [features/job-interviews.feature] I am told that the <missing> is required
			wantValidation(t, err, tt.missing+" is required")
		})
	}
}

// @scenario [#interview-past-date]
func TestAddPastInterview(t *testing.T) {
	// @given [features/job-interviews.feature] today is 2026-10-09
	h := newHarness(t)
	// @and [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
	id := h.acme(StatusInterviewing)

	// @when [features/job-interviews.feature] I add a "Phone screen" interview on 2026-10-02
	h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-02", Kind: InterviewPhoneScreen})

	// @then [features/job-interviews.feature] the job's interviews show a "Phone screen" interview on 2026-10-02
	if got := h.interviewsOf(t, id); !slices.Equal(got, []string{"Phone screen on 2026-10-02"}) {
		t.Errorf("interviews = %v", got)
	}
}

// @scenario [#interview-any-status]
func TestAddInterviewInAnyStatus(t *testing.T) {
	for _, status := range Statuses {
		t.Run(status.Label(), func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "<status>"
			id := h.acme(status)

			// @when [features/job-interviews.feature] I add a "Phone screen" interview on 2026-10-14
			h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-14", Kind: InterviewPhoneScreen})

			// @then [features/job-interviews.feature] the job's interviews show a "Phone screen" interview on 2026-10-14
			if got := h.interviewsOf(t, id); !slices.Equal(got, []string{"Phone screen on 2026-10-14"}) {
				t.Errorf("interviews = %v", got)
			}
			// @and [features/job-interviews.feature] the card stays in the "<status>" column
			if _, in := h.card(t, id); in != status {
				t.Errorf("card is in %q, want %q", in, status)
			}
		})
	}
}

// @scenario [#view-interviews]
func TestViewInterviews(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Phone screen" interview on 2026-10-02
	id := h.acme(StatusInterviewing)
	h.seedInterview(id, store.Interview{Date: "2026-10-02", Kind: string(InterviewPhoneScreen)})
	// @and [features/job-interviews.feature] a "Technical" interview on 2026-10-14
	h.seedInterview(id, store.Interview{Date: "2026-10-14", Kind: string(InterviewTechnical)})
	other := h.seed(store.Job{Company: "Globex"})
	h.seedInterview(other, store.Interview{Date: "2026-10-03", Kind: string(InterviewPanel)})

	// @when [features/job-interviews.feature] I open the card
	got := h.interviewsOf(t, id)

	// @then [features/job-interviews.feature] I see both interviews
	if want := []string{"Phone screen on 2026-10-02", "Technical on 2026-10-14"}; !slices.Equal(got, want) {
		t.Errorf("interviews = %v, want %v", got, want)
	}
}

// @scenario [#interviews-in-date-order]
func TestInterviewsInDateOrder(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusInterviewing)
	// @and [features/job-interviews.feature] I added an "Onsite" interview on 2026-10-20
	h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-20", Kind: InterviewOnsite})
	// @and [features/job-interviews.feature] I then added a "Phone screen" interview on 2026-10-02
	h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-02", Kind: InterviewPhoneScreen})
	// @and [features/job-interviews.feature] I then added a "Technical" interview on 2026-10-14
	h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-14", Kind: InterviewTechnical})

	// @when [features/job-interviews.feature] I open the card
	var kinds []string
	for _, i := range h.job(t, id).Interviews {
		kinds = append(kinds, i.Kind.Label())
	}

	// @then [features/job-interviews.feature] the interviews are listed as "Phone screen", "Technical", and "Onsite" in that order
	if want := []string{"Phone screen", "Technical", "Onsite"}; !slices.Equal(kinds, want) {
		t.Errorf("interviews = %v, want %v", kinds, want)
	}
}

// @scenario [#past-interviews-marked]
func TestPastInterviewsMarked(t *testing.T) {
	// @given [features/job-interviews.feature] today is 2026-10-09
	h := newHarness(t)
	// @and [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Phone screen" interview on 2026-10-02
	id := h.acme(StatusInterviewing)
	h.seedInterview(id, store.Interview{Date: "2026-10-02", Kind: string(InterviewPhoneScreen)})
	// @and [features/job-interviews.feature] a "Technical" interview on 2026-10-14
	h.seedInterview(id, store.Interview{Date: "2026-10-14", Kind: string(InterviewTechnical)})
	h.seedInterview(id, store.Interview{Date: "2026-10-09", Kind: string(InterviewPanel)})

	// @when [features/job-interviews.feature] I open the card
	past := map[InterviewKind]bool{}
	for _, i := range h.job(t, id).Interviews {
		past[i.Kind] = i.Past
	}

	// @then [features/job-interviews.feature] the "Phone screen" interview is shown as past
	if !past[InterviewPhoneScreen] {
		t.Error("the phone screen on 2026-10-02 isn't past")
	}
	// @and [features/job-interviews.feature] the "Technical" interview is shown as upcoming
	if past[InterviewTechnical] {
		t.Error("the technical interview on 2026-10-14 is past")
	}
	if past[InterviewPanel] {
		t.Error("an interview today is past, want upcoming")
	}
}

// @scenario [#no-interviews]
func TestNoInterviews(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with no interviews
	id := h.acme(StatusApplied)

	// @when [features/job-interviews.feature] I open the card
	got := h.job(t, id).Interviews

	// An empty list, not nil, so the frontend gets [] and says the job has none.
	if got == nil || len(got) != 0 {
		t.Errorf("interviews = %#v, want an empty list", got)
	}
}

// @scenario [#card-shows-next-interview]
func TestCardShowsNextInterview(t *testing.T) {
	// @given [features/job-interviews.feature] today is 2026-10-09
	h := newHarness(t)
	// @and [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Phone screen" interview on 2026-10-02
	id := h.acme(StatusInterviewing)
	h.seedInterview(id, store.Interview{Date: "2026-10-02", Kind: string(InterviewPhoneScreen)})
	// @and [features/job-interviews.feature] an "Onsite" interview on 2026-10-20
	h.seedInterview(id, store.Interview{Date: "2026-10-20", Kind: string(InterviewOnsite)})
	// @and [features/job-interviews.feature] a "Technical" interview on 2026-10-14
	h.seedInterview(id, store.Interview{Date: "2026-10-14", Kind: string(InterviewTechnical)})
	other := h.seed(store.Job{Company: "Globex", Status: string(StatusInterviewing)})
	h.seedInterview(other, store.Interview{Date: "2026-10-10", Kind: string(InterviewPanel)})

	// @when [features/job-interviews.feature] I open the job search board
	card, _ := h.card(t, id)

	// @then [features/job-interviews.feature] the card shows a "Technical" interview on 2026-10-14
	want := NextInterview{Date: "2026-10-14", Kind: InterviewTechnical}
	if card.NextInterview == nil || *card.NextInterview != want {
		t.Errorf("next interview = %+v, want %+v", card.NextInterview, want)
	}
}

// @scenario [#card-interview-today]
func TestCardShowsInterviewToday(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interviews.feature] today is 2026-10-14
	h.setToday(2026, time.October, 14)
	// @and [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
	id := h.acme(StatusInterviewing)
	h.seedInterview(id, store.Interview{Date: "2026-10-14", Kind: string(InterviewTechnical)})

	// @when [features/job-interviews.feature] I open the job search board
	card, _ := h.card(t, id)

	// @then [features/job-interviews.feature] the card shows a "Technical" interview today
	want := NextInterview{Date: "2026-10-14", Kind: InterviewTechnical, Today: true}
	if card.NextInterview == nil || *card.NextInterview != want {
		t.Errorf("next interview = %+v, want %+v", card.NextInterview, want)
	}
}

// @scenario [#card-no-upcoming-interview]
func TestCardNoUpcomingInterview(t *testing.T) {
	// @given [features/job-interviews.feature] today is 2026-10-09
	h := newHarness(t)
	// @and [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with only a "Phone screen" interview on 2026-10-02
	id := h.acme(StatusInterviewing)
	h.seedInterview(id, store.Interview{Date: "2026-10-02", Kind: string(InterviewPhoneScreen)})

	// @when [features/job-interviews.feature] I open the job search board
	card, _ := h.card(t, id)

	// @then [features/job-interviews.feature] the card doesn't show an interview
	if card.NextInterview != nil {
		t.Errorf("next interview = %+v, want none", card.NextInterview)
	}
}

func TestCardNextInterviewPrefersTimedOnTheSameDay(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusInterviewing)
	h.seedInterview(id, store.Interview{Date: "2026-10-14", Kind: string(InterviewPanel)})
	h.seedInterview(id, store.Interview{Date: "2026-10-14", Time: "15:00", Kind: string(InterviewBehavioral)})
	h.seedInterview(id, store.Interview{Date: "2026-10-14", Time: "10:00", Kind: string(InterviewTechnical)})

	card, _ := h.card(t, id)
	want := NextInterview{Date: "2026-10-14", Time: "10:00", Kind: InterviewTechnical}
	if card.NextInterview == nil || *card.NextInterview != want {
		t.Errorf("next interview = %+v, want %+v", card.NextInterview, want)
	}
}

func TestBoardFailsWhenInterviewsCantBeListed(t *testing.T) {
	h := newHarness(t)
	h.acme(StatusSaved)
	h.store.fail = "ListUpcomingInterviews"
	if _, err := h.svc.Board(h.ctx); !errors.Is(err, errFake) {
		t.Errorf("err = %v, want errFake", err)
	}
}

// @scenario [#edit-interview]
func TestEditInterview(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
	id := h.acme(StatusInterviewing)
	interview := h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-14", Kind: InterviewTechnical})

	// @when [features/job-interviews.feature] I change the interview's date to 2026-10-16
	d := InterviewDetails{Date: "2026-10-16", Kind: interview.Kind}
	// @and [features/job-interviews.feature] I write in its notes "Rescheduled by the recruiter."
	d.Notes = "Rescheduled by the recruiter."
	updated, err := h.svc.UpdateInterview(h.ctx, interview.ID, d)
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-interviews.feature] the job's interviews show a "Technical" interview on 2026-10-16
	if got := h.interviewsOf(t, id); !slices.Equal(got, []string{"Technical on 2026-10-16"}) {
		t.Errorf("interviews = %v", got)
	}
	// @and [features/job-interviews.feature] the interview shows the notes "Rescheduled by the recruiter."
	if got := h.onlyInterview(t, id).Notes; got != "Rescheduled by the recruiter." {
		t.Errorf("notes = %q", got)
	}
	if updated != h.onlyInterview(t, id) {
		t.Errorf("returned %+v, saved %+v", updated, h.onlyInterview(t, id))
	}
}

func TestEditInterviewRejectsInvalidDetails(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusInterviewing)
	interview := h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-14", Kind: InterviewTechnical, Notes: "Bring a laptop."})

	_, err := h.svc.UpdateInterview(h.ctx, interview.ID, InterviewDetails{Kind: InterviewTechnical})
	wantValidation(t, err, "date is required")
	if got := h.onlyInterview(t, id); got != interview {
		t.Errorf("interview = %+v, want it unchanged: %+v", got, interview)
	}
}

func TestEditMissingInterview(t *testing.T) {
	h := newHarness(t)
	_, err := h.svc.UpdateInterview(h.ctx, 99, InterviewDetails{Date: "2026-10-14", Kind: InterviewTechnical})
	if !errors.Is(err, ErrInterviewNotFound) {
		t.Errorf("err = %v, want ErrInterviewNotFound", err)
	}
}

// @scenario [#remove-interview]
func TestRemoveInterview(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
	id := h.acme(StatusInterviewing)
	interview := h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-14", Kind: InterviewTechnical})
	keep := h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-20", Kind: InterviewOnsite})

	// The frontend asks for confirmation before calling RemoveInterview.
	// @when [features/job-interviews.feature] I remove the interview
	if err := h.svc.RemoveInterview(h.ctx, interview.ID); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-interviews.feature] the job no longer shows the "Technical" interview
	if got := h.job(t, id).Interviews; len(got) != 1 || got[0].ID != keep.ID {
		t.Errorf("interviews = %+v, want only the onsite", got)
	}
}

func TestRemoveMissingInterview(t *testing.T) {
	h := newHarness(t)
	if err := h.svc.RemoveInterview(h.ctx, 99); !errors.Is(err, ErrInterviewNotFound) {
		t.Errorf("err = %v, want ErrInterviewNotFound", err)
	}
}

// @scenario [#move-keeps-interviews]
func TestMoveKeepsInterviews(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing" with a "Technical" interview on 2026-10-14
	id := h.acme(StatusInterviewing)
	h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-14", Kind: InterviewTechnical})

	// @when [features/job-interviews.feature] I move the card to "Offer"
	mustMove(t, h, Move{JobID: id, Status: StatusOffer})

	// @then [features/job-interviews.feature] the job still shows the "Technical" interview on 2026-10-14
	if got := h.interviewsOf(t, id); !slices.Equal(got, []string{"Technical on 2026-10-14"}) {
		t.Errorf("interviews = %v", got)
	}
}

// @scenario [#interviews-deleted-with-job]
func TestDeleteJobDeletesItsInterviews(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
	id := h.acme(StatusInterviewing)
	h.mustAddInterview(t, id, InterviewDetails{Date: "2026-10-14", Kind: InterviewTechnical})
	other := h.seed(store.Job{Company: "Globex"})
	h.mustAddInterview(t, other, InterviewDetails{Date: "2026-10-14", Kind: InterviewTechnical})

	// The frontend asks for confirmation before calling DeleteJob.
	// @when [features/job-interviews.feature] I delete the job
	if err := h.svc.DeleteJob(h.ctx, id); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-interviews.feature] its interviews are removed too
	for _, i := range h.store.interviews {
		if i.JobID == id {
			t.Errorf("interview %+v outlived its job", i)
		}
	}
	if len(h.store.interviews) != 1 {
		t.Errorf("%d interviews left, want Globex's 1", len(h.store.interviews))
	}
}

func TestAddInterviewToMissingJob(t *testing.T) {
	h := newHarness(t)
	_, err := h.svc.AddInterview(h.ctx, 99, InterviewDetails{Date: "2026-10-14", Kind: InterviewTechnical})
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("err = %v, want ErrJobNotFound", err)
	}
}

func TestValidateInterview(t *testing.T) {
	for _, tt := range []struct {
		name string
		d    InterviewDetails
		want string
	}{
		{"blank date", InterviewDetails{Date: "  ", Kind: InterviewTechnical}, "date is required"},
		{"not a date", InterviewDetails{Date: "Oct 14", Kind: InterviewTechnical}, "the date must be a date like 2026-10-14"},
		{"impossible date", InterviewDetails{Date: "2026-02-30", Kind: InterviewTechnical}, "the date must be a date like 2026-10-14"},
		{"not a time", InterviewDetails{Date: "2026-10-14", Time: "2pm", Kind: InterviewTechnical}, "the time must be a time like 14:00"},
		{"one-digit hour", InterviewDetails{Date: "2026-10-14", Time: "9:00", Kind: InterviewTechnical}, "the time must be a time like 14:00"},
		{"hour out of range", InterviewDetails{Date: "2026-10-14", Time: "25:00", Kind: InterviewTechnical}, "the time must be a time like 14:00"},
		{"unknown kind", InterviewDetails{Date: "2026-10-14", Kind: "coffee_chat"}, "choose whether the interview is a phone screen, technical, behavioral, panel, onsite, or other"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			id := h.acme(StatusInterviewing)
			_, err := h.svc.AddInterview(h.ctx, id, tt.d)
			wantValidation(t, err, tt.want)
			if got := h.job(t, id).Interviews; len(got) != 0 {
				t.Errorf("interviews = %v, want none", got)
			}
		})
	}
}

func TestAddInterviewTrimsDateAndTime(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusInterviewing)
	i := h.mustAddInterview(t, id, InterviewDetails{Date: " 2026-10-14 ", Time: " 09:30 ", Kind: InterviewTechnical})
	if i.Date != "2026-10-14" || i.Time != "09:30" {
		t.Errorf("interview = %+v", i)
	}
}

func TestInterview(t *testing.T) {
	h := newHarness(t)
	h.setToday(2026, time.October, 14)
	id := h.acme(StatusInterviewing)
	past := h.seedInterview(id, store.Interview{Date: "2026-10-13", Time: "14:00", Kind: string(InterviewTechnical), Notes: "Bring a laptop."})

	got, err := h.svc.Interview(h.ctx, past)
	if err != nil {
		t.Fatal(err)
	}
	want := Interview{ID: past, JobID: id, Date: "2026-10-13", Time: "14:00", Kind: InterviewTechnical, Notes: "Bring a laptop.", Past: true}
	if got != want {
		t.Errorf("Interview = %+v, want %+v", got, want)
	}
	if _, err := h.svc.Interview(h.ctx, 999); !errors.Is(err, ErrInterviewNotFound) {
		t.Errorf("err = %v, want ErrInterviewNotFound", err)
	}
}

func TestInterviewKindsListsEveryOne(t *testing.T) {
	if len(InterviewKinds) != len(interviewKindLabels) {
		t.Errorf("InterviewKinds has %d, labels have %d", len(InterviewKinds), len(interviewKindLabels))
	}
	for _, k := range InterviewKinds {
		if !k.valid() {
			t.Errorf("%q isn't an interview kind", k)
		}
	}
}
