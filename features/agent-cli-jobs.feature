Feature: Agent command line for jobs
  As a job seeker working through my own AI agent
  I want my agent to find, read, add, change, move, and delete my jobs
  So that my agent can keep my board up to date and help me decide where to put my effort

  The rules for a job's details, moving it, closing it, and its next action are the same as in
  the app; see job-cards.feature, job-board.feature, job-interest.feature, and
  job-next-action.feature.

  Rule: My agent lists my jobs one line each, in board order

    @status:not-implemented @id:cli-list-jobs
    Scenario: List my jobs
      Given I have a job for "Software Engineer" at "Acme" in "Applied"
      And it has been in "Applied" for 9 days
      And I rated it 4 stars
      And its next action is "Follow up" expected on 2026-10-15
      When my agent lists my jobs
      Then it sees one line for the job
      And the line shows its ID, "Applied", 9 days in status, "Acme", "Software Engineer", 4 stars, and "Follow up" expected on 2026-10-15

    @status:not-implemented @id:cli-list-board-order
    Scenario: Jobs are listed column by column, in each column's order
      Given the "Saved" column holds the jobs at "Initech" and "Acme" in that order
      And the "Applied" column holds the job at "Globex"
      When my agent lists my jobs
      Then it sees "Initech", "Acme", and "Globex" in that order

    @status:not-implemented @id:cli-list-leaves-out-long-text
    Scenario: The list leaves out each job's description and notes
      Given I have a job for "Software Engineer" at "Acme" with a description and notes
      When my agent lists my jobs
      Then the line for the job doesn't show its description or notes

    @status:not-implemented @id:cli-list-overdue-marked
    Scenario: The list marks a next action that's due or overdue
      Given today is 2026-10-14
      And I have a job at "Acme" whose next action was expected on 2026-10-10
      And I have a job at "Globex" whose next action is expected today
      When my agent lists my jobs
      Then the line for "Acme" marks its next action as overdue
      And the line for "Globex" marks its next action as due

    @status:not-implemented @id:cli-list-empty
    Scenario: My agent is told when no jobs are listed
      Given I have no jobs
      When my agent lists my jobs
      Then my agent is told there are no jobs to list

  Rule: Closed jobs are left out unless my agent asks for them

    @status:not-implemented @id:cli-list-leaves-out-closed
    Scenario: The list leaves out closed jobs
      Given I have a job at "Acme" in "Applied"
      And I have a job at "Globex" in "Closed"
      When my agent lists my jobs
      Then it sees "Acme"
      But it doesn't see "Globex"

    @status:not-implemented @id:cli-list-all
    Scenario: My agent asks for every job, closed ones too
      Given I have a job at "Acme" in "Applied"
      And I have a job at "Globex" in "Closed"
      When my agent lists all my jobs
      Then it sees "Acme" and "Globex"

  Rule: My agent can narrow the list

    @status:not-implemented @id:cli-filter-status
    Scenario: List only the jobs in some statuses
      Given I have jobs at "Acme" in "Saved", "Globex" in "Applied", and "Initech" in "Interviewing"
      When my agent lists the jobs in "Applied" and "Interviewing"
      Then it sees "Globex" and "Initech"
      But it doesn't see "Acme"

    @status:not-implemented @id:cli-filter-closed-status
    Scenario: Asking for Closed lists closed jobs
      Given I have a job at "Globex" in "Closed"
      When my agent lists the jobs in "Closed"
      Then it sees "Globex"

    @status:not-implemented @id:cli-filter-search
    Scenario: Search the list the way the board's search does
      Given I have jobs for "Software Engineer" at "Acme" and "Designer" at "Globex"
      When my agent lists the jobs matching "engineer"
      Then it sees "Acme"
      But it doesn't see "Globex"

    @status:not-implemented @id:cli-filter-overdue
    Scenario: List only the jobs whose next action is overdue
      Given today is 2026-10-14
      And I have a job at "Acme" whose next action was expected on 2026-10-10
      And I have a job at "Globex" whose next action is expected today
      When my agent lists the overdue jobs
      Then it sees "Acme"
      But it doesn't see "Globex"

    @status:not-implemented @id:cli-filter-stale
    Scenario: List only the jobs that have been in their status for some days
      Given I have a job at "Acme" that has been in "Applied" for 20 days
      And I have a job at "Globex" that has been in "Applied" for 3 days
      When my agent lists the jobs that have been in their status for at least 14 days
      Then it sees "Acme"
      But it doesn't see "Globex"

    @status:not-implemented @id:cli-filter-interest
    Scenario: List only the jobs rated at least some stars
      Given I have a job at "Acme" rated 5 stars
      And I have a job at "Globex" rated 3 stars
      And I have a job at "Initech" with no rating
      When my agent lists the jobs rated at least 4 stars
      Then it sees "Acme"
      But it doesn't see "Globex" or "Initech"

    @status:not-implemented @id:cli-filter-combined
    Scenario: Every filter my agent gives applies
      Given I have a job for "Software Engineer" at "Acme" in "Applied" rated 5 stars
      And I have a job for "Software Engineer" at "Globex" in "Saved" rated 5 stars
      And I have a job for "Designer" at "Initech" in "Applied" rated 5 stars
      When my agent lists the jobs in "Applied" matching "engineer"
      Then it sees "Acme"
      But it doesn't see "Globex" or "Initech"

    @status:not-implemented @id:cli-filter-no-match
    Scenario: My agent is told when no jobs match
      Given I have a job for "Software Engineer" at "Acme"
      When my agent lists the jobs matching "pilot"
      Then my agent is told no jobs match

  Rule: My agent can see everything about one job

    @status:not-implemented @id:cli-show-job
    Scenario: Show a job's full details
      Given I have a job for "Software Engineer" at "Acme" with every detail filled in
      And it has a contact, an interview, a next action, and an attached document
      And it has moved from "Saved" to "Applied"
      When my agent shows the job
      Then it sees every detail of the job, including its description, notes, and posting link
      And it sees its contacts, its interviews, its next action, and its status history
      And it sees each attached document with its kind and where its stored copy is

    @status:not-implemented @id:cli-show-missing-document
    Scenario: Showing a job marks a document whose file is missing
      Given I have a job at "Acme" with "resume-2026.pdf" attached
      And the stored copy of "resume-2026.pdf" is gone
      When my agent shows the job
      Then "resume-2026.pdf" is marked as missing its file

    @status:not-implemented @id:cli-no-opening
    Scenario: My agent can't open anything on my screen
      Given I have a job at "Acme" with a posting link and an attached document
      When my agent looks at what it can do with the job
      Then it can't open the posting link, a contact's profile link, or a document on my screen
      But it can read the links and where the documents' stored copies are

  Rule: My agent can add a job

    @status:not-implemented @id:cli-add-job
    Scenario: Add a job with its details
      When my agent adds a job for "Software Engineer" at "Acme" with a salary, a location, and a posting link
      Then the board has a job for "Software Engineer" at "Acme" at the top of "Saved"
      And my agent is shown the job with its ID

    @status:not-implemented @id:cli-add-job-status
    Scenario: Add a job straight to a status
      When my agent adds a job for "Software Engineer" at "Acme" to "Applied"
      Then the job is at the top of the "Applied" column

    @status:not-implemented @id:cli-add-job-rules
    Scenario Outline: Adding a job follows the app's rules
      When my agent tries to add a job <problem>
      Then no job is added
      And my agent is told <reason>

      Examples:
        | problem                                         | reason                                |
        | with no company                                 | the job needs a company               |
        | with no role                                    | the job needs a role                  |
        | to "Closed"                                     | a job can't be added straight to Closed |
        | with a posting link that isn't a web address    | the link must be a web address        |
        | rated 6 stars                                   | a rating must be from 1 to 5 stars    |

  Rule: My agent changes only the details it names

    @status:not-implemented @id:cli-update-job
    Scenario: Change one detail and keep the rest
      Given I have a job for "Software Engineer" at "Acme" with notes and a salary of "$120k"
      When my agent changes the job's salary to "$130k"
      Then the job's salary is "$130k"
      And the job's notes are unchanged

    @status:not-implemented @id:cli-clear-job-detail
    Scenario Outline: Clear a detail
      Given I have a job at "Acme" with <detail> set
      When my agent clears the job's <detail>
      Then the job has no <detail>

      Examples:
        | detail          |
        | salary          |
        | notes           |
        | date applied    |
        | interest rating |

  Rule: My agent can move a job

    @status:not-implemented @id:cli-move-top
    Scenario: A moved job goes to the top of its new column
      Given the "Applied" column holds the jobs at "Initech" and "Globex"
      And I have a job at "Acme" in "Saved"
      When my agent moves the job at "Acme" to "Applied"
      Then the "Applied" column holds "Acme", "Initech", and "Globex" in that order

    @status:not-implemented @id:cli-move-bottom
    Scenario: Move a job to the bottom of a column
      Given the "Applied" column holds the jobs at "Initech" and "Globex"
      And I have a job at "Acme" in "Saved"
      When my agent moves the job at "Acme" to the bottom of "Applied"
      Then the "Applied" column holds "Initech", "Globex", and "Acme" in that order

    @status:not-implemented @id:cli-move-position
    Scenario: Move a job to a chosen place in a column
      Given the "Applied" column holds the jobs at "Initech" and "Globex"
      And I have a job at "Acme" in "Saved"
      When my agent moves the job at "Acme" to second place in "Applied"
      Then the "Applied" column holds "Initech", "Acme", and "Globex" in that order

    @status:not-implemented @id:cli-move-follows-rules
    Scenario: Moving a job follows the app's rules
      Given I have a job at "Acme" in "Saved" with no date applied
      And today is 2026-10-14
      When my agent moves the job to "Applied"
      Then the job's date applied is 2026-10-14
      And the move is in the job's status history

    @status:not-implemented @id:cli-move-to-closed-needs-close
    Scenario: Moving a job to Closed points my agent to closing it
      Given I have a job at "Acme" in "Applied"
      When my agent tries to move the job to "Closed"
      Then the job doesn't move
      And my agent is told to close the job with a reason instead

  Rule: My agent closes a job with a reason

    @status:not-implemented @id:cli-close-job
    Scenario Outline: Close a job with a reason
      Given I have a job at "Acme" in "Applied"
      When my agent closes the job because "<reason>"
      Then the job is in "Closed" with the reason "<reason>"

      Examples:
        | reason          |
        | Rejected        |
        | Withdrew        |
        | No response     |
        | Declined offer  |
        | Position filled |

    @status:not-implemented @id:cli-close-other
    Scenario: Close a job with my own reason
      Given I have a job at "Acme" in "Applied"
      When my agent closes the job with the reason "Other" and the text "Relocated"
      Then the job is in "Closed" with the reason "Relocated"

    @status:not-implemented @id:cli-close-needs-reason
    Scenario Outline: Closing a job needs a reason
      Given I have a job at "Acme" in "Applied"
      When my agent tries to close the job <problem>
      Then the job is still in "Applied"
      And my agent is told <reason>

      Examples:
        | problem                                 | reason                                       |
        | without a reason                        | the reasons it can use                       |
        | with the reason "Other" and no text     | the "Other" reason needs its text            |

    @status:not-implemented @id:cli-change-closing-reason
    Scenario: Change a closed job's reason
      Given I have a job at "Acme" in "Closed" with the reason "No response"
      When my agent closes the job because "Rejected"
      Then the job is in "Closed" with the reason "Rejected"

  Rule: My agent can delete a job

    @status:not-implemented @id:cli-delete-job
    Scenario: Delete a job
      Given I have a job at "Acme" with a contact, an interview, and an attached document
      When my agent deletes the job
      Then the board has no job at "Acme"
      And its contacts, interviews, attached documents, and status history are deleted with it

  Rule: My agent can set and clear a job's next action

    @status:not-implemented @id:cli-set-next-action
    Scenario: Set a next action with an expected date
      Given I have a job at "Acme"
      When my agent sets the job's next action to "Send thank-you note" expected on 2026-10-20
      Then the job's next action is "Send thank-you note" expected on 2026-10-20

    @status:not-implemented @id:cli-set-next-action-no-date
    Scenario: Set a next action without an expected date
      Given I have a job at "Acme"
      When my agent sets the job's next action to "Research the team"
      Then the job's next action is "Research the team" with no expected date

    @status:not-implemented @id:cli-clear-next-action-date
    Scenario: Clear only the expected date
      Given I have a job at "Acme" whose next action is "Follow up" expected on 2026-10-20
      When my agent clears the next action's expected date
      Then the job's next action is "Follow up" with no expected date

    @status:not-implemented @id:cli-clear-next-action
    Scenario: Clear the next action
      Given I have a job at "Acme" whose next action is "Follow up"
      When my agent clears the job's next action
      Then the job has no next action

    @status:not-implemented @id:cli-next-action-needs-text
    Scenario: A next action needs to say what it is
      Given I have a job at "Acme"
      When my agent tries to set the job's next action to nothing
      Then the job has no next action
      And my agent is told the next action needs to say what it is
