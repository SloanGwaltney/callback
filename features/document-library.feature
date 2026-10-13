Feature: Document library
  As a job seeker
  I want one place to keep the documents I use across my job search
  So that my base resume, cover letter templates, and similar files are always at hand

  Rule: The library is its own page

    @status:implemented @id:open-library
    Scenario: Go from the board to the document library
      Given I open the job search board
      When I go to the document library
      Then I see the document library

    @status:implemented @id:back-to-board
    Scenario: Go back to the board from the document library
      Given I am in the document library
      When I go back to the board
      Then I see the job search board

    @status:implemented @id:empty-library
    Scenario: An empty library prompts me to add my first document
      Given I have not added any documents to the library
      When I go to the document library
      Then I am prompted to add my first document

  Rule: Documents can be added to the library with an optional description

    @status:implemented @id:add-library-document
    Scenario: Add a document with a description
      Given I am in the document library
      When I add the file "resume-base-2026.docx" with the description "Base resume, backend roles"
      Then the library lists "resume-base-2026.docx" with the description "Base resume, backend roles"

    @status:implemented @id:add-library-document-no-description
    Scenario: Add a document without a description
      Given I am in the document library
      When I add the file "cover-template.md" with no description
      Then the library lists "cover-template.md" with no description

    @status:implemented @id:library-newest-first
    Scenario: The newest documents are listed first
      Given I added "resume-base-2026.docx" to the library
      And I then added "cover-template.md"
      And I then added "portfolio-links.txt"
      When I go to the document library
      Then the library lists "portfolio-links.txt", "cover-template.md", and "resume-base-2026.docx" in that order

    @status:implemented @id:library-same-file-twice
    Scenario: The same file can be added more than once
      Given the library lists "resume-base-2026.docx" with the description "Before the rewrite"
      When I add "resume-base-2026.docx" again with the description "After the rewrite"
      Then the library lists "resume-base-2026.docx" twice, once with each description

    @status:implemented @id:library-document-kept-if-original-moved
    Scenario: A library document still opens after the original file is moved
      Given I added "resume-base-2026.docx" to the library
      And I have since moved or deleted the original "resume-base-2026.docx" on my computer
      When I open "resume-base-2026.docx" from the library
      Then the document opens with the content it had when I added it

  Rule: The library takes the same documents as job attachments

    @status:implemented @id:library-allowed-file-types
    Scenario Outline: Text-like document types can be added
      Given I am in the document library
      When I add the file "<file>"
      Then the library lists "<file>"

      Examples:
        | file            |
        | notes.txt       |
        | resume.pdf      |
        | prep.md         |
        | cover.docx      |
        | cover.doc       |
        | resume.rtf      |
        | resume.odt      |

    @status:implemented @id:library-file-type-any-case
    Scenario: A document's type is recognized whatever the letter case of its extension
      Given I am in the document library
      When I add the file "RESUME.PDF"
      Then the library lists "RESUME.PDF"

    @status:implemented @id:library-disallowed-file-types
    Scenario Outline: Other file types can't be added
      Given I am in the document library
      When I try to add the file "<file>"
      Then the file is not added
      And I am told which document types are allowed

      Examples:
        | file            |
        | headshot.png    |
        | take-home.zip   |
        | setup.exe       |
        | salaries.xlsx   |

    @status:implemented @id:library-file-size-limit
    Scenario: A file larger than 10 MB can't be added
      Given I am in the document library
      When I try to add a 12 MB file "portfolio.pdf"
      Then the file is not added
      And I am told files can be at most 10 MB

    @status:implemented @id:library-file-size-at-limit
    Scenario: A file of exactly 10 MB can be added
      Given I am in the document library
      When I add a 10 MB file "portfolio.pdf"
      Then the library lists "portfolio.pdf"

  Rule: Library documents can be opened, described, and removed

    @status:implemented @id:open-library-document
    Scenario: Open a library document
      Given the library lists "resume-base-2026.docx"
      When I open "resume-base-2026.docx" from the library
      Then it opens in my computer's default app for that file type

    @status:implemented @id:open-missing-library-document
    Scenario: Told when a library document's file is missing
      Given the library lists "resume-base-2026.docx"
      And its stored copy is no longer in the app's data directory
      When I open "resume-base-2026.docx" from the library
      Then I am told the file for "resume-base-2026.docx" can't be found
      And the library still lists "resume-base-2026.docx"

    @status:implemented @id:edit-library-description
    Scenario: Change a document's description
      Given the library lists "resume-base-2026.docx" with the description "Base resume"
      When I change its description to "Base resume, backend roles"
      Then the library lists "resume-base-2026.docx" with the description "Base resume, backend roles"

    @status:implemented @id:clear-library-description
    Scenario: Clear a document's description
      Given the library lists "resume-base-2026.docx" with the description "Base resume"
      When I clear its description
      Then the library lists "resume-base-2026.docx" with no description

    @status:implemented @id:remove-library-document
    Scenario: Remove a library document after confirming
      Given the library lists "resume-base-2026.docx"
      When I remove "resume-base-2026.docx" from the library
      And I confirm the removal
      Then the library no longer lists "resume-base-2026.docx"
      And the original "resume-base-2026.docx" on my computer is not deleted

    @status:implemented @id:cancel-remove-library-document
    Scenario: Cancel removing a library document
      Given the library lists "resume-base-2026.docx"
      When I remove "resume-base-2026.docx" from the library
      But I cancel the removal
      Then the library still lists "resume-base-2026.docx"

  Rule: The library is separate from job attachments

    @status:implemented @id:library-not-on-jobs
    Scenario: Library documents aren't attached to any job
      Given the library lists "resume-base-2026.docx"
      And I have a job for "Software Engineer" at "Acme" with no attached documents
      When I open the card
      Then the job has no attached documents

    @status:implemented @id:delete-job-keeps-library
    Scenario: Deleting a job doesn't touch the library
      Given the library lists "resume-base-2026.docx"
      And I have attached "resume-base-2026.docx" to a job for "Software Engineer" at "Acme"
      When I delete the job
      And I confirm the deletion
      Then the library still lists "resume-base-2026.docx"

    @status:implemented @id:remove-library-keeps-attachments
    Scenario: Removing a library document doesn't remove it from jobs
      Given the library lists "resume-base-2026.docx"
      And I have attached "resume-base-2026.docx" to a job for "Software Engineer" at "Acme"
      When I remove "resume-base-2026.docx" from the library
      And I confirm the removal
      Then the job still lists "resume-base-2026.docx"

  Rule: The library is kept between sessions

    @status:implemented @id:library-persists
    Scenario: The library is the same after reopening the app
      Given the library lists "resume-base-2026.docx" with the description "Base resume"
      When I close and reopen the app
      And I go to the document library
      Then the library lists "resume-base-2026.docx" with the description "Base resume"
