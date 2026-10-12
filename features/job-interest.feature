Feature: Job interest rating
  As a job seeker
  I want to rate how interested I am in each job from one to five stars
  So that I can see at a glance which applications deserve the most effort

  Rule: A job can be rated from one to five stars

    @status:not-implemented @id:rate-interest
    Scenario Outline: Rate how interested I am in a job
      Given I have a job for "Software Engineer" at "Acme" with no interest rating
      When I answer "How interested are you in this job?" with <stars> stars
      Then the job's interest rating is <stars> stars

      Examples:
        | stars |
        | 1     |
        | 2     |
        | 3     |
        | 4     |
        | 5     |

    @status:not-implemented @id:rate-interest-when-adding
    Scenario: Rate my interest while adding a job
      Given I open the job search board
      When I add a job with company "Acme" and role "Software Engineer"
      And I answer "How interested are you in this job?" with 4 stars
      Then a card for "Software Engineer" at "Acme" appears with an interest rating of 4 stars

    @status:not-implemented @id:unrated-by-default
    Scenario: A job has no interest rating until I give it one
      Given I open the job search board
      When I add a job with company "Acme" and role "Software Engineer" and nothing else
      Then the job has no interest rating

    @status:not-implemented @id:interest-rating-range
    Scenario Outline: An interest rating must be from one to five stars
      Given I have a job for "Software Engineer" at "Acme" with an interest rating of 3 stars
      When I try to set the interest rating to <stars> stars
      Then the interest rating is not saved
      And the job's interest rating is still 3 stars
      And I am told the rating must be from 1 to 5 stars

      Examples:
        | stars |
        | 0     |
        | 6     |

    @status:not-implemented @id:interest-any-status
    Scenario Outline: A job in any status can be rated
      Given I have a job for "Software Engineer" at "Acme" in "<status>"
      When I answer "How interested are you in this job?" with 4 stars
      Then the job's interest rating is 4 stars

      Examples:
        | status       |
        | Saved        |
        | Applied      |
        | Interviewing |
        | Offer        |
        | Closed       |

  Rule: The interest rating can be changed and cleared

    @status:not-implemented @id:change-interest
    Scenario: Change my interest rating
      Given I have a job for "Software Engineer" at "Acme" with an interest rating of 2 stars
      When I change the interest rating to 5 stars
      Then the job's interest rating is 5 stars

    @status:not-implemented @id:clear-interest
    Scenario: Clear my interest rating
      Given I have a job for "Software Engineer" at "Acme" with an interest rating of 3 stars
      When I clear the interest rating
      Then the job has no interest rating

  Rule: The interest rating is shown on the card

    @status:not-implemented @id:card-shows-interest
    Scenario: The card shows the interest rating as stars
      Given I have a job for "Software Engineer" at "Acme" with an interest rating of 4 stars
      When I open the job search board
      Then the card shows 4 of 5 stars

    @status:not-implemented @id:card-unrated
    Scenario: An unrated card shows no stars
      Given I have a job for "Software Engineer" at "Acme" with no interest rating
      When I open the job search board
      Then the card doesn't show an interest rating

    @status:not-implemented @id:view-interest
    Scenario: See the interest rating when opening the card
      Given I have a job for "Software Engineer" at "Acme" with an interest rating of 4 stars
      When I open the card
      Then I see "How interested are you in this job?" answered with 4 stars

  Rule: The interest rating only changes when I change it

    @status:not-implemented @id:rating-doesnt-reorder
    Scenario: Rating a job doesn't change the order of its column
      Given the "Saved" column holds the jobs at "Acme" and "Globex" in that order
      When I give the job at "Globex" an interest rating of 5 stars
      Then the "Saved" column still shows "Acme" and "Globex" in that order

    @status:not-implemented @id:move-keeps-interest
    Scenario: Moving a card keeps its interest rating
      Given I have a job for "Software Engineer" at "Acme" in "Applied" with an interest rating of 4 stars
      When I move the card to "Interviewing"
      Then the job's interest rating is still 4 stars

    @status:not-implemented @id:interest-persists
    Scenario: The interest rating is the same after reopening the app
      Given I have a job for "Software Engineer" at "Acme" with an interest rating of 4 stars
      When I close and reopen the app
      Then the card shows 4 of 5 stars
