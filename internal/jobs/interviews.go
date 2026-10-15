package jobs

import (
	"context"
	"strings"
	"time"

	"callback/internal/store"
)

// InterviewKind is what sort of interview it is.
type InterviewKind string

const (
	InterviewPhoneScreen InterviewKind = "phone_screen"
	InterviewTechnical   InterviewKind = "technical"
	InterviewBehavioral  InterviewKind = "behavioral"
	InterviewPanel       InterviewKind = "panel"
	InterviewOnsite      InterviewKind = "onsite"
	InterviewOther       InterviewKind = "other"
)

// InterviewKinds are the kinds an interview can be, in the order they're offered.
var InterviewKinds = []InterviewKind{
	InterviewPhoneScreen, InterviewTechnical, InterviewBehavioral, InterviewPanel, InterviewOnsite, InterviewOther,
}

var interviewKindLabels = map[InterviewKind]string{
	InterviewPhoneScreen: "Phone screen",
	InterviewTechnical:   "Technical",
	InterviewBehavioral:  "Behavioral",
	InterviewPanel:       "Panel",
	InterviewOnsite:      "Onsite",
	InterviewOther:       "Other",
}

// Label is the kind as shown to the user.
func (k InterviewKind) Label() string { return interviewKindLabels[k] }

func (k InterviewKind) valid() bool { _, ok := interviewKindLabels[k]; return ok }

// timeLayout is how interview times are written: HH:MM on a 24-hour clock.
const timeLayout = "15:04"

// InterviewDetails are the fields of an interview the user fills in.
type InterviewDetails struct {
	// Date is YYYY-MM-DD.
	Date string `json:"date"`
	// Time is HH:MM on a 24-hour clock, or "" when the time isn't known.
	Time  string        `json:"time"`
	Kind  InterviewKind `json:"kind"`
	Notes string        `json:"notes"`
}

// Interview is one interview for a job.
type Interview struct {
	ID    int64         `json:"id"`
	JobID int64         `json:"jobId"`
	Date  string        `json:"date"`
	Time  string        `json:"time"`
	Kind  InterviewKind `json:"kind"`
	Notes string        `json:"notes"`
	// Past is whether the interview's date is before today. An interview today is still upcoming.
	Past bool `json:"past"`
}

// NextInterview is the soonest interview of a job that hasn't passed, as shown on its card.
type NextInterview struct {
	Date string        `json:"date"`
	Time string        `json:"time"`
	Kind InterviewKind `json:"kind"`
	// Today is whether the interview is today.
	Today bool `json:"today"`
}

// Details returns the interview's details as the user fills them in, so a change can start from
// what's saved and keep what it doesn't change.
func (i Interview) Details() InterviewDetails {
	return InterviewDetails{Date: i.Date, Time: i.Time, Kind: i.Kind, Notes: i.Notes}
}

func interviewFromRow(row store.Interview, today string) Interview {
	return Interview{
		ID:    row.ID,
		JobID: row.JobID,
		Date:  row.Date,
		Time:  row.Time,
		Kind:  InterviewKind(row.Kind),
		Notes: row.Notes,
		// YYYY-MM-DD strings sort in date order.
		Past: row.Date < today,
	}
}

// nextInterviews returns each job's next interview: its first on or after today, by date and then
// time, with untimed interviews last on their day.
// @scenario [#card-shows-next-interview] [#card-interview-today] [#card-no-upcoming-interview]
func (s *Service) nextInterviews(ctx context.Context) (map[int64]*NextInterview, error) {
	today := s.today()
	rows, err := s.store.ListUpcomingInterviews(ctx, today)
	if err != nil {
		return nil, err
	}
	next := map[int64]*NextInterview{}
	for _, row := range rows {
		// Rows come in order within each job, so the first is the soonest.
		if _, ok := next[row.JobID]; ok {
			continue
		}
		next[row.JobID] = &NextInterview{
			Date:  row.Date,
			Time:  row.Time,
			Kind:  InterviewKind(row.Kind),
			Today: row.Date == today,
		}
	}
	return next, nil
}

// Interview returns one interview.
func (s *Service) Interview(ctx context.Context, id int64) (Interview, error) {
	row, err := s.store.GetInterview(ctx, id)
	if err != nil {
		return Interview{}, notFound(err, ErrInterviewNotFound)
	}
	return interviewFromRow(row, s.today()), nil
}

// AddInterview adds an interview to a job. Nothing is saved if any detail is invalid. The date can
// be in the past, a job in any status can have interviews added, and adding one doesn't move it.
func (s *Service) AddInterview(ctx context.Context, jobID int64, d InterviewDetails) (Interview, error) {
	d, err := validateInterview(d)
	if err != nil {
		return Interview{}, err
	}
	if _, err := s.store.GetJob(ctx, jobID); err != nil {
		return Interview{}, notFound(err, ErrJobNotFound)
	}
	row, err := s.store.CreateInterview(ctx, store.CreateInterviewParams{
		JobID: jobID,
		Date:  d.Date,
		Time:  d.Time,
		Kind:  string(d.Kind),
		Notes: d.Notes,
	})
	if err != nil {
		return Interview{}, err
	}
	return interviewFromRow(row, s.today()), nil
}

// UpdateInterview replaces an interview's details. Nothing is saved if any detail is invalid.
func (s *Service) UpdateInterview(ctx context.Context, id int64, d InterviewDetails) (Interview, error) {
	d, err := validateInterview(d)
	if err != nil {
		return Interview{}, err
	}
	row, err := s.store.GetInterview(ctx, id)
	if err != nil {
		return Interview{}, notFound(err, ErrInterviewNotFound)
	}
	err = s.store.UpdateInterview(ctx, store.UpdateInterviewParams{
		ID:    id,
		Date:  d.Date,
		Time:  d.Time,
		Kind:  string(d.Kind),
		Notes: d.Notes,
	})
	if err != nil {
		return Interview{}, err
	}
	row.Date, row.Time, row.Kind, row.Notes = d.Date, d.Time, string(d.Kind), d.Notes
	return interviewFromRow(row, s.today()), nil
}

// RemoveInterview removes an interview from its job.
func (s *Service) RemoveInterview(ctx context.Context, id int64) error {
	if _, err := s.store.GetInterview(ctx, id); err != nil {
		return notFound(err, ErrInterviewNotFound)
	}
	return s.store.DeleteInterview(ctx, id)
}

// validateInterview checks the details the user entered and returns them tidied: date and time
// trimmed of surrounding space.
func validateInterview(d InterviewDetails) (InterviewDetails, error) {
	d.Date = strings.TrimSpace(d.Date)
	d.Time = strings.TrimSpace(d.Time)
	if d.Date == "" {
		return d, invalid("date is required")
	}
	if d.Kind == "" {
		return d, invalid("kind is required")
	}
	if _, err := time.Parse(dateLayout, d.Date); err != nil {
		return d, invalid("the date must be a date like 2026-10-14")
	}
	if d.Time != "" {
		// The length check refuses "9:00": times are stored with two-digit hours so they sort.
		if _, err := time.Parse(timeLayout, d.Time); err != nil || len(d.Time) != len(timeLayout) {
			return d, invalid("the time must be a time like 14:00")
		}
	}
	if !d.Kind.valid() {
		return d, invalid("choose whether the interview is a phone screen, technical, behavioral, panel, onsite, or other")
	}
	return d, nil
}
