package db

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"

	"callback/internal/store"
)

// Storage-tier tests for the interviews table and its queries.

func createInterview(t *testing.T, q store.Querier, jobID int64, date, tm, kind string) store.Interview {
	t.Helper()
	i, err := q.CreateInterview(context.Background(), store.CreateInterviewParams{JobID: jobID, Date: date, Time: tm, Kind: kind})
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func interviewKinds(interviews []store.Interview) []string {
	var kinds []string
	for _, i := range interviews {
		kinds = append(kinds, i.Kind)
	}
	return kinds
}

func TestSchemaInterviews(t *testing.T) {
	conn := openSchema(t)
	// Foreign keys are set per connection, so keep every statement on one.
	conn.SetMaxOpenConns(1)

	res := mustExec(t, conn, `INSERT INTO jobs (company, role, position) VALUES ('Acme', 'Software Engineer', 0)`)
	jobID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"phone_screen", "technical", "behavioral", "panel", "onsite", "other"} {
		mustExec(t, conn, `INSERT INTO interviews (job_id, date, kind) VALUES (?, '2026-10-14', ?)`, jobID, kind)
	}
	mustExec(t, conn, `INSERT INTO interviews (job_id, date, time, kind) VALUES (?, '2026-10-14', '23:59', 'panel')`, jobID)

	var tm, notes string
	if err := conn.QueryRow(`SELECT time, notes FROM interviews LIMIT 1`).Scan(&tm, &notes); err != nil {
		t.Fatal(err)
	}
	if tm != "" || notes != "" {
		t.Errorf("optional fields default to %q %q, want ''", tm, notes)
	}

	for _, tt := range []struct{ name, query string }{
		{"missing date", `INSERT INTO interviews (job_id, kind) VALUES (?, 'technical')`},
		{"malformed date", `INSERT INTO interviews (job_id, date, kind) VALUES (?, 'Oct 14', 'technical')`},
		{"malformed time", `INSERT INTO interviews (job_id, date, time, kind) VALUES (?, '2026-10-14', '2pm', 'technical')`},
		{"one-digit hour", `INSERT INTO interviews (job_id, date, time, kind) VALUES (?, '2026-10-14', '9:00', 'technical')`},
		{"missing kind", `INSERT INTO interviews (job_id, date) VALUES (?, '2026-10-14')`},
		{"unknown kind", `INSERT INTO interviews (job_id, date, kind) VALUES (?, '2026-10-14', 'coffee_chat')`},
	} {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			if _, err := conn.Exec(tt.query, jobID); err == nil {
				t.Error("insert succeeded, want a constraint error")
			}
		})
	}

	t.Run("rejects a missing job", func(t *testing.T) {
		if _, err := conn.Exec(`INSERT INTO interviews (job_id, date, kind) VALUES (999, '2026-10-14', 'technical')`); err == nil {
			t.Error("insert succeeded, want a foreign key error")
		}
	})
}

