Feature: Job next action
  As a job seeker
  I want to write down what I'm waiting for or need to do next on each job, and when
  So that I know which applications need my attention and when to follow up

  Rule: A job can have one next action, with an optional expected date

    @status:not-implemented @id:set-next-action
    Scenario: Set a next action with an expected date
      Given I have a job for "Software Engineer" at "Acme" with no next action
      When I set the next action to "Waiting for a call back from the recruiter" expected on 2026-10-14
      Then the job's next action is "Waiting for a call back from the recruiter"
      And its expected date is 2026-10-14

    @status:not-implemented @id:set-next-action-no-date
    Scenario: Set a next action without an expected date
      Given I have a job for "Software Engineer" at "Acme" with no next action
      When I set the next action to "Send a thank-you email" with no expected date
      Then the job's next action is "Send a thank-you email"
      And it has no expected date

    @status:not-implemented @id:next-action-needs-text
    Scenario: A next action can't be saved without saying what it is
      Given I have a job for "Software Engineer" at "Acme" with no next action
      When I try to set a next action expected on 2026-10-14 with nothing but spaces written
      Then the next action is not saved
      And I am told to write what the next action is

    @status:not-implemented @id:next-action-past-date
    Scenario: The expected date can be in the past
      Given today is 2026-10-09
      And I have a job for "Software Engineer" at "Acme" with no next action
      When I set the next action to "Follow up on application" expected on 2026-10-05
      Then the job's next action is "Follow up on application"
      And its expected date is 2026-10-05

    @status:not-implemented @id:next-action-any-status
    Scenario Outline: A next action can be set on a job in any status
      Given I have a job for "Software Engineer" at "Acme" in "<status>"
      When I set the next action to "Check the posting is still open" expected on 2026-10-14
      Then the job's next action is "Check the posting is still open"
      And the card stays in the "<status>" column

      Examples:
        | status       |
        | Saved        |
        | Applied      |
        | Interviewing |
        | Offer        |
        | Closed       |

  Rule: The next action can be changed and cleared

    @status:not-implemented @id:edit-next-action
    Scenario: Change the next action and its expected date
      Given I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
      When I change the next action to "Expecting an email with the offer details" expected on 2026-10-16
      Then the job's next action is "Expecting an email with the offer details"
      And its expected date is 2026-10-16

    @status:not-implemented @id:clear-next-action-date
    Scenario: Clear only the expected date
      Given I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
      When I clear the expected date
      Then the job's next action is still "Waiting for a call back"
      And it has no expected date

    @status:not-implemented @id:clear-next-action
    Scenario: Clear the next action once it's done
      Given I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
      When I clear the next action
      Then the job has no next action
      And the card no longer shows a next action

  Rule: The board card shows the next action and whether it's due

    @status:not-implemented @id:card-shows-next-action
    Scenario: The card shows the next action and its expected date
      Given today is 2026-10-09
      And I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
      When I open the job search board
      Then the card shows "Waiting for a call back" expected on 2026-10-14

    @status:not-implemented @id:card-next-action-no-date
    Scenario: The card shows a next action with no expected date
      Given I have a job for "Software Engineer" at "Acme" with the next action "Send a thank-you email" and no expected date
      When I open the job search board
      Then the card shows "Send a thank-you email" with no expected date
      And the next action is not marked as due or overdue

    @status:not-implemented @id:card-next-action-due-today
    Scenario: A next action expected today is marked as due
      Given today is 2026-10-14
      And I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
      When I open the job search board
      Then the card marks the next action as due today

    @status:not-implemented @id:card-next-action-overdue
    Scenario: A next action whose expected date has passed is marked as overdue
      Given today is 2026-10-15
      And I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
      When I open the job search board
      Then the card marks the next action as overdue

    @status:not-implemented @id:card-next-action-upcoming
    Scenario: A next action expected later isn't marked
      Given today is 2026-10-09
      And I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
      When I open the job search board
      Then the next action is not marked as due or overdue

    @status:not-implemented @id:closed-card-no-due-marks
    Scenario Outline: A closed job's next action is never marked as due or overdue
      Given today is 2026-10-14
      And I have a job for "Software Engineer" at "Acme" in "Closed" with the next action "Ask for feedback" expected on <expected>
      When I open the job search board
      Then the card shows "Ask for feedback"
      But the next action is not marked as due or overdue

      Examples:
        | expected   |
        | 2026-10-14 |
        | 2026-10-10 |

    @status:not-implemented @id:card-no-next-action
    Scenario: A job with no next action shows none on its card
      Given I have a job for "Software Engineer" at "Acme" with no next action
      When I open the job search board
      Then the card doesn't show a next action

    @status:not-implemented @id:view-next-action
    Scenario: See the next action when opening the card
      Given I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
      When I open the card
      Then I see the next action "Waiting for a call back" and its expected date 2026-10-14

  Rule: The board can be filtered to overdue or not overdue jobs

    @status:not-implemented @id:filter-overdue
    Scenario: Show only overdue jobs
      Given today is 2026-10-14
      And I have a job at "Acme" with a next action expected on 2026-10-10
      And a job at "Globex" with a next action expected on 2026-10-20
      And a job at "Initech" with no next action
      When I filter the board to overdue jobs
      Then only the card for "Acme" is shown

    @status:not-implemented @id:filter-not-overdue
    Scenario: Show only jobs that aren't overdue
      Given today is 2026-10-14
      And I have a job at "Acme" with a next action expected on 2026-10-10
      And a job at "Globex" with a next action expected on 2026-10-20
      And a job at "Initech" with no next action
      When I filter the board to jobs that aren't overdue
      Then the cards for "Globex" and "Initech" are shown
      And the card for "Acme" is not shown

    @status:not-implemented @id:filter-due-today-not-overdue
    Scenario: A job due today isn't overdue yet
      Given today is 2026-10-14
      And I have a job at "Acme" with a next action expected on 2026-10-14
      When I filter the board to overdue jobs
      Then the card for "Acme" is not shown

    @status:not-implemented @id:filter-closed-never-overdue
    Scenario: A closed job is never overdue
      Given today is 2026-10-14
      And I have a job at "Acme" in "Closed" with a next action expected on 2026-10-10
      When I filter the board to overdue jobs
      Then the card for "Acme" is not shown

    @status:not-implemented @id:filter-show-all
    Scenario: Show every job again
      Given today is 2026-10-14
      And I have a job at "Acme" with a next action expected on 2026-10-10
      And a job at "Globex" with no next action
      And I have filtered the board to overdue jobs
      When I filter the board to all jobs
      Then the cards for "Acme" and "Globex" are shown

    @status:not-implemented @id:filter-keeps-columns
    Scenario: Filtered cards stay in their own columns and the counts show how many are shown
      Given today is 2026-10-14
      And I have 2 overdue jobs in "Applied" and 1 overdue job in "Interviewing"
      And 3 jobs in "Applied" that aren't overdue
      When I filter the board to overdue jobs
      Then the "Applied" column shows 2 cards and a count of 2
      And the "Interviewing" column shows 1 card and a count of 1

    @status:not-implemented @id:filter-with-search
    Scenario: The filter and the search both apply
      Given today is 2026-10-14
      And I have an overdue job for "Software Engineer" at "Acme"
      And an overdue job for "Designer" at "Globex"
      And a job for "Software Engineer" at "Initech" that isn't overdue
      When I filter the board to overdue jobs
      And I search for "Engineer"
      Then only the card for "Acme" is shown

    @status:not-implemented @id:filter-no-matches
    Scenario: Told when no jobs match the filter
      Given today is 2026-10-14
      And none of my jobs are overdue
      When I filter the board to overdue jobs
      Then I am told that no jobs match

    @status:not-implemented @id:filter-resets-pages
    Scenario: Changing the filter returns every column to its first page
      Given the "Applied" column has 3 pages of cards
      And I am on its second page
      When I filter the board to jobs that aren't overdue
      Then the "Applied" column shows its first page

    @status:not-implemented @id:add-job-clears-filter
    Scenario: Adding a job shows every job again
      Given I have filtered the board to overdue jobs
      When I add a job for "Software Engineer" at "Acme" to the "Saved" column
      Then the board shows all jobs
      And the card for "Acme" is shown

    @status:not-implemented @id:filter-not-kept
    Scenario: The filter is cleared after reopening the app
      Given I have filtered the board to overdue jobs
      When I close and reopen the app
      Then the board shows all jobs

  Rule: The next action only changes when I change it

    @status:not-implemented @id:move-keeps-next-action
    Scenario Outline: Moving a card keeps its next action
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing" with the next action "Waiting for a call back" expected on 2026-10-14
      When I move the card to "<to>"
      Then the job's next action is still "Waiting for a call back"
      And its expected date is still 2026-10-14

      Examples:
        | to      |
        | Offer   |
        | Applied |
        | Closed  |

    @status:not-implemented @id:next-action-persists
    Scenario: The next action is the same after reopening the app
      Given I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
      When I close and reopen the app
      Then the job's next action is "Waiting for a call back"
      And its expected date is 2026-10-14
