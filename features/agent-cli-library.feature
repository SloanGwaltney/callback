Feature: Agent command line for the document library
  As a job seeker working through my own AI agent
  I want my agent to read and add to my document library
  So that my agent can work from my resumes and letters when it helps me apply

  The library's rules are the same as in the app; see document-library.feature.

  Rule: My agent can see and read the library

    @status:not-implemented @id:cli-list-library
    Scenario: List the library
      Given my library holds "resume-2026.pdf" described as "General resume"
      And my library holds "cover-letter.docx" with no description
      When my agent lists the library
      Then it sees both documents, newest first
      And it sees each one's ID, description, and where its stored copy is

    @status:not-implemented @id:cli-list-library-missing
    Scenario: The list marks a library document whose file is missing
      Given my library holds "resume-2026.pdf"
      And the stored copy of "resume-2026.pdf" is gone
      When my agent lists the library
      Then "resume-2026.pdf" is marked as missing its file

    @status:not-implemented @id:cli-list-library-empty
    Scenario: My agent is told when the library is empty
      Given my library is empty
      When my agent lists the library
      Then my agent is told the library has no documents

  Rule: My agent can add, describe, and remove library documents

    @status:not-implemented @id:cli-add-library-document
    Scenario: Add a document to the library
      Given there is a file "resume-2026.pdf" on my computer
      When my agent adds "resume-2026.pdf" to the library described as "General resume"
      Then the library lists "resume-2026.pdf" described as "General resume"
      And my agent is shown the document with its ID and where its stored copy is

    @status:not-implemented @id:cli-add-library-rules
    Scenario Outline: Adding a library document follows the app's rules
      When my agent tries to add <file> to the library
      Then the library has no new document
      And my agent is told <reason>

      Examples:
        | file                         | reason                              |
        | a file that doesn't exist    | there's no file at that path        |
        | "photo.png"                  | the file types it can add           |
        | a PDF larger than 10 MB      | a document can be at most 10 MB     |

    @status:not-implemented @id:cli-describe-library-document
    Scenario: Change a library document's description
      Given my library holds "resume-2026.pdf" described as "General resume"
      When my agent changes its description to "Resume for backend roles"
      Then the library lists "resume-2026.pdf" described as "Resume for backend roles"

    @status:not-implemented @id:cli-clear-library-description
    Scenario: Clear a library document's description
      Given my library holds "resume-2026.pdf" described as "General resume"
      When my agent clears its description
      Then the library lists "resume-2026.pdf" with no description

    @status:not-implemented @id:cli-remove-library-document
    Scenario: Remove a library document
      Given my library holds "resume-2026.pdf"
      When my agent removes it from the library
      Then the library doesn't list "resume-2026.pdf"
