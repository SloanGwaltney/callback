Feature: Agent command line for a job's contacts, interviews, and documents
  As a job seeker working through my own AI agent
  I want my agent to keep track of the people, interviews, and documents for each job
  So that my agent can help me prepare and follow up without me copying things in by hand

  The rules for contacts, interviews, and attached documents are the same as in the app; see
  job-contacts.feature, job-interviews.feature, and job-documents.feature.

  Rule: My agent can add, change, and remove a job's contacts

    @status:implemented @id:cli-add-contact
    Scenario: Add a contact to a job
      Given I have a job at "Acme"
      When my agent adds "Jane Doe", a recruiter, with an email and phone number, to the job
      Then the job's contacts include "Jane Doe" as its recruiter
      And my agent is shown the contact with its ID

    @status:implemented @id:cli-add-contact-rules
    Scenario Outline: Adding a contact follows the app's rules
      Given I have a job at "Acme"
      When my agent tries to add a contact <problem>
      Then the job has no new contact
      And my agent is told <reason>

      Examples:
        | problem                                       | reason                                 |
        | with no name                                  | the contact needs a name               |
        | with no relationship                          | the relationships it can use           |
        | with the relationship "friend"                | the relationships it can use           |
        | with an email that isn't an email address     | the email must be an email address     |
        | with a profile link that isn't a web address  | the profile link must be a web address |

    @status:implemented @id:cli-update-contact
    Scenario: Change one of a contact's details and keep the rest
      Given "Jane Doe" is a contact on the job at "Acme" with an email and notes
      When my agent changes the contact's phone number
      Then the contact's phone number is the new one
      And the contact's email and notes are unchanged

    @status:implemented @id:cli-remove-contact
    Scenario: Remove a contact
      Given "Jane Doe" is a contact on the job at "Acme"
      When my agent removes the contact
      Then the job's contacts don't include "Jane Doe"

    @status:implemented @id:cli-unknown-contact
    Scenario: My agent is told when there's no contact with an ID
      When my agent tries to remove the contact with ID 999
      Then my agent is told there's no contact with ID 999

  Rule: My agent can add, change, and remove a job's interviews

    @status:implemented @id:cli-add-interview
    Scenario: Add an interview to a job
      Given I have a job at "Acme"
      When my agent adds a technical interview on 2026-10-20 at 14:00 to the job
      Then the job's interviews include a technical interview on 2026-10-20 at 14:00
      And my agent is shown the interview with its ID

    @status:implemented @id:cli-add-interview-rules
    Scenario Outline: Adding an interview follows the app's rules
      Given I have a job at "Acme"
      When my agent tries to add an interview <problem>
      Then the job has no new interview
      And my agent is told <reason>

      Examples:
        | problem                           | reason                                  |
        | with no date                      | the interview needs a date              |
        | on "next Tuesday"                 | dates are written as year, month, day   |
        | with no kind                      | the interview kinds it can use          |
        | of the kind "coffee"              | the interview kinds it can use          |

    @status:implemented @id:cli-update-interview
    Scenario: Change an interview's time and keep the rest
      Given the job at "Acme" has a technical interview on 2026-10-20 at 14:00 with notes
      When my agent changes the interview's time to 15:30
      Then the interview is on 2026-10-20 at 15:30
      And its kind and notes are unchanged

    @status:implemented @id:cli-remove-interview
    Scenario: Remove an interview
      Given the job at "Acme" has a technical interview on 2026-10-20
      When my agent removes the interview
      Then the job has no interviews

  Rule: My agent can attach, relabel, and remove a job's documents

    @status:implemented @id:cli-attach-document
    Scenario: Attach a document from a file on my computer
      Given I have a job at "Acme"
      And there is a file "acme-resume.pdf" on my computer
      When my agent attaches "acme-resume.pdf" to the job as a resume
      Then the job lists "acme-resume.pdf" as a resume
      And my agent is shown the document with its ID and where its stored copy is

    @status:implemented @id:cli-attach-document-rules
    Scenario Outline: Attaching a document follows the app's rules
      Given I have a job at "Acme"
      When my agent tries to attach <file>
      Then the job has no new document
      And my agent is told <reason>

      Examples:
        | file                             | reason                                  |
        | a file that doesn't exist        | there's no file at that path            |
        | "photo.png"                      | the file types it can attach            |
        | a PDF larger than 10 MB          | a document can be at most 10 MB         |
        | "notes.txt" as a "portfolio"     | the document kinds it can use           |

    @status:implemented @id:cli-relabel-document
    Scenario: Change what kind of document an attachment is
      Given the job at "Acme" has "acme-letter.pdf" attached as other
      When my agent marks "acme-letter.pdf" as a cover letter
      Then the job lists "acme-letter.pdf" as a cover letter

    @status:implemented @id:cli-remove-document
    Scenario: Remove an attached document
      Given the job at "Acme" has "acme-resume.pdf" attached
      When my agent removes the document
      Then the job doesn't list "acme-resume.pdf"
