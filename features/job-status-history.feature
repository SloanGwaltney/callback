Feature: Job status history
  As a job seeker
  I want to see when each job moved between statuses
  So that I know how long an application has been waiting and how it got where it is

  Rule: A job's status changes are recorded with the date they happened

    @status:implemented @id:history-starts-when-added
    Scenario: A new job's history starts with the status it was added to
      Given today is 2026-10-09
      When I add a job for "Software Engineer" at "Acme" to the "Saved" column
      And I open the card
      Then the status history shows "Saved" on 2026-10-09

    @status:implemented @id:history-records-move
    Scenario: Moving a card to another status adds it to the history
      Given I added a job for "Software Engineer" at "Acme" to "Saved" on 2026-10-01
      And today is 2026-10-09
      When I move the card to "Applied"
      And I open the card
      Then the status history shows "Saved" on 2026-10-01, then "Applied" on 2026-10-09

    @status:implemented @id:history-oldest-first
    Scenario: The history lists every change from oldest to newest
      Given I added a job for "Software Engineer" at "Acme" to "Saved" on 2026-10-01
      And I moved it to "Applied" on 2026-10-02
      And I moved it to "Interviewing" on 2026-10-06
      And I moved it to "Offer" on 2026-10-09
      When I open the card
      Then the status history shows "Saved", "Applied", "Interviewing", and "Offer" in that order
      And each status shows the date the job moved to it

    @status:implemented @id:history-same-day-order
    Scenario: Several changes on the same day keep the order they happened in
      Given I added a job for "Software Engineer" at "Acme" to "Saved" on 2026-10-09
      And I moved it to "Applied" and then to "Interviewing" on 2026-10-09
      When I open the card
      Then the status history shows "Saved", "Applied", and "Interviewing" in that order

    @status:implemented @id:history-records-moving-back
    Scenario: Moving a card back to an earlier status is recorded too
      Given I have a job for "Software Engineer" at "Acme" that moved from "Applied" to "Interviewing"
      When I move the card back to "Applied"
      And I open the card
      Then the status history shows "Applied", "Interviewing", and "Applied" in that order

    @status:implemented @id:history-records-closing-reason
    Scenario: Closing a job records its closing reason in the history
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing"
      When I move the card to "Closed"
      And I choose the closing reason "Rejected"
      And I open the card
      Then the status history ends with "Closed" and the reason "Rejected"

    @status:implemented @id:history-keeps-original-closing-reason
    Scenario: Changing the closing reason later doesn't rewrite the history
      Given I closed a job for "Software Engineer" at "Acme" with the closing reason "No response"
      When I change the closing reason to "Rejected"
      And I open the card
      Then the card shows the closing reason "Rejected"
      But the status history still shows "Closed" with the reason "No response"

  Rule: Only real status changes are recorded

    @status:implemented @id:history-ignores-reorder
    Scenario: Reordering a card within its column doesn't change the history
      Given the "Saved" column holds the jobs at "Globex" and "Acme" in that order
      And the job at "Acme" has "Saved" as its only status history
      When I move the "Acme" card above the "Globex" card
      And I open the "Acme" card
      Then the status history still shows only "Saved"

    @status:implemented @id:history-ignores-cancelled-close
    Scenario: Cancelling a move to Closed doesn't change the history
      Given I have a job for "Software Engineer" at "Acme" whose history ends with "Interviewing"
      When I move the card to "Closed"
      But I cancel instead of choosing a closing reason
      And I open the card
      Then the status history still ends with "Interviewing"

    @status:implemented @id:history-ignores-date-applied
    Scenario: Changing the date applied doesn't change the history
      Given I moved a job for "Software Engineer" at "Acme" to "Applied" on 2026-10-09
      When I set the date applied to 2026-10-01
      And I open the card
      Then the status history still shows "Applied" on 2026-10-09

  Rule: The history is a record and can't be edited

    @status:implemented @id:history-read-only
    Scenario: The status history can't be edited
      Given I have a job for "Software Engineer" at "Acme" with a status history
      When I open the card
      Then I can see the status history
      But I can't change, add, or remove its entries

  Rule: The board card shows how long the job has been in its status

    @status:implemented @id:card-shows-days-in-status
    Scenario Outline: The card shows how many days the job has been in its current status
      Given I moved a job for "Software Engineer" at "Acme" to "Applied" on <moved>
      And today is 2026-10-09
      When I open the job search board
      Then the card shows "<shown>"

      Examples:
        | moved      | shown    |
        | 2026-10-09 | Today    |
        | 2026-10-08 | 1 day    |
        | 2026-10-01 | 8 days   |

    @status:implemented @id:days-in-status-from-added
    Scenario: A job that hasn't moved counts from when it was added
      Given I added a job for "Software Engineer" at "Acme" to "Saved" on 2026-10-06
      And today is 2026-10-09
      When I open the job search board
      Then the card shows "3 days"

    @status:implemented @id:days-in-status-resets-on-move
    Scenario: Moving a card to another status starts the count again
      Given I moved a job for "Software Engineer" at "Acme" to "Applied" on 2026-10-01
      And today is 2026-10-09
      When I move the card to "Interviewing"
      Then the card shows "Today"

    @status:implemented @id:days-in-status-ignores-reorder
    Scenario: Reordering a card within its column doesn't reset the count
      Given today is 2026-10-09
      And the "Applied" column holds the jobs at "Globex" and "Acme" in that order
      And the job at "Acme" moved to "Applied" on 2026-10-01
      When I move the "Acme" card above the "Globex" card
      Then the "Acme" card shows "8 days"

    @status:implemented @id:days-in-status-any-column
    Scenario Outline: Every column's cards show their days in status
      Given I moved a job for "Software Engineer" at "Acme" to "<status>" on 2026-10-01
      And today is 2026-10-09
      When I open the job search board
      Then the card shows "8 days"

      Examples:
        | status       |
        | Saved        |
        | Applied      |
        | Interviewing |
        | Offer        |
        | Closed       |

  Rule: The history is kept with the job

    @status:implemented @id:history-persists
    Scenario: The status history is the same after reopening the app
      Given I have a job for "Software Engineer" at "Acme" that moved from "Saved" to "Applied"
      When I close and reopen the app
      And I open the card
      Then the status history shows "Saved" and "Applied" with the same dates as before

    @status:implemented @id:history-deleted-with-job
    Scenario: Deleting a job deletes its status history
      Given I have a job for "Software Engineer" at "Acme" with a status history
      When I delete the job
      And I confirm the deletion
      Then its status history is removed too
