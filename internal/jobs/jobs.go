// Package jobs holds the business rules for the job board, job cards, job documents, and job
// contacts. The Wails bindings in app.go call it, and it reaches storage only through the Store
// and Files interfaces. See docs/adr/0016-put-business-rules-in-services-over-a-transactional-store-interface.md.
package jobs

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"callback/internal/store"
)

// Store is the database: the sqlc queries plus transactions. internal/db.Store implements it.
type Store interface {
	store.Querier
	InTx(ctx context.Context, fn func(store.Querier) error) error
}

// Files keeps the app's own copies of attached documents. internal/docfiles.Dir implements it.
// See docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md.
type Files interface {
	// Size returns the size in bytes of the file at src.
	Size(src string) (int64, error)
	// Save copies the file at src and returns the copy's stored path.
	Save(src string) (stored string, err error)
	// Remove deletes a stored copy.
	Remove(stored string) error
	// Path returns the absolute path of a stored copy.
	Path(stored string) string
}

// Opener opens a file or web address in the OS default handler. internal/desktop.Opener
// implements it. See docs/adr/0017-open-documents-and-links-with-a-hand-written-os-opener.md.
type Opener interface {
	Open(target string) error
}

// Service carries out what the user does on the board, on job cards, and with documents and
// contacts.
type Service struct {
	store  Store
	files  Files
	opener Opener
	now    func() time.Time
}

// New returns a Service. now supplies the current time, which decides "today" for dates applied.
func New(store Store, files Files, opener Opener, now func() time.Time) *Service {
	return &Service{store: store, files: files, opener: opener, now: now}
}

// dateLayout is how dates applied are written: YYYY-MM-DD.
const dateLayout = "2006-01-02"

// today returns the current local date as YYYY-MM-DD.
func (s *Service) today() string {
	return s.now().Format(dateLayout)
}

// ValidationError is a problem with what the user entered or chose. Its message is written to be
// shown to the user as is.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

func invalid(msg string) error { return &ValidationError{Msg: msg} }

var (
	// ErrJobNotFound means no job has the given ID.
	ErrJobNotFound = errors.New("job not found")
	// ErrDocumentNotFound means no document has the given ID.
	ErrDocumentNotFound = errors.New("document not found")
	// ErrContactNotFound means no contact has the given ID.
	ErrContactNotFound = errors.New("contact not found")
)

// notFound turns sql.ErrNoRows from a lookup into want.
func notFound(err, want error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return want
	}
	return err
}

// Status is a column on the board.
type Status string

const (
	StatusSaved        Status = "saved"
	StatusApplied      Status = "applied"
	StatusInterviewing Status = "interviewing"
	StatusOffer        Status = "offer"
	StatusClosed       Status = "closed"
)

// Statuses lists every status in the board's column order, left to right.
var Statuses = []Status{StatusSaved, StatusApplied, StatusInterviewing, StatusOffer, StatusClosed}

var statusLabels = map[Status]string{
	StatusSaved:        "Saved",
	StatusApplied:      "Applied",
	StatusInterviewing: "Interviewing",
	StatusOffer:        "Offer",
	StatusClosed:       "Closed",
}

// Label is the status's column title.
func (s Status) Label() string { return statusLabels[s] }

func (s Status) valid() bool { _, ok := statusLabels[s]; return ok }

// ClosingReason records why a job was closed.
type ClosingReason string

const (
	ReasonRejected       ClosingReason = "rejected"
	ReasonWithdrew       ClosingReason = "withdrew"
	ReasonNoResponse     ClosingReason = "no_response"
	ReasonDeclinedOffer  ClosingReason = "declined_offer"
	ReasonPositionFilled ClosingReason = "position_filled"
	// ReasonOther comes with the user's own text.
	ReasonOther ClosingReason = "other"
)

var reasonLabels = map[ClosingReason]string{
	ReasonRejected:       "Rejected",
	ReasonWithdrew:       "Withdrew",
	ReasonNoResponse:     "No response",
	ReasonDeclinedOffer:  "Declined offer",
	ReasonPositionFilled: "Position filled",
	ReasonOther:          "Other",
}

// Label is the reason as shown to the user.
func (r ClosingReason) Label() string { return reasonLabels[r] }

func (r ClosingReason) valid() bool { _, ok := reasonLabels[r]; return ok }

// DocumentKind is what an attached document is for.
type DocumentKind string

const (
	KindResume      DocumentKind = "resume"
	KindCoverLetter DocumentKind = "cover_letter"
	KindOther       DocumentKind = "other"
)

func (k DocumentKind) valid() bool {
	return k == KindResume || k == KindCoverLetter || k == KindOther
}

// MinInterest and MaxInterest are the fewest and most stars an interest rating can have.
const (
	MinInterest = 1
	MaxInterest = 5
)

// nullInterest turns an unrated interest (nil) into NULL.
func nullInterest(stars *int) sql.NullInt64 {
	if stars == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*stars), Valid: true}
}

// interestFromRow turns a stored interest rating back into stars, or nil when it's NULL.
func interestFromRow(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	stars := int(n.Int64)
	return &stars
}

// nullString turns "" into NULL.
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
