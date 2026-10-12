package jobs

import (
	"context"
	"net/mail"
	"strings"

	"callback/internal/store"
)

// Relationship is how a contact relates to a job.
type Relationship string

const (
	RelationshipRecruiter     Relationship = "recruiter"
	RelationshipHiringManager Relationship = "hiring_manager"
	RelationshipReferrer      Relationship = "referrer"
	RelationshipInterviewer   Relationship = "interviewer"
	RelationshipOther         Relationship = "other"
)

var relationshipLabels = map[Relationship]string{
	RelationshipRecruiter:     "Recruiter",
	RelationshipHiringManager: "Hiring manager",
	RelationshipReferrer:      "Referrer",
	RelationshipInterviewer:   "Interviewer",
	RelationshipOther:         "Other",
}

// Label is the relationship as shown to the user.
func (r Relationship) Label() string { return relationshipLabels[r] }

func (r Relationship) valid() bool { _, ok := relationshipLabels[r]; return ok }

// ContactDetails are the fields of a contact the user fills in.
type ContactDetails struct {
	Name         string       `json:"name"`
	Relationship Relationship `json:"relationship"`
	Email        string       `json:"email"`
	// Phone is free text, written however the user has it.
	Phone       string `json:"phone"`
	ProfileLink string `json:"profileLink"`
	Notes       string `json:"notes"`
}

// Contact is a person the user deals with on one job. Each job keeps its own contacts, so the
// same person on two jobs is two contacts, changed separately.
type Contact struct {
	ID           int64        `json:"id"`
	JobID        int64        `json:"jobId"`
	Name         string       `json:"name"`
	Relationship Relationship `json:"relationship"`
	Email        string       `json:"email"`
	Phone        string       `json:"phone"`
	ProfileLink  string       `json:"profileLink"`
	Notes        string       `json:"notes"`
}

func contactFromRow(row store.Contact) Contact {
	return Contact{
		ID:           row.ID,
		JobID:        row.JobID,
		Name:         row.Name,
		Relationship: Relationship(row.Relationship),
		Email:        row.Email,
		Phone:        row.Phone,
		ProfileLink:  row.ProfileLink,
		Notes:        row.Notes,
	}
}

// AddContact adds a contact to a job, after the job's other contacts. Nothing is saved if any
// detail is invalid. A job in any status can have contacts added, and adding one doesn't move it.
func (s *Service) AddContact(ctx context.Context, jobID int64, d ContactDetails) (Contact, error) {
	d, err := validateContact(d)
	if err != nil {
		return Contact{}, err
	}
	if _, err := s.store.GetJob(ctx, jobID); err != nil {
		return Contact{}, notFound(err, ErrJobNotFound)
	}
	row, err := s.store.CreateContact(ctx, store.CreateContactParams{
		JobID:        jobID,
		Name:         d.Name,
		Relationship: string(d.Relationship),
		Email:        d.Email,
		Phone:        d.Phone,
		ProfileLink:  d.ProfileLink,
		Notes:        d.Notes,
	})
	if err != nil {
		return Contact{}, err
	}
	return contactFromRow(row), nil
}

// UpdateContact replaces a contact's details. It changes only this job's contact, even when
// another job has a contact with the same name. Nothing is saved if any detail is invalid.
func (s *Service) UpdateContact(ctx context.Context, id int64, d ContactDetails) (Contact, error) {
	d, err := validateContact(d)
	if err != nil {
		return Contact{}, err
	}
	row, err := s.store.GetContact(ctx, id)
	if err != nil {
		return Contact{}, notFound(err, ErrContactNotFound)
	}
	err = s.store.UpdateContact(ctx, store.UpdateContactParams{
		ID:           id,
		Name:         d.Name,
		Relationship: string(d.Relationship),
		Email:        d.Email,
		Phone:        d.Phone,
		ProfileLink:  d.ProfileLink,
		Notes:        d.Notes,
	})
	if err != nil {
		return Contact{}, err
	}
	return Contact{
		ID: id, JobID: row.JobID, Name: d.Name, Relationship: d.Relationship, Email: d.Email,
		Phone: d.Phone, ProfileLink: d.ProfileLink, Notes: d.Notes,
	}, nil
}

// RemoveContact removes a contact from its job.
func (s *Service) RemoveContact(ctx context.Context, id int64) error {
	if _, err := s.store.GetContact(ctx, id); err != nil {
		return notFound(err, ErrContactNotFound)
	}
	return s.store.DeleteContact(ctx, id)
}

// OpenContactLink opens a contact's profile link in the web browser.
func (s *Service) OpenContactLink(ctx context.Context, id int64) error {
	contact, err := s.store.GetContact(ctx, id)
	if err != nil {
		return notFound(err, ErrContactNotFound)
	}
	if contact.ProfileLink == "" {
		return invalid("this contact has no profile link")
	}
	// Checked again at open time so a stored link can never make the OS launch a local program.
	// See docs/adr/0017-open-documents-and-links-with-a-hand-written-os-opener.md.
	if !isWebAddress(contact.ProfileLink) {
		return invalid(msgProfileLink)
	}
	return s.opener.Open(contact.ProfileLink)
}

const (
	msgEmail       = "the email must be an email address, like dana@example.com"
	msgProfileLink = "the profile link must be a web address, like https://www.linkedin.com/in/dana-smith"
)

// validateContact checks the details the user entered and returns them tidied: name, email, and
// profile link trimmed of surrounding space.
func validateContact(d ContactDetails) (ContactDetails, error) {
	d.Name = strings.TrimSpace(d.Name)
	d.Email = strings.TrimSpace(d.Email)
	d.ProfileLink = strings.TrimSpace(d.ProfileLink)
	if d.Name == "" {
		return d, invalid("name is required")
	}
	if d.Relationship == "" {
		return d, invalid("relationship is required")
	}
	if !d.Relationship.valid() {
		return d, invalid("choose whether the contact is a recruiter, a hiring manager, a referrer, an interviewer, or other")
	}
	if d.Email != "" && !isEmailAddress(d.Email) {
		return d, invalid(msgEmail)
	}
	if d.ProfileLink != "" && !isWebAddress(d.ProfileLink) {
		return d, invalid(msgProfileLink)
	}
	return d, nil
}

// isEmailAddress reports whether email is a bare address like dana@acme.example.com, with no
// display name or angle brackets.
func isEmailAddress(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Name == "" && addr.Address == email
}
