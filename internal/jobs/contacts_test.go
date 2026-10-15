package jobs

import (
	"errors"
	"slices"
	"testing"

	"callback/internal/store"
)

// dana is the contact most scenarios add: Dana Smith, a recruiter, and nothing else.
func dana() ContactDetails {
	return ContactDetails{Name: "Dana Smith", Relationship: RelationshipRecruiter}
}

func (h *harness) mustAddContact(t *testing.T, jobID int64, d ContactDetails) Contact {
	t.Helper()
	c, err := h.svc.AddContact(h.ctx, jobID, d)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// contactsOf lists a job's contacts as "name as relationship", in the order the job shows them.
func (h *harness) contactsOf(t *testing.T, jobID int64) []string {
	t.Helper()
	var out []string
	for _, c := range h.job(t, jobID).Contacts {
		out = append(out, c.Name+" as "+c.Relationship.Label())
	}
	return out
}

// onlyContact returns the job's one contact.
func (h *harness) onlyContact(t *testing.T, jobID int64) Contact {
	t.Helper()
	contacts := h.job(t, jobID).Contacts
	if len(contacts) != 1 {
		t.Fatalf("job has %d contacts, want 1", len(contacts))
	}
	return contacts[0]
}

// @scenario [#add-contact]
func TestAddContactWithAllDetails(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)

	// @when [features/job-contacts.feature] I add a contact with:
	h.mustAddContact(t, id, ContactDetails{
		Name:         "Dana Smith",
		Relationship: RelationshipRecruiter,
		Email:        "dana@acme.example.com",
		Phone:        "(555) 010-4477",
		ProfileLink:  "https://www.linkedin.com/in/dana-smith",
		Notes:        "Prefers email. Out of office Fridays.",
	})

	// @then [features/job-contacts.feature] the job's contacts show "Dana Smith" as a recruiter
	if got := h.contactsOf(t, id); !slices.Equal(got, []string{"Dana Smith as Recruiter"}) {
		t.Errorf("contacts = %v", got)
	}
	// @and [features/job-contacts.feature] the contact shows the email, phone, profile link, and notes
	c := h.onlyContact(t, id)
	if c.Email != "dana@acme.example.com" || c.Phone != "(555) 010-4477" ||
		c.ProfileLink != "https://www.linkedin.com/in/dana-smith" || c.Notes != "Prefers email. Out of office Fridays." {
		t.Errorf("contact = %+v", c)
	}
}

// @scenario [#add-contact-minimum]
func TestAddContactMinimum(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)

	// @when [features/job-contacts.feature] I add the contact "Dana Smith" as a recruiter and nothing else
	h.mustAddContact(t, id, dana())

	// @then [features/job-contacts.feature] the job's contacts show "Dana Smith" as a recruiter
	if got := h.contactsOf(t, id); !slices.Equal(got, []string{"Dana Smith as Recruiter"}) {
		t.Errorf("contacts = %v", got)
	}
	if c := h.onlyContact(t, id); c.Email != "" || c.Phone != "" || c.ProfileLink != "" || c.Notes != "" {
		t.Errorf("contact = %+v, want only a name and a relationship", c)
	}
}

// @scenario [#contact-relationships]
func TestContactRelationships(t *testing.T) {
	tests := []struct {
		spec string
		rel  Relationship
	}{
		{"a recruiter", RelationshipRecruiter},
		{"a hiring manager", RelationshipHiringManager},
		{"a referrer", RelationshipReferrer},
		{"an interviewer", RelationshipInterviewer},
		{"other", RelationshipOther},
	}
	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
			id := h.acme(StatusSaved)

			// @when [features/job-contacts.feature] I add the contact "Dana Smith" as <relationship>
			h.mustAddContact(t, id, ContactDetails{Name: "Dana Smith", Relationship: tt.rel})

			// @then [features/job-contacts.feature] the job's contacts show "Dana Smith" as <relationship>
			if got := h.onlyContact(t, id).Relationship; got != tt.rel {
				t.Errorf("relationship = %q, want %q", got, tt.rel)
			}
		})
	}
}