func TestInterviewQueries(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	acme := createJob(t, s, "Acme", "interviewing", 0)

	tech, err := s.CreateInterview(ctx, store.CreateInterviewParams{
		JobID: acme, Date: "2026-10-14", Time: "14:00", Kind: "technical", Notes: "Live coding with two engineers.",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = s.UpdateInterview(ctx, store.UpdateInterviewParams{
		ID: tech.ID, Date: "2026-10-16", Time: "", Kind: "technical", Notes: "Rescheduled by the recruiter.",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetInterview(ctx, tech.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := store.Interview{ID: tech.ID, JobID: acme, Date: "2026-10-16", Kind: "technical", Notes: "Rescheduled by the recruiter."}
	if got != want {
		t.Errorf("interview = %+v\nwant %+v", got, want)
	}

	if err := s.DeleteInterview(ctx, tech.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetInterview(ctx, tech.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("after delete, err = %v, want sql.ErrNoRows", err)
	}
}

// @scenario [features/job-interviews.feature#interviews-in-date-order] Interviews are listed from earliest to latest
func TestListInterviewsForJobIsInDateOrder(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme"
	acme := createJob(t, s, "Acme", "interviewing", 0)
	globex := createJob(t, s, "Globex", "interviewing", 1)
	// @and [features/job-interviews.feature] I added an "Onsite" interview on 2026-10-20
	createInterview(t, s, acme, "2026-10-20", "", "onsite")
	createInterview(t, s, globex, "2026-10-01", "", "panel")
	// @and [features/job-interviews.feature] I then added a "Phone screen" interview on 2026-10-02
	createInterview(t, s, acme, "2026-10-02", "", "phone_screen")
	// @and [features/job-interviews.feature] I then added a "Technical" interview on 2026-10-14
	createInterview(t, s, acme, "2026-10-14", "", "technical")

	// @when [features/job-interviews.feature] I open the card
	interviews, err := s.ListInterviewsForJob(ctx, acme)
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-interviews.feature] the interviews are listed as "Phone screen", "Technical", and "Onsite" in that order
	if got, want := interviewKinds(interviews), []string{"phone_screen", "technical", "onsite"}; !slices.Equal(got, want) {
		t.Errorf("interviews = %v, want %v", got, want)
	}
}

// @scenario [features/job-interviews.feature#interviews-same-day-by-time] Interviews on the same day are listed by time, with untimed ones last
func TestListInterviewsForJobOrdersSameDayByTime(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme"
	acme := createJob(t, s, "Acme", "interviewing", 0)
	// @and [features/job-interviews.feature] it has a "Panel" interview on 2026-10-14 with no time
	createInterview(t, s, acme, "2026-10-14", "", "panel")
	// @and [features/job-interviews.feature] a "Behavioral" interview on 2026-10-14 at 3:00 PM
	createInterview(t, s, acme, "2026-10-14", "15:00", "behavioral")
	// @and [features/job-interviews.feature] a "Technical" interview on 2026-10-14 at 10:00 AM
	createInterview(t, s, acme, "2026-10-14", "10:00", "technical")

	// @when [features/job-interviews.feature] I open the card
	interviews, err := s.ListInterviewsForJob(ctx, acme)
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-interviews.feature] the interviews are listed as "Technical", "Behavioral", and "Panel" in that order
	if got, want := interviewKinds(interviews), []string{"technical", "behavioral", "panel"}; !slices.Equal(got, want) {
		t.Errorf("interviews = %v, want %v", got, want)
	}
}

func TestListUpcomingInterviews(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	acme := createJob(t, s, "Acme", "interviewing", 0)
	globex := createJob(t, s, "Globex", "interviewing", 1)
	createInterview(t, s, globex, "2026-10-10", "", "panel")
	createInterview(t, s, acme, "2026-10-02", "", "phone_screen")
	createInterview(t, s, acme, "2026-10-20", "", "onsite")
	createInterview(t, s, acme, "2026-10-09", "", "behavioral")
	createInterview(t, s, acme, "2026-10-09", "09:00", "technical")

	interviews, err := s.ListUpcomingInterviews(ctx, "2026-10-09")
	if err != nil {
		t.Fatal(err)
	}
	// Today counts, the past doesn't, and each job's come together in order.
	if got, want := interviewKinds(interviews), []string{"technical", "behavioral", "onsite", "panel"}; !slices.Equal(got, want) {
		t.Errorf("upcoming = %v, want %v", got, want)
	}
}

// @scenario [features/job-interviews.feature#interviews-persist] Interviews are the same after reopening the app
func TestInterviewsPersistAcrossReopen(t *testing.T) {
	dataDir := t.TempDir()
	s, conn := openStore(t, dataDir)
	ctx := context.Background()
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14 at 2:00 PM
	acme := createJob(t, s, "Acme", "interviewing", 0)
	createInterview(t, s, acme, "2026-10-14", "14:00", "technical")

	// @when [features/job-interviews.feature] I close and reopen the app
	conn.Close()
	reopened, _ := openStore(t, dataDir)

	// @and [features/job-interviews.feature] I open the card
	interviews, err := reopened.ListInterviewsForJob(ctx, acme)
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-interviews.feature] the job's interviews show a "Technical" interview on 2026-10-14 at 2:00 PM
	if len(interviews) != 1 || interviews[0].Kind != "technical" || interviews[0].Date != "2026-10-14" || interviews[0].Time != "14:00" {
		t.Errorf("after reopening: %+v", interviews)
	}
}

// @scenario [features/job-interviews.feature#interviews-deleted-with-job] Deleting a job deletes its interviews
func TestDeleteJobDeletesItsInterviewRows(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
	acme := createJob(t, s, "Acme", "interviewing", 0)
	tech := createInterview(t, s, acme, "2026-10-14", "", "technical")
	globex := createJob(t, s, "Globex", "interviewing", 1)
	createInterview(t, s, globex, "2026-10-14", "", "technical")

	// @when [features/job-interviews.feature] I delete the job
	err := s.InTx(ctx, func(q store.Querier) error { return q.DeleteJob(ctx, acme) })
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-interviews.feature] its interviews are removed too
	if _, err := s.GetInterview(ctx, tech.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Acme's interview: err = %v, want sql.ErrNoRows", err)
	}
	if left, _ := s.ListInterviewsForJob(ctx, globex); len(left) != 1 {
		t.Errorf("Globex has %d interviews, want 1", len(left))
	}
}
