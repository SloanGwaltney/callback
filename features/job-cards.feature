Feature: Job cards
  As a job seeker
  I want to record the details of each job I'm tracking
  So that everything about an application is in one place

  Rule: A job needs a company and a role

    @status:implemented @id:add-job
    Scenario: Add a job with all its details
      Given I open the job search board
      When I add a job with:
        | company         | Acme                                |
        | role            | Software Engineer                   |
        | job description | Build and maintain internal tools.  |
        | link            | https://acme.example.com/jobs/123   |
        | salary          | $120k - $140k                       |
        | location        | Remote (US)                         |
        | notes           | Referred by Dana from the meetup.   |
      Then a card for "Software Engineer" at "Acme" appears in the "Saved" column
      And the card shows the company, the role, and the location

    @status:implemented @id:add-job-minimum
    Scenario: Add a job with only a company and a role
      Given I open the job search board
      When I add a job with company "Acme" and role "Software Engineer" and nothing else
      Then a card for "Software Engineer" at "Acme" appears in the "Saved" column

    @status:implemented @id:add-job-in-column
    Scenario: Add a job directly to a status column
      Given I open the job search board
      When I add a job with company "Acme" and role "Software Engineer" to the "Applied" column
      Then a card for "Software Engineer" at "Acme" appears in the "Applied" column

    @status:implemented @id:add-job-not-closed
    Scenario: A job can't be added straight to Closed
      Given I open the job search board
      When I try to add a job with company "Acme" and role "Software Engineer" to the "Closed" column
      Then the job is not added
      And I am told to add the job to another column and move it to Closed

    @status:implemented @id:add-job-missing-required
    Scenario Outline: A job can't be saved without a company or a role
      Given I open the job search board
      When I try to add a job with company "<company>" and role "<role>"
      Then the job is not added
      And I am told that the <missing> is required

      Examples:
        | company | role              | missing |
        |         | Software Engineer | company |
        | Acme    |                   | role    |

  Rule: Job details can be viewed and edited

    @status:implemented @id:view-job-details
    Scenario: View a job's full details
      Given I have a job for "Software Engineer" at "Acme" with every detail filled in
      When I open the card
      Then I see the company, role, job description, link, salary, location, notes, and date applied
      And I see the documents attached to the job

    @status:implemented @id:edit-job-details
    Scenario: Edit a job's details
      Given I have a job for "Software Engineer" at "Acme"
      When I change the role to "Senior Software Engineer"
      And I change the job description
      Then the card shows the role "Senior Software Engineer"
      And the job shows the new job description

    @status:implemented @id:salary-free-text
    Scenario Outline: Salary is written however the posting gives it
      Given I have a job for "Software Engineer" at "Acme"
      When I set the salary to "<salary>"
      Then the job shows the salary "<salary>"

      Examples:
        | salary            |
        | $120,000          |
        | $120k - $140k     |
        | $65/hr            |
        | Not listed        |

    @status:implemented @id:add-notes
    Scenario: Keep notes on a job
      Given I have a job for "Software Engineer" at "Acme"
      When I write in the notes "Recruiter call went well, follow up Friday."
      Then the job's notes show "Recruiter call went well, follow up Friday."

    @status:implemented @id:edit-notes
    Scenario: Add to existing notes
      Given I have a job for "Software Engineer" at "Acme" with the notes "Referred by Dana."
      When I add "Phone screen booked for 10/14." on a new line in the notes
      Then the job's notes show both lines

    @status:implemented @id:open-job-link
    Scenario: Open a job's posting link
      Given I have a job for "Software Engineer" at "Acme" with the link "https://acme.example.com/jobs/123"
      When I open the job's link
      Then the posting opens in my web browser

    @status:implemented @id:invalid-job-link
    Scenario: A job link must be a web address
      Given I have a job for "Software Engineer" at "Acme"
      When I try to set the link to "not a url"
      Then the link is not saved
      And I am told the link must be a web address

  Rule: The date applied records when I applied

    @status:implemented @id:set-date-applied
    Scenario: Set the date applied
      Given I have a job for "Software Engineer" at "Acme" in "Applied"
      When I set the date applied to 2026-10-01
      Then the job's date applied is 2026-10-01
      And the card shows the date applied

    @status:implemented @id:date-applied-any-status
    Scenario Outline: The date applied can be set in any status
      Given I have a job for "Software Engineer" at "Acme" in "<status>"
      When I set the date applied to 2026-10-01
      Then the job's date applied is 2026-10-01

      Examples:
        | status       |
        | Saved        |
        | Interviewing |
        | Offer        |
        | Closed       |

    @status:implemented @id:clear-date-applied
    Scenario: Clear the date applied
      Given I have a job for "Software Engineer" at "Acme" with a date applied of 2026-10-01
      When I clear the date applied
      Then the job has no date applied

    @status:implemented @id:future-date-applied
    Scenario: The date applied can't be in the future
      Given today is 2026-10-09
      And I have a job for "Software Engineer" at "Acme"
      When I try to set the date applied to 2026-10-20
      Then the date applied is not saved
      And I am told the date applied can't be in the future

  Rule: Jobs can be deleted

    @status:implemented @id:delete-job
    Scenario: Delete a job after confirming
      Given I have a job for "Software Engineer" at "Acme"
      When I delete the job
      And I confirm the deletion
      Then the card for "Software Engineer" at "Acme" no longer appears on the board
      And its attached documents are removed too

    @status:implemented @id:cancel-delete-job
    Scenario: Cancel deleting a job
      Given I have a job for "Software Engineer" at "Acme"
      When I delete the job
      But I cancel the deletion
      Then the card for "Software Engineer" at "Acme" is still on the board
