Feature: Job contacts
  As a job seeker
  I want to keep the people I'm dealing with on each job
  So that I know who to follow up with and how to reach them

  Rule: A contact has a name and a relationship to the job

    @status:implemented @id:add-contact
    Scenario: Add a contact with all their details
      Given I have a job for "Software Engineer" at "Acme"
      When I add a contact with:
        | name         | Dana Smith                                |
        | relationship | Recruiter                                 |
        | email        | dana@acme.example.com                     |
        | phone        | (555) 010-4477                            |
        | profile link | https://www.linkedin.com/in/dana-smith    |
        | notes        | Prefers email. Out of office Fridays.     |
      Then the job's contacts show "Dana Smith" as a recruiter
      And the contact shows the email, phone, profile link, and notes

    @status:implemented @id:add-contact-minimum
    Scenario: Add a contact with only a name and a relationship
      Given I have a job for "Software Engineer" at "Acme"
      When I add the contact "Dana Smith" as a recruiter and nothing else
      Then the job's contacts show "Dana Smith" as a recruiter

    @status:implemented @id:contact-relationships
    Scenario Outline: Choose how a contact relates to the job
      Given I have a job for "Software Engineer" at "Acme"
      When I add the contact "Dana Smith" as <relationship>
      Then the job's contacts show "Dana Smith" as <relationship>

      Examples:
        | relationship     |
        | a recruiter      |
        | a hiring manager |
        | a referrer       |
        | an interviewer   |
        | other            |

    @status:implemented @id:contact-missing-required
    Scenario Outline: A contact can't be saved without a name or a relationship
      Given I have a job for "Software Engineer" at "Acme"
      When I try to add a contact with name "<name>" and relationship "<relationship>"
      Then the contact is not added
      And I am told that the <missing> is required

      Examples:
        | name       | relationship | missing      |
        |            | Recruiter    | name         |
        | Dana Smith |              | relationship |

    @status:implemented @id:contact-invalid-email
    Scenario: A contact's email must be an email address
      Given I have a job for "Software Engineer" at "Acme"
      When I try to add the contact "Dana Smith" as a recruiter with the email "dana at acme"
      Then the contact is not added
      And I am told the email must be an email address

    @status:implemented @id:contact-invalid-profile-link
    Scenario: A contact's profile link must be a web address
      Given I have a job for "Software Engineer" at "Acme"
      When I try to add the contact "Dana Smith" as a recruiter with the profile link "not a url"
      Then the contact is not added
      And I am told the profile link must be a web address

    @status:implemented @id:contact-phone-free-text
    Scenario Outline: A phone number is written however I have it
      Given I have a job for "Software Engineer" at "Acme"
      When I add the contact "Dana Smith" as a recruiter with the phone "<phone>"
      Then the contact shows the phone "<phone>"

      Examples:
        | phone              |
        | (555) 010-4477     |
        | +44 20 7946 0958   |
        | 555-0104 ext. 12   |

    @status:implemented @id:contact-any-status
    Scenario Outline: Contacts can be added to a job in any status
      Given I have a job for "Software Engineer" at "Acme" in "<status>"
      When I add the contact "Dana Smith" as a recruiter
      Then the job's contacts show "Dana Smith" as a recruiter
      And the card stays in the "<status>" column

      Examples:
        | status       |
        | Saved        |
        | Applied      |
        | Interviewing |
        | Offer        |
        | Closed       |

  Rule: A job's contacts are shown when I open its card

    @status:implemented @id:view-contacts
    Scenario: See a job's contacts when opening its card
      Given I have a job for "Software Engineer" at "Acme"
      And I added the contact "Dana Smith" as a recruiter
      And I then added the contact "Lee Park" as a hiring manager
      When I open the card
      Then I see "Dana Smith" and "Lee Park" in the order I added them

    @status:implemented @id:no-contacts
    Scenario: A job with no contacts says so
      Given I have a job for "Software Engineer" at "Acme" with no contacts
      When I open the card
      Then I am told the job has no contacts yet

    @status:implemented @id:open-contact-profile-link
    Scenario: Open a contact's profile link
      Given I have a job for "Software Engineer" at "Acme"
      And the contact "Dana Smith" has the profile link "https://www.linkedin.com/in/dana-smith"
      When I open Dana Smith's profile link
      Then the profile opens in my web browser

  Rule: Contacts can be changed and removed

    @status:implemented @id:edit-contact
    Scenario: Edit a contact
      Given I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith" as a recruiter
      When I change Dana Smith's email to "dana.smith@acme.example.com"
      And I change the relationship to hiring manager
      Then the job's contacts show "Dana Smith" as a hiring manager
      And the contact shows the email "dana.smith@acme.example.com"

    @status:implemented @id:remove-contact
    Scenario: Remove a contact after confirming
      Given I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith"
      When I remove the contact
      And I confirm the removal
      Then the job no longer shows "Dana Smith"

    @status:implemented @id:cancel-remove-contact
    Scenario: Cancel removing a contact
      Given I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith"
      When I remove the contact
      But I cancel the removal
      Then the job still shows "Dana Smith"

  Rule: Each job keeps its own contacts

    @status:implemented @id:same-person-many-jobs
    Scenario: The same person can be a contact on more than one job
      Given I have a job for "Software Engineer" at "Acme" and a job for "Developer" at "Globex"
      When I add the contact "Dana Smith" as a recruiter to both jobs
      Then both jobs show "Dana Smith" as a recruiter

    @status:implemented @id:contacts-edited-separately
    Scenario: Editing a contact on one job doesn't change it on another
      Given I have a job at "Acme" and a job at "Globex" that both have the contact "Dana Smith"
      When I change Dana Smith's email on the "Acme" job
      Then the "Globex" job's contact "Dana Smith" keeps the email it had

  Rule: Contacts are kept with the job

    @status:implemented @id:move-keeps-contacts
    Scenario: Moving a card keeps its contacts
      Given I have a job for "Software Engineer" at "Acme" in "Applied" with the contact "Dana Smith"
      When I move the card to "Interviewing"
      Then the job still shows "Dana Smith"

    @status:implemented @id:contacts-persist
    Scenario: Contacts are the same after reopening the app
      Given I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith" as a recruiter
      When I close and reopen the app
      And I open the card
      Then the job's contacts show "Dana Smith" as a recruiter

    @status:implemented @id:contacts-deleted-with-job
    Scenario: Deleting a job deletes its contacts
      Given I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith"
      When I delete the job
      And I confirm the deletion
      Then its contacts are removed too