// @scenario [#contact-missing-required]
func TestContactMissingRequired(t *testing.T) {
	tests := []struct {
		name, rel, missing string
	}{
		{"", "Recruiter", "name"},
		{"   ", "Recruiter", "name"},
		{"Dana Smith", "", "relationship"},
	}
	for _, tt := range tests {
		t.Run(tt.missing, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
			id := h.acme(StatusSaved)
			rel := Relationship("")
			if tt.rel != "" {
				rel = RelationshipRecruiter
			}

			// @when [features/job-contacts.feature] I try to add a contact with name "<name>" and relationship "<relationship>"
			_, err := h.svc.AddContact(h.ctx, id, ContactDetails{Name: tt.name, Relationship: rel})

			// @then [features/job-contacts.feature] the contact is not added
			if got := h.contactsOf(t, id); len(got) != 0 {
				t.Errorf("contacts = %v, want none", got)
			}
			// @and [features/job-contacts.feature] I am told that the <missing> is required
			wantValidation(t, err, tt.missing+" is required")
		})
	}
}

// @scenario [#contact-invalid-email]
func TestContactInvalidEmail(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)

	// @when [features/job-contacts.feature] I try to add the contact "Dana Smith" as a recruiter with the email "dana at acme"
	d := dana()
	d.Email = "dana at acme"
	_, err := h.svc.AddContact(h.ctx, id, d)

	// @then [features/job-contacts.feature] the contact is not added
	if got := h.contactsOf(t, id); len(got) != 0 {
		t.Errorf("contacts = %v, want none", got)
	}
	// @and [features/job-contacts.feature] I am told the email must be an email address
	wantValidation(t, err, "the email must be an email address, like dana@example.com")
}

func TestEmailAddressCheck(t *testing.T) {
	for _, email := range []string{"dana@acme.example.com", "dana.smith+jobs@acme.io", "d@x"} {
		if !isEmailAddress(email) {
			t.Errorf("%q was rejected", email)
		}
	}
	for _, email := range []string{"dana at acme", "dana", "@acme.com", "dana@", "Dana <dana@acme.com>", "dana@acme.com, lee@acme.com"} {
		if isEmailAddress(email) {
			t.Errorf("%q was accepted", email)
		}
	}
}

// @scenario [#contact-invalid-profile-link]
func TestContactInvalidProfileLink(t *testing.T) {
	for _, link := range []string{"not a url", "linkedin.com/in/dana-smith", "file:///C:/Windows/System32/calc.exe"} {
		t.Run(link, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
			id := h.acme(StatusSaved)

			// @when [features/job-contacts.feature] I try to add the contact "Dana Smith" as a recruiter with the profile link "not a url"
			d := dana()
			d.ProfileLink = link
			_, err := h.svc.AddContact(h.ctx, id, d)

			// @then [features/job-contacts.feature] the contact is not added
			if got := h.contactsOf(t, id); len(got) != 0 {
				t.Errorf("contacts = %v, want none", got)
			}
			// @and [features/job-contacts.feature] I am told the profile link must be a web address
			wantValidation(t, err, "the profile link must be a web address, like https://www.linkedin.com/in/dana-smith")
		})
	}
}

// @scenario [#contact-phone-free-text]
func TestContactPhoneFreeText(t *testing.T) {
	for _, phone := range []string{"(555) 010-4477", "+44 20 7946 0958", "555-0104 ext. 12"} {
		t.Run(phone, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
			id := h.acme(StatusSaved)

			// @when [features/job-contacts.feature] I add the contact "Dana Smith" as a recruiter with the phone "<phone>"
			d := dana()
			d.Phone = phone
			h.mustAddContact(t, id, d)

			// @then [features/job-contacts.feature] the contact shows the phone "<phone>"
			if got := h.onlyContact(t, id).Phone; got != phone {
				t.Errorf("phone = %q, want %q", got, phone)
			}
		})
	}
}

