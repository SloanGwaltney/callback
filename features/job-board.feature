Feature: Job search board
  As a job seeker
  I want my job applications laid out on a board by status
  So that I can see where every application stands at a glance

  Rule: The board has a fixed set of status columns

    @status:implemented @id:board-status-columns
    Scenario: Board shows the status columns in order
      Given I open the job search board
      Then I see the columns "Saved", "Applied", "Interviewing", "Offer", and "Closed" in that order

    @status:implemented @id:empty-board
    Scenario: Empty board when no jobs have been added
      Given I have not added any jobs
      When I open the job search board
      Then every status column is empty
      And I am prompted to add my first job

    @status:implemented @id:column-card-count
    Scenario: Each column shows how many cards it holds
      Given I have 3 jobs in "Saved" and 1 job in "Applied"
      When I open the job search board
      Then the "Saved" column shows a count of 3
      And the "Applied" column shows a count of 1
      And the "Interviewing" column shows a count of 0

  Rule: Cards can be moved between any statuses

    @status:implemented @id:move-card-forward
    Scenario: Move a card to the next status
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      When I move the card to "Applied"
      Then the card appears in the "Applied" column
      And the card no longer appears in the "Saved" column

    @status:implemented @id:move-card-any-status
    Scenario Outline: Move a card to any other status
      Given I have a job for "Software Engineer" at "Acme" in "<from>"
      When I move the card to "<to>"
      Then the card appears in the "<to>" column

      Examples:
        | from         | to           |
        | Saved        | Interviewing |
        | Applied      | Interviewing |
        | Interviewing | Offer        |
        | Offer        | Interviewing |
        | Closed       | Applied      |

    @status:implemented @id:move-keeps-details
    Scenario: Moving a card keeps its details and documents
      Given I have a job for "Software Engineer" at "Acme" in "Applied"
      And the job has a job description, a link, and an attached resume
      When I move the card to "Interviewing"
      Then the card still shows the company "Acme" and the role "Software Engineer"
      And the job description, link, and attached resume are unchanged

    @status:implemented @id:move-to-applied-sets-date
    Scenario: Moving a card to Applied fills in today's date when no date is set
      Given today is 2026-10-09
      And I have a job for "Software Engineer" at "Acme" in "Saved" with no date applied
      When I move the card to "Applied"
      Then the job's date applied is 2026-10-09

    @status:implemented @id:move-to-applied-keeps-date
    Scenario: Moving a card to Applied keeps a date applied that is already set
      Given today is 2026-10-09
      And I have a job for "Software Engineer" at "Acme" in "Interviewing" with a date applied of 2026-10-01
      When I move the card to "Applied"
      Then the job's date applied is still 2026-10-01

    @status:implemented @id:move-to-saved-clears-date
    Scenario Outline: Moving a card back to Saved clears the date applied
      Given I have a job for "Software Engineer" at "Acme" in "<from>" with a date applied of 2026-10-01
      When I move the card to "Saved"
      Then the job has no date applied

      Examples:
        | from         |
        | Applied      |
        | Interviewing |
        | Offer        |
        | Closed       |

    @status:implemented @id:reorder-saved-keeps-date
    Scenario: Reordering a card within Saved keeps its date applied
      Given the "Saved" column holds the jobs at "Globex" and "Acme" in that order
      And the job at "Acme" has a date applied of 2026-10-01
      When I move the "Acme" card above the "Globex" card
      Then the job's date applied is still 2026-10-01

    @status:implemented @id:reorder-applied-no-date
    Scenario: Reordering a card within Applied doesn't fill in a date applied
      Given the "Applied" column holds the jobs at "Globex" and "Acme" in that order
      And the job at "Acme" has no date applied
      When I move the "Acme" card above the "Globex" card
      Then the job has no date applied

  Rule: Closing a job records why it was closed

    @status:implemented @id:close-asks-reason
    Scenario Outline: Moving a card to Closed asks for a reason
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing"
      When I move the card to "Closed"
      And I choose the closing reason "<reason>"
      Then the card appears in the "Closed" column
      And the card shows the closing reason "<reason>"

      Examples:
        | reason            |
        | Rejected          |
        | Withdrew          |
        | No response       |
        | Declined offer    |
        | Position filled   |

    @status:implemented @id:close-other-reason
    Scenario: Write my own reason when closing with Other
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing"
      When I move the card to "Closed"
      And I choose the closing reason "Other" and write "Company went through layoffs"
      Then the card appears in the "Closed" column
      And the card shows the closing reason "Company went through layoffs"

    @status:implemented @id:close-other-needs-text
    Scenario: Closing with Other needs a written reason
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing"
      When I move the card to "Closed"
      And I choose the closing reason "Other" without writing a reason
      Then the card stays in the "Interviewing" column
      And I am told to write why the job was closed

    @status:implemented @id:close-cancel
    Scenario: Cancelling the closing reason leaves the card where it was
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing"
      When I move the card to "Closed"
      But I cancel instead of choosing a closing reason
      Then the card stays in the "Interviewing" column

    @status:implemented @id:change-closing-reason
    Scenario: Change the closing reason of a closed job
      Given I have a job for "Software Engineer" at "Acme" in "Closed" with the closing reason "No response"
      When I change the closing reason to "Rejected"
      Then the card shows the closing reason "Rejected"

    @status:implemented @id:reopen-clears-reason
    Scenario: Moving a card out of Closed clears the closing reason
      Given I have a job for "Software Engineer" at "Acme" in "Closed" with the closing reason "No response"
      When I move the card to "Interviewing"
      Then the card appears in the "Interviewing" column
      And the job has no closing reason

    @status:implemented @id:reorder-closed-keeps-reason
    Scenario: Reordering a card within Closed keeps its closing reason
      Given the "Closed" column holds the jobs at "Globex" and "Acme" in that order
      And the job at "Acme" has the closing reason "Rejected"
      When I move the "Acme" card above the "Globex" card
      Then the card shows the closing reason "Rejected"

  Rule: Cards can be ordered within a column

    @status:implemented @id:reorder-card
    Scenario: Reorder a card within its column
      Given the "Saved" column holds the jobs at "Acme", "Globex", and "Initech" in that order
      When I move the "Initech" card above the "Acme" card
      Then the "Saved" column shows "Initech", "Acme", and "Globex" in that order

    @status:implemented @id:move-card-to-position
    Scenario: Place a moved card at a chosen position in another column
      Given the "Applied" column holds the jobs at "Globex" and "Initech" in that order
      And I have a job at "Acme" in "Saved"
      When I move the "Acme" card into "Applied" between "Globex" and "Initech"
      Then the "Applied" column shows "Globex", "Acme", and "Initech" in that order

    @status:implemented @id:new-card-at-top
    Scenario: A newly added job goes to the top of its column
      Given the "Saved" column holds the jobs at "Globex" and "Initech" in that order
      When I add a job at "Acme" to the "Saved" column
      Then the "Saved" column shows "Acme", "Globex", and "Initech" in that order

  Rule: The board is kept between sessions

    @status:implemented @id:board-persists
    Scenario: Board is the same after reopening the app
      Given I have a job for "Software Engineer" at "Acme" in "Interviewing"
      When I close and reopen the app
      Then the card for "Software Engineer" at "Acme" is in the "Interviewing" column

    @status:implemented @id:card-order-persists
    Scenario: Card order is the same after reopening the app
      Given the "Saved" column holds the jobs at "Initech", "Acme", and "Globex" in that order
      When I close and reopen the app
      Then the "Saved" column shows "Initech", "Acme", and "Globex" in that order

  Rule: The board shows changes made outside the app

    @status:implemented @id:refresh-on-focus
    Scenario: The board shows my agent's changes when I come back to the app
      Given the app is open with a job for "Software Engineer" at "Acme" in "Applied"
      And I switch to another window
      And my agent moves the job to "Interviewing"
      When I switch back to the app
      Then the card for "Software Engineer" at "Acme" is in the "Interviewing" column

    @status:implemented @id:refresh-keeps-search-and-pages
    Scenario: Refreshing the board keeps my search and the page I'm on
      Given the app is open and I have searched for "Engineer"
      And the "Applied" column is on its second page
      And I switch to another window
      When I switch back to the app
      Then the search still shows "Engineer"
      And the "Applied" column is still on its second page
