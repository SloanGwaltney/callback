Feature: Board search
  As a job seeker
  I want to search the board by job title or company
  So that I can find a job quickly when the board gets crowded

  Rule: Search matches the role and the company

    @status:implemented @id:search-by-role
    Scenario: Search by role
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      And I have a job for "Product Designer" at "Globex" in "Applied"
      When I search the board for "Engineer"
      Then I see the card for "Software Engineer" at "Acme"
      And I don't see the card for "Product Designer" at "Globex"

    @status:implemented @id:search-by-company
    Scenario: Search by company
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      And I have a job for "Product Designer" at "Globex" in "Applied"
      When I search the board for "Globex"
      Then I see the card for "Product Designer" at "Globex"
      And I don't see the card for "Software Engineer" at "Acme"

    @status:implemented @id:search-partial-any-case
    Scenario Outline: Search matches part of a word in any letter case
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      When I search the board for "<text>"
      Then I see the card for "Software Engineer" at "Acme"

      Examples:
        | text     |
        | acme     |
        | ACM      |
        | soft     |
        | ENGINEER |

    @status:implemented @id:search-ignores-other-details
    Scenario: Search doesn't match a job's other details
      Given I have a job for "Software Engineer" at "Globex" with the notes "Heard about it from someone at Acme."
      When I search the board for "Acme"
      Then I don't see the card for "Software Engineer" at "Globex"

    @status:implemented @id:search-blank-shows-all
    Scenario: Searching for only spaces shows every job
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      And I have a job for "Product Designer" at "Globex" in "Applied"
      When I search the board for "   "
      Then I see the card for "Software Engineer" at "Acme"
      And I see the card for "Product Designer" at "Globex"

  Rule: Search filters every column at once and keeps cards in their columns

    @status:implemented @id:search-keeps-columns
    Scenario: Matching cards stay in their own columns
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      And I have a job for "Data Engineer" at "Globex" in "Interviewing"
      When I search the board for "Engineer"
      Then the card for "Software Engineer" at "Acme" is in the "Saved" column
      And the card for "Data Engineer" at "Globex" is in the "Interviewing" column
      And I still see every status column

    @status:implemented @id:search-counts-matches
    Scenario: Column counts show how many cards match
      Given I have 3 jobs in "Saved", 2 of them at "Acme"
      And I have 1 job in "Applied" at "Globex"
      When I search the board for "Acme"
      Then the "Saved" column shows a count of 2
      And the "Applied" column shows a count of 0

    @status:implemented @id:search-no-matches
    Scenario: Told when nothing matches
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      When I search the board for "Initech"
      Then every status column is empty
      And I am told no jobs match "Initech"
      But I am not prompted to add my first job

    @status:implemented @id:clear-search
    Scenario: Clearing the search shows every job again
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      And I have a job for "Product Designer" at "Globex" in "Applied"
      And I have searched the board for "Acme"
      When I clear the search
      Then I see the card for "Software Engineer" at "Acme"
      And I see the card for "Product Designer" at "Globex"

  Rule: Cards can be moved to another status but not reordered while searching

    @status:implemented @id:search-move-card
    Scenario: Move a matching card to another status while searching
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      And I have searched the board for "Acme"
      When I move the card to "Applied"
      Then the card appears in the "Applied" column
      And the search for "Acme" is still applied

    @status:implemented @id:search-move-to-top
    Scenario: A card moved to another status while searching goes to the top of that column
      Given the "Applied" column holds the jobs at "Globex" and "Initech" in that order
      And I have a job at "Acme" in "Saved"
      And I have searched the board for "Acme"
      When I move the "Acme" card to "Applied"
      And I clear the search
      Then the "Applied" column shows "Acme", "Globex", and "Initech" in that order

    @status:implemented @id:search-no-reorder
    Scenario: Cards can't be reordered within a column while searching
      Given the "Saved" column holds the jobs at "Acme", "Globex", and "Acme Robotics" in that order
      And I have searched the board for "Acme"
      When I try to move the "Acme Robotics" card above the "Acme" card
      Then the "Acme Robotics" card stays where it was
      And I am told cards can't be reordered while searching
      And after I clear the search the "Saved" column shows "Acme", "Globex", and "Acme Robotics" in that order

  Rule: Search lasts only for the current session

    @status:implemented @id:search-not-kept
    Scenario: Search is cleared after reopening the app
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      And I have a job for "Product Designer" at "Globex" in "Applied"
      And I have searched the board for "Acme"
      When I close and reopen the app
      Then the search is empty
      And I see the card for "Product Designer" at "Globex"

  Rule: Adding a job clears the search

    @status:implemented @id:add-job-clears-search
    Scenario: Adding a job clears the search
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      And I have searched the board for "Acme"
      When I add a job at "Globex" to the "Saved" column
      Then the search is empty
      And I see the card for "Software Engineer" at "Acme"
      And I see the card for the job at "Globex"