// @scenario [#contact-any-status]
func TestAddContactInAnyStatus(t *testing.T) {
	for _, status := range Statuses {
		t.Run(status.Label(), func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" in "<status>"
			id := h.acme(statusFromLabel(t, status.Label()))
			before := h.job(t, id)

			// @when [features/job-contacts.feature] I add the contact "Dana Smith" as a recruiter
			h.mustAddContact(t, id, dana())

			// @then [features/job-contacts.feature] the job's contacts show "Dana Smith" as a recruiter
			if got := h.contactsOf(t, id); !slices.Equal(got, []string{"Dana Smith as Recruiter"}) {
				t.Errorf("contacts = %v", got)
			}
			// @and [features/job-contacts.feature] the card stays in the "<status>" column
			if _, got := h.card(t, id); got != status {
				t.Errorf("card is in %q, want %q", got, status)
			}
			if after := h.job(t, id); after.DateApplied != before.DateApplied || after.ClosingReason != before.ClosingReason {
				t.Errorf("adding a contact changed the job: %+v, was %+v", after, before)
			}
		})
	}
}

func TestAddContactToMissingJob(t *testing.T) {
	h := newHarness(t)
	if _, err := h.svc.AddContact(h.ctx, 99, dana()); !errors.Is(err, ErrJobNotFound) {
		t.Errorf("err = %v, want ErrJobNotFound", err)
	}
}

func TestAddContactRejectsAnUnknownRelationship(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	_, err := h.svc.AddContact(h.ctx, id, ContactDetails{Name: "Dana Smith", Relationship: "boss"})
	wantValidation(t, err, "choose whether the contact is a recruiter, a hiring manager, a referrer, an interviewer, or other")
}

func TestAddContactTrimsNameEmailAndLink(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	c := h.mustAddContact(t, id, ContactDetails{
		Name: "  Dana Smith ", Relationship: RelationshipRecruiter,
		Email: " dana@acme.example.com ", ProfileLink: " https://www.linkedin.com/in/dana-smith ",
	})
	if c.Name != "Dana Smith" || c.Email != "dana@acme.example.com" || c.ProfileLink != "https://www.linkedin.com/in/dana-smith" {
		t.Errorf("contact = %+v", c)
	}
}

// @scenario [#view-contacts]
func TestViewContactsInOrderAdded(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)
	// @and [features/job-contacts.feature] I added the contact "Dana Smith" as a recruiter
	h.mustAddContact(t, id, dana())
	// @and [features/job-contacts.feature] I then added the contact "Lee Park" as a hiring manager
	h.mustAddContact(t, id, ContactDetails{Name: "Lee Park", Relationship: RelationshipHiringManager})

	// @when [features/job-contacts.feature] I open the card
	got := h.contactsOf(t, id)

	// @then [features/job-contacts.feature] I see "Dana Smith" and "Lee Park" in the order I added them
	if want := []string{"Dana Smith as Recruiter", "Lee Park as Hiring manager"}; !slices.Equal(got, want) {
		t.Errorf("contacts = %v, want %v", got, want)
	}
}

// @scenario [#open-contact-profile-link]
func TestOpenContactLink(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)
	// @and [features/job-contacts.feature] the contact "Dana Smith" has the profile link "https://www.linkedin.com/in/dana-smith"
	contact := h.seedContact(id, store.Contact{Name: "Dana Smith", ProfileLink: "https://www.linkedin.com/in/dana-smith"})

	// @when [features/job-contacts.feature] I open Dana Smith's profile link
	if err := h.svc.OpenContactLink(h.ctx, contact); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-contacts.feature] the profile opens in my web browser
	if !slices.Equal(h.opener.opened, []string{"https://www.linkedin.com/in/dana-smith"}) {
		t.Errorf("opened %v", h.opener.opened)
	}
}

func TestOpenContactLinkRefusesWhatIsntAWebAddress(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	none := h.seedContact(id, store.Contact{Name: "Dana Smith"})
	// Written straight to the store, as an older or hand-edited database could hold it.
	local := h.seedContact(id, store.Contact{Name: "Lee Park", ProfileLink: `C:\Windows\System32\calc.exe`})

	wantValidation(t, h.svc.OpenContactLink(h.ctx, none), "this contact has no profile link")
	wantValidation(t, h.svc.OpenContactLink(h.ctx, local), "the profile link must be a web address, like https://www.linkedin.com/in/dana-smith")
	if err := h.svc.OpenContactLink(h.ctx, 99); !errors.Is(err, ErrContactNotFound) {
		t.Errorf("err = %v, want ErrContactNotFound", err)
	}
	if len(h.opener.opened) != 0 {
		t.Errorf("opened %v, want nothing", h.opener.opened)
	}
}

