Feature: Job documents
  As a job seeker
  I want to attach the documents I used for each job
  So that I know exactly which resume and cover letter I sent

  Rule: Documents can be attached to a job

    @status:not-implemented @id:attach-document
    Scenario Outline: Attach a document of a given kind
      Given I have a job for "Software Engineer" at "Acme"
      When I attach the file "<file>" as a <kind>
      Then the job lists "<file>" as a <kind>

      Examples:
        | file                 | kind         |
        | resume-2026.pdf      | resume       |
        | acme-cover.docx      | cover letter |
        | portfolio-links.txt  | other        |

    @status:not-implemented @id:attach-multiple-documents
    Scenario: Attach several documents to one job
      Given I have a job for "Software Engineer" at "Acme"
      When I attach "resume-2026.pdf" as a resume
      And I attach "acme-cover.docx" as a cover letter
      And I attach "take-home-notes.md" as other
      Then the job lists all three documents with their kinds

    @status:not-implemented @id:card-shows-document-count
    Scenario: The card shows that documents are attached
      Given I have a job for "Software Engineer" at "Acme" with 2 attached documents
      When I look at the card on the board
      Then the card shows that it has 2 documents

    @status:not-implemented @id:document-kept-if-original-moved
    Scenario: An attached document still opens after the original file is moved
      Given I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
      And I have since moved or deleted the original "resume-2026.pdf" on my computer
      When I open the attached "resume-2026.pdf"
      Then the document opens with the content it had when I attached it

  Rule: Only text-like documents up to 10 MB can be attached

    @status:not-implemented @id:allowed-file-types
    Scenario Outline: Text-like document types can be attached
      Given I have a job for "Software Engineer" at "Acme"
      When I attach the file "<file>" as other
      Then the job lists "<file>" as other

      Examples:
        | file            |
        | notes.txt       |
        | resume.pdf      |
        | prep.md         |
        | cover.docx      |
        | cover.doc       |
        | resume.rtf      |
        | resume.odt      |

    @status:not-implemented @id:file-type-any-case
    Scenario: A document's type is recognized whatever the letter case of its extension
      Given I have a job for "Software Engineer" at "Acme"
      When I attach the file "RESUME.PDF" as other
      Then the job lists "RESUME.PDF" as other

    @status:not-implemented @id:disallowed-file-types
    Scenario Outline: Other file types can't be attached
      Given I have a job for "Software Engineer" at "Acme"
      When I try to attach the file "<file>"
      Then the file is not attached
      And I am told which document types are allowed

      Examples:
        | file            |
        | headshot.png    |
        | take-home.zip   |
        | setup.exe       |
        | salaries.xlsx   |

    @status:not-implemented @id:file-size-limit
    Scenario: A file larger than 10 MB can't be attached
      Given I have a job for "Software Engineer" at "Acme"
      When I try to attach a 12 MB file "portfolio.pdf"
      Then the file is not attached
      And I am told files can be at most 10 MB

    @status:not-implemented @id:file-size-at-limit
    Scenario: A file of exactly 10 MB can be attached
      Given I have a job for "Software Engineer" at "Acme"
      When I attach a 10 MB file "portfolio.pdf" as other
      Then the job lists "portfolio.pdf" as other

  Rule: Attached documents can be opened, relabeled, and removed

    @status:not-implemented @id:open-document
    Scenario: Open an attached document
      Given I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
      When I open the attached "resume-2026.pdf"
      Then it opens in my computer's default app for that file type

    @status:not-implemented @id:change-document-kind
    Scenario: Change what kind of document an attachment is
      Given I have attached "acme-cover.docx" as other to a job for "Software Engineer" at "Acme"
      When I change its kind to cover letter
      Then the job lists "acme-cover.docx" as a cover letter

    @status:not-implemented @id:remove-document
    Scenario: Remove an attached document
      Given I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
      When I remove the attached "resume-2026.pdf"
      Then the job no longer lists "resume-2026.pdf"
      And the original "resume-2026.pdf" on my computer is not deleted

    @status:not-implemented @id:same-document-many-jobs
    Scenario: The same file can be attached to more than one job
      Given I have a job for "Software Engineer" at "Acme" and a job for "Developer" at "Globex"
      When I attach "resume-2026.pdf" as a resume to both jobs
      Then both jobs list "resume-2026.pdf" as a resume
      And removing it from one job leaves it attached to the other
