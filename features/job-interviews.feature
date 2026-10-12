Feature: Job interviews
  As a job seeker
  I want to keep a list of the interviews for each job
  So that I know what's coming up and can look back at how each one went

  Rule: An interview has a date and a kind

    @status:implemented @id:add-interview
    Scenario: Add an interview with all its details
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing"
      When I add an interview with:
        | date  | 2026-10-14                          |
        | time  | 2:00 PM                             |
        | kind  | Technical                           |
        | notes | Live coding with two engineers.     |
      Then the job's interviews show a "Technical" interview on 2026-10-14 at 2:00 PM
      And the interview shows the notes "Live coding with two engineers."

    @status:implemented @id:add-interview-minimum
    Scenario: Add an interview with only a date and a kind
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing"
      When I add a "Phone screen" interview on 2026-10-14 with no time and no notes
      Then the job's interviews show a "Phone screen" interview on 2026-10-14

    @status:implemented @id:interview-kinds
    Scenario Outline: Choose what kind of interview it is
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing"
      When I add a "<kind>" interview on 2026-10-14
      Then the job's interviews show a "<kind>" interview on 2026-10-14

      Examples:
        | kind         |
        | Phone screen |
        | Technical    |
        | Behavioral   |
        | Panel        |
        | Onsite       |
        | Other        |

    @status:implemented @id:interview-missing-required
    Scenario Outline: An interview can't be saved without a date or a kind
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing"
      When I try to add an interview with date "<date>" and kind "<kind>"
      Then the interview is not added
      And I am told that the <missing> is required

      Examples:
        | date       | kind      | missing |
        |            | Technical | date    |
        | 2026-10-14 |           | kind    |

    @status:implemented @id:interview-past-date
    Scenario: An interview that already happened can be added
      Given today is 2026-10-09
      And I have a job for "Software Engineer" at "Acme" in "Interviewing"
      When I add a "Phone screen" interview on 2026-10-02
      Then the job's interviews show a "Phone screen" interview on 2026-10-02

    @status:implemented @id:interview-any-status
    Scenario Outline: Interviews can be added to a job in any status
      Given I have a job for "Software Engineer" at "Acme" in "<status>"
      When I add a "Phone screen" interview on 2026-10-14
      Then the job's interviews show a "Phone screen" interview on 2026-10-14
      And the card stays in the "<status>" column

      Examples:
        | status       |
        | Saved        |
        | Applied      |
        | Interviewing |
        | Offer        |
        | Closed       |

  Rule: A job's interviews are listed in date order

    @status:implemented @id:view-interviews
    Scenario: See a job's interviews when opening its card
      Given I have a job for "Software Engineer" at "Acme" with a "Phone screen" interview on 2026-10-02
      And a "Technical" interview on 2026-10-14
      When I open the card
      Then I see both interviews

    @status:implemented @id:interviews-in-date-order
    Scenario: Interviews are listed from earliest to latest
      Given I have a job for "Software Engineer" at "Acme"
      And I added an "Onsite" interview on 2026-10-20
      And I then added a "Phone screen" interview on 2026-10-02
      And I then added a "Technical" interview on 2026-10-14
      When I open the card
      Then the interviews are listed as "Phone screen", "Technical", and "Onsite" in that order

    @status:implemented @id:interviews-same-day-by-time
    Scenario: Interviews on the same day are listed by time, with untimed ones last
      Given I have a job for "Software Engineer" at "Acme"
      And it has a "Panel" interview on 2026-10-14 with no time
      And a "Behavioral" interview on 2026-10-14 at 3:00 PM
      And a "Technical" interview on 2026-10-14 at 10:00 AM
      When I open the card
      Then the interviews are listed as "Technical", "Behavioral", and "Panel" in that order

    @status:implemented @id:past-interviews-marked
    Scenario: Interviews that have already happened are shown as past
      Given today is 2026-10-09
      And I have a job for "Software Engineer" at "Acme" with a "Phone screen" interview on 2026-10-02
      And a "Technical" interview on 2026-10-14
      When I open the card
      Then the "Phone screen" interview is shown as past
      And the "Technical" interview is shown as upcoming

    @status:implemented @id:no-interviews
    Scenario: A job with no interviews says so
      Given I have a job for "Software Engineer" at "Acme" with no interviews
      When I open the card
      Then I am told the job has no interviews yet

  Rule: The board card shows the next interview

    @status:implemented @id:card-shows-next-interview
    Scenario: The card shows the date of the next upcoming interview
      Given today is 2026-10-09
      And I have a job for "Software Engineer" at "Acme" with a "Phone screen" interview on 2026-10-02
      And a "Technical" interview on 2026-10-14
      And an "Onsite" interview on 2026-10-20
      When I open the job search board
      Then the card shows a "Technical" interview on 2026-10-14

    @status:implemented @id:card-interview-today
    Scenario: An interview today counts as upcoming
      Given today is 2026-10-14
      And I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
      When I open the job search board
      Then the card shows a "Technical" interview today

    @status:implemented @id:card-no-upcoming-interview
    Scenario: The card shows no interview when every interview has passed
      Given today is 2026-10-09
      And I have a job for "Software Engineer" at "Acme" with only a "Phone screen" interview on 2026-10-02
      When I open the job search board
      Then the card doesn't show an interview

  Rule: Interviews can be changed and removed

    @status:implemented @id:edit-interview
    Scenario: Edit an interview
      Given I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
      When I change the interview's date to 2026-10-16
      And I write in its notes "Rescheduled by the recruiter."
      Then the job's interviews show a "Technical" interview on 2026-10-16
      And the interview shows the notes "Rescheduled by the recruiter."

    @status:implemented @id:remove-interview
    Scenario: Remove an interview after confirming
      Given I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
      When I remove the interview
      And I confirm the removal
      Then the job no longer shows the "Technical" interview

    @status:implemented @id:cancel-remove-interview
    Scenario: Cancel removing an interview
      Given I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
      When I remove the interview
      But I cancel the removal
      Then the job still shows the "Technical" interview on 2026-10-14

  Rule: Interviews are kept with the job

    @status:implemented @id:move-keeps-interviews
    Scenario: Moving a card keeps its interviews
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing" with a "Technical" interview on 2026-10-14
      When I move the card to "Offer"
      Then the job still shows the "Technical" interview on 2026-10-14

    @status:implemented @id:interviews-persist
    Scenario: Interviews are the same after reopening the app
      Given I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14 at 2:00 PM
      When I close and reopen the app
      And I open the card
      Then the job's interviews show a "Technical" interview on 2026-10-14 at 2:00 PM

    @status:implemented @id:interviews-deleted-with-job
    Scenario: Deleting a job deletes its interviews
      Given I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
      When I delete the job
      And I confirm the deletion
      Then its interviews are removed too