// @scenario [#edit-contact]
func TestEditContact(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith" as a recruiter
	id := h.acme(StatusSaved)
	contact := h.seedContact(id, store.Contact{Name: "Dana Smith", Relationship: "recruiter", Email: "dana@acme.example.com", Phone: "(555) 010-4477"})

	d := ContactDetails{Name: "Dana Smith", Relationship: RelationshipRecruiter, Phone: "(555) 010-4477"}
	// @when [features/job-contacts.feature] I change Dana Smith's email to "dana.smith@acme.example.com"
	d.Email = "dana.smith@acme.example.com"
	// @and [features/job-contacts.feature] I change the relationship to hiring manager
	d.Relationship = RelationshipHiringManager
	updated, err := h.svc.UpdateContact(h.ctx, contact, d)
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-contacts.feature] the job's contacts show "Dana Smith" as a hiring manager
	if got := h.contactsOf(t, id); !slices.Equal(got, []string{"Dana Smith as Hiring manager"}) {
		t.Errorf("contacts = %v", got)
	}
	// @and [features/job-contacts.feature] the contact shows the email "dana.smith@acme.example.com"
	if c := h.onlyContact(t, id); c.Email != "dana.smith@acme.example.com" || c.Phone != "(555) 010-4477" || c != updated {
		t.Errorf("contact = %+v, returned %+v", c, updated)
	}
}

func TestEditContactKeepsItWhenInvalid(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	contact := h.seedContact(id, store.Contact{Name: "Dana Smith", Email: "dana@acme.example.com"})

	_, err := h.svc.UpdateContact(h.ctx, contact, ContactDetails{Name: "Dana Smith", Relationship: RelationshipRecruiter, Email: "dana at acme"})
	wantValidation(t, err, "the email must be an email address, like dana@example.com")
	if c := h.onlyContact(t, id); c.Email != "dana@acme.example.com" {
		t.Errorf("email = %q, want it unchanged", c.Email)
	}

	if _, err := h.svc.UpdateContact(h.ctx, 99, dana()); !errors.Is(err, ErrContactNotFound) {
		t.Errorf("err = %v, want ErrContactNotFound", err)
	}
}

// @scenario [#remove-contact]
func TestRemoveContact(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith"
	id := h.acme(StatusSaved)
	contact := h.seedContact(id, store.Contact{Name: "Dana Smith"})
	h.seedContact(id, store.Contact{Name: "Lee Park"})

	// @when [features/job-contacts.feature] I remove the contact
	// ("And I confirm the removal" is the frontend's to ask.)
	if err := h.svc.RemoveContact(h.ctx, contact); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-contacts.feature] the job no longer shows "Dana Smith"
	if got := h.contactsOf(t, id); !slices.Equal(got, []string{"Lee Park as Recruiter"}) {
		t.Errorf("contacts = %v, want only Lee Park", got)
	}

	if err := h.svc.RemoveContact(h.ctx, contact); !errors.Is(err, ErrContactNotFound) {
		t.Errorf("removing it again: err = %v, want ErrContactNotFound", err)
	}
}

// @scenario [#same-person-many-jobs]
func TestSamePersonOnManyJobs(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" and a job for "Developer" at "Globex"
	acme := h.acme(StatusSaved)
	globex := h.seed(store.Job{Company: "Globex", Role: "Developer"})

	// @when [features/job-contacts.feature] I add the contact "Dana Smith" as a recruiter to both jobs
	h.mustAddContact(t, acme, dana())
	h.mustAddContact(t, globex, dana())

	// @then [features/job-contacts.feature] both jobs show "Dana Smith" as a recruiter
	for _, id := range []int64{acme, globex} {
		if got := h.contactsOf(t, id); !slices.Equal(got, []string{"Dana Smith as Recruiter"}) {
			t.Errorf("job %d contacts = %v", id, got)
		}
	}
}

