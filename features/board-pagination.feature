Feature: Board pagination
  As a job seeker
  I want each board column to show its cards a page at a time
  So that the board stays readable and quick when I'm tracking many jobs

  # The page size is a system setting set in code, not something the user changes.
  Rule: Each column shows up to 10 cards per page

    @status:implemented @id:column-first-page
    Scenario: A column with more than 10 cards shows its first page
      Given the "Saved" column holds 25 jobs
      When I open the job search board
      Then the "Saved" column shows its first 10 cards in order
      And the "Saved" column shows that it is on page 1 of 3

    @status:implemented @id:column-fits-one-page
    Scenario: A column with 10 cards or fewer has no pages to move between
      Given the "Applied" column holds 10 jobs
      When I open the job search board
      Then the "Applied" column shows all 10 cards
      And the "Applied" column offers no other pages

    @status:implemented @id:count-shows-column-total
    Scenario: The column count shows every card in the column, not just the page
      Given the "Saved" column holds 25 jobs
      And the "Saved" column is on page 2
      Then the "Saved" column shows a count of 25

  Rule: Each column moves between its pages on its own

    @status:implemented @id:next-page
    Scenario: Go to a column's next page
      Given the "Saved" column holds 25 jobs
      And the "Saved" column is on page 1
      When I go to the next page of the "Saved" column
      Then the "Saved" column shows cards 11 to 20
      And the "Saved" column shows that it is on page 2 of 3

    @status:implemented @id:previous-page
    Scenario: Go to a column's previous page
      Given the "Saved" column holds 25 jobs
      And the "Saved" column is on page 3
      When I go to the previous page of the "Saved" column
      Then the "Saved" column shows cards 11 to 20
      And the "Saved" column shows that it is on page 2 of 3

    @status:implemented @id:page-bounds
    Scenario Outline: A column can't page past its first or last page
      Given the "Saved" column holds 25 jobs
      And the "Saved" column is on page <page>
      Then I can't go to the <direction> page of the "Saved" column

      Examples:
        | page | direction |
        | 1    | previous  |
        | 3    | next      |

    @status:implemented @id:columns-page-independently
    Scenario: Paging one column leaves the other columns where they are
      Given the "Saved" column holds 25 jobs
      And the "Applied" column holds 15 jobs
      When I go to the next page of the "Saved" column
      Then the "Saved" column is on page 2
      And the "Applied" column is still on page 1

  Rule: Each column pages with its own arrows, and changing page is animated

    @status:implemented @id:page-arrows
    Scenario: A column with more than one page shows previous and next arrows
      Given the "Saved" column holds 25 jobs
      When I open the job search board
      Then the "Saved" column shows a previous arrow and a next arrow
      And the previous arrow can't be used on page 1

    @status:implemented @id:page-transition
    Scenario Outline: Changing page slides the cards in the direction of the arrow
      Given the "Saved" column holds 25 jobs
      And the "Saved" column is on page 2
      When I use the <arrow> arrow of the "Saved" column
      Then the cards on page 2 slide out toward the <out> side of the column
      And the cards on page <page> slide in from the <in> side of the column
      And the other columns don't move

      Examples:
        | arrow    | out   | page | in    |
        | next     | left  | 3    | right |
        | previous | right | 1    | left  |

    @status:implemented @id:page-transition-reduced-motion
    Scenario: Changing page doesn't animate when my system asks for reduced motion
      Given my system is set to reduce motion
      And the "Saved" column holds 25 jobs
      When I use the next arrow of the "Saved" column
      Then the "Saved" column shows cards 11 to 20 without sliding

    @status:implemented @id:one-page-per-column
    Scenario: A column shows only one page of cards at a time
      Given the "Saved" column holds 25 jobs
      When I open the job search board
      Then the "Saved" column shows only the cards on its current page
      And cards on other pages of the "Saved" column can't be seen or moved until I page to them

  Rule: Moving and adding cards keeps the pages consistent

    @status:implemented @id:new-card-shows-first-page
    Scenario: Adding a job shows the first page of its column
      Given the "Saved" column holds 25 jobs
      And the "Saved" column is on page 2
      When I add a job at "Acme" to the "Saved" column
      Then the "Saved" column is on page 1
      And the card for "Acme" is first in the "Saved" column

    @status:implemented @id:move-card-onto-page
    Scenario: Place a moved card at a chosen position on a later page
      Given the "Applied" column holds 25 jobs
      And the "Applied" column is on page 2, showing the jobs at "Globex" and "Initech" next to each other
      And I have a job at "Acme" in "Saved"
      When I move the "Acme" card into "Applied" between "Globex" and "Initech"
      Then the "Applied" column shows "Globex", "Acme", and "Initech" in that order on page 2

    @status:implemented @id:last-card-leaves-page
    Scenario: Moving the only card off the last page shows the page before it
      Given the "Saved" column holds 21 jobs
      And the "Saved" column is on page 3, showing 1 card
      When I move that card to "Applied"
      Then the "Saved" column is on page 2 of 2

    @status:implemented @id:page-filled-after-move
    Scenario: A page fills back up after a card moves out of it
      Given the "Saved" column holds 25 jobs
      And the "Saved" column is on page 1
      When I move a card on page 1 to "Applied"
      Then the "Saved" column still shows 10 cards on page 1
      And the card that was first on page 2 is now last on page 1

  Rule: Search and pagination work together

    @status:implemented @id:search-results-paged
    Scenario: Matching cards are shown a page at a time
      Given the "Saved" column holds 15 jobs at "Acme" and 10 jobs at "Globex"
      When I search the board for "Acme"
      Then the "Saved" column shows that it is on page 1 of 2
      And the "Saved" column shows a count of 15

    @status:implemented @id:search-resets-pages
    Scenario: Changing the search returns every column to its first page
      Given the "Saved" column holds 25 jobs
      And the "Saved" column is on page 3
      When I search the board for "Acme"
      Then the "Saved" column is on page 1

  Rule: Pages aren't kept between sessions

    @status:implemented @id:pages-reset-on-reopen
    Scenario: Every column starts on its first page after reopening the app
      Given the "Saved" column holds 25 jobs
      And the "Saved" column is on page 2
      When I close and reopen the app
      Then the "Saved" column is on page 1