// @scenario [#contacts-edited-separately]
func TestContactsEditedSeparately(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-contacts.feature] I have a job at "Acme" and a job at "Globex" that both have the contact "Dana Smith"
	acme := h.acme(StatusSaved)
	globex := h.seed(store.Job{Company: "Globex"})
	acmeDana := h.mustAddContact(t, acme, ContactDetails{Name: "Dana Smith", Relationship: RelationshipRecruiter, Email: "dana@acme.example.com"})
	h.mustAddContact(t, globex, ContactDetails{Name: "Dana Smith", Relationship: RelationshipRecruiter, Email: "dana@acme.example.com"})

	// @when [features/job-contacts.feature] I change Dana Smith's email on the "Acme" job
	_, err := h.svc.UpdateContact(h.ctx, acmeDana.ID, ContactDetails{Name: "Dana Smith", Relationship: RelationshipRecruiter, Email: "dana.smith@acme.example.com"})
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-contacts.feature] the "Globex" job's contact "Dana Smith" keeps the email it had
	if got := h.onlyContact(t, globex).Email; got != "dana@acme.example.com" {
		t.Errorf("Globex's Dana has the email %q, want dana@acme.example.com", got)
	}
	if got := h.onlyContact(t, acme).Email; got != "dana.smith@acme.example.com" {
		t.Errorf("Acme's Dana has the email %q", got)
	}
}

// @scenario [#move-keeps-contacts]
func TestMoveKeepsContacts(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" in "Applied" with the contact "Dana Smith"
	id := h.seed(store.Job{Company: "Acme", Status: string(StatusApplied), DateApplied: nullString("2026-10-01")})
	h.seedContact(id, store.Contact{Name: "Dana Smith", Email: "dana@acme.example.com"})
	before := h.job(t, id).Contacts

	// @when [features/job-contacts.feature] I move the card to "Interviewing"
	mustMove(t, h, Move{JobID: id, Status: StatusInterviewing})

	// @then [features/job-contacts.feature] the job still shows "Dana Smith"
	if after := h.job(t, id).Contacts; !slices.Equal(after, before) {
		t.Errorf("contacts after the move = %+v, before %+v", after, before)
	}
}

// @scenario [#contacts-deleted-with-job]
func TestDeleteJobDeletesItsContacts(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith"
	id := h.acme(StatusSaved)
	h.seedContact(id, store.Contact{Name: "Dana Smith"})
	globex := h.seed(store.Job{Company: "Globex"})
	h.seedContact(globex, store.Contact{Name: "Dana Smith"})

	// @when [features/job-contacts.feature] I delete the job
	if err := h.svc.DeleteJob(h.ctx, id); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-contacts.feature] its contacts are removed too
	if contacts, _ := h.store.ListContactsForJob(h.ctx, id); len(contacts) != 0 {
		t.Errorf("%d contacts left after deleting the job", len(contacts))
	}
	if got := h.contactsOf(t, globex); len(got) != 1 {
		t.Errorf("Globex's contacts = %v, want Dana Smith kept", got)
	}
}

func TestContact(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	added := h.mustAddContact(t, id, ContactDetails{Name: "Dana Smith", Relationship: RelationshipRecruiter, Email: "dana@acme.example.com"})

	got, err := h.svc.Contact(h.ctx, added.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != added {
		t.Errorf("Contact = %+v, want %+v", got, added)
	}
	if _, err := h.svc.Contact(h.ctx, 999); !errors.Is(err, ErrContactNotFound) {
		t.Errorf("err = %v, want ErrContactNotFound", err)
	}
}

func TestRelationshipsListsEveryOne(t *testing.T) {
	if len(Relationships) != len(relationshipLabels) {
		t.Errorf("Relationships has %d, labels have %d", len(Relationships), len(relationshipLabels))
	}
	for _, r := range Relationships {
		if !r.valid() {
			t.Errorf("%q isn't a relationship", r)
		}
	}
}
