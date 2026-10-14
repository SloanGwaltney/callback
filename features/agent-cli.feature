Feature: Agent command line
  As a job seeker working through my own AI agent
  I want my agent to work with my Callback data through a command line program
  So that I can hand my agent job search tasks without leaving the agent I already use

  The command line follows the same rules as the app. Which jobs, contacts, interviews, and
  documents are valid is specified in their own feature files; these scenarios cover how my agent
  reaches them and what it's told.

  Rule: The help is how my agent learns what it can do

    @status:implemented @id:cli-help
    Scenario: The help lists every command
      When my agent asks Callback for help
      Then it sees every command, each with a short description of what it does

    @status:implemented @id:cli-command-help
    Scenario: The help for a command lists its options and the values they take
      When my agent asks for help on moving a job
      Then it sees every option for moving a job
      And it sees every status a job can be moved to

    @status:implemented @id:cli-help-lists-choices
    Scenario Outline: The help lists the values for every option that takes a fixed set
      When my agent asks for help on <command>
      Then it sees every <choice> it can use

      Examples:
        | command               | choice                    |
        | moving a job          | status                    |
        | closing a job         | closing reason            |
        | adding a contact      | relationship              |
        | adding an interview   | interview kind            |
        | attaching a document  | document kind             |
        | attaching a document  | file type it can attach   |

    @status:implemented @id:cli-overview
    Scenario: My agent can read an overview of Callback and its commands
      When my agent asks Callback for its overview
      Then it reads, in prose, what Callback keeps track of: jobs and their statuses, contacts, interviews, documents, next actions, status history, and the document library
      And it reads how the commands fit together, from finding a job by its ID to changing it
      And it reads how output, IDs, dates, long text, and errors work

    @status:not-implemented @id:cli-overview-working-safely
    Scenario: The overview tells my agent how to work safely
      When my agent asks Callback for its overview
      Then it's told to make a backup before changing many jobs at once, and how long to keep it
      And it's told how to restore that backup if the changes go wrong

    @status:implemented @id:cli-overview-limits
    Scenario: The overview says what my agent can't do
      When my agent asks Callback for its overview
      Then it's told it can't open anything on my screen, delete backups, or restore a daily backup
      And it's told to change a job's document by attaching a new file, not by editing the stored copy

    @status:implemented @id:cli-overview-read-documents
    Scenario: The overview tells my agent it can read my documents
      When my agent asks Callback for its overview
      Then it's told where to find the stored copies of a job's documents and the library's
      And it's told to read them when they help with what it's doing

    @status:implemented @id:cli-overview-matches-commands
    Scenario: Every command the overview names is one Callback has
      When my agent asks Callback for its overview
      Then every command it names is a command Callback has

    @status:implemented @id:cli-help-points-to-overview
    Scenario: The help points my agent to the overview
      When my agent asks Callback for help
      Then it's told how to read the overview

    @status:implemented @id:cli-help-stored-files
    Scenario: The help says how to use a document's stored copy
      When my agent asks for help on showing a job
      Then it's told it can read the stored copies of the job's documents
      And it's told to change a document by attaching a new file, not by editing the stored copy

  Rule: Output is plain text unless my agent asks for JSON

    @status:implemented @id:cli-text-output
    Scenario: Output is plain text by default
      Given I have a job for "Software Engineer" at "Acme"
      When my agent lists my jobs
      Then the list is plain text, one line per job

    @status:implemented @id:cli-json-output
    Scenario: My agent can ask for any command's output as JSON
      Given I have a job for "Software Engineer" at "Acme"
      When my agent lists my jobs as JSON
      Then the list is JSON with the same details as the plain text list, and more

    @status:implemented @id:cli-output-ids
    Scenario: Everything listed comes with its ID
      Given I have a job for "Software Engineer" at "Acme" with a contact, an interview, and a document
      When my agent shows the job
      Then the job, its contact, its interview, and its document each come with their ID

    @status:implemented @id:cli-dates
    Scenario: Dates are written as year, month, and day
      Given I have a job for "Software Engineer" at "Acme" that I applied to on October 1, 2026
      When my agent shows the job
      Then the date applied is written as 2026-10-01

    @status:implemented @id:cli-change-prints-result
    Scenario: A change prints what it changed
      Given I have a job for "Software Engineer" at "Acme" in "Saved"
      When my agent moves the job to "Applied"
      Then my agent is shown the job as it is after the move

  Rule: My agent is told plainly when something goes wrong

    @status:implemented @id:cli-error
    Scenario: A command that can't be done says why and changes nothing
      Given I have a job for "Software Engineer" at "Acme"
      When my agent tries to change the job's role to nothing
      Then the job's role is still "Software Engineer"
      And my agent is told the job needs a role
      And the command ends with the exit code for a validation error

    @status:implemented @id:cli-error-lists-choices
    Scenario: A value that isn't one of the choices is answered with the choices
      Given I have a job for "Software Engineer" at "Acme"
      When my agent tries to move the job to "Intervewing"
      Then the job doesn't move
      And my agent is told there's no status "Intervewing"
      And my agent is told the statuses it can use

    @status:implemented @id:cli-unknown-command
    Scenario: An unknown command points my agent to the help
      When my agent runs a command Callback doesn't have
      Then my agent is told there's no such command
      And my agent is told how to see the help
      And the command ends with the exit code for a usage error

    @status:implemented @id:cli-unknown-job
    Scenario: My agent is told when there's no job with an ID
      When my agent tries to show the job with ID 999
      Then my agent is told there's no job with ID 999
      And the command ends with the exit code for something not found

  Rule: Each kind of failure has its own exit code

    @status:implemented @id:cli-exit-codes
    Scenario Outline: A failed command ends with the exit code for its kind of failure
      When my agent runs a command that fails because <failure>
      Then the command ends with the exit code for <kind>
      And no other kind of failure ends with that exit code

      Examples:
        | failure                                                  | kind                           |
        | it isn't a command Callback has, or an option is wrong   | a usage error                  |
        | a value breaks one of the app's rules                    | a validation error             |
        | there's no job, contact, interview, document, or backup with that ID or name | something not found |
        | my data was last opened by a newer Callback              | a newer version of my data     |
        | anything else went wrong, like my data can't be opened   | an unexpected error            |

    @status:implemented @id:cli-validation-one-code
    Scenario Outline: Every validation error shares one exit code, with a message saying what's wrong
      Given I have a job for "Software Engineer" at "Acme"
      When my agent tries to <change>
      Then the command ends with the exit code for a validation error
      And my agent is told <message>

      Examples:
        | change                                   | message                                |
        | change the job's role to nothing         | the job needs a role                   |
        | rate the job 6 stars                     | a rating must be from 1 to 5 stars     |
        | set a posting link that isn't a web address | the link must be a web address      |

    @status:implemented @id:cli-success-code
    Scenario: A command that works ends with exit code 0
      Given I have a job for "Software Engineer" at "Acme"
      When my agent lists my jobs
      Then the command ends with exit code 0

    @status:implemented @id:cli-exit-codes-listed
    Scenario: The overview lists every exit code and what it means
      When my agent asks Callback for its overview
      Then it sees every exit code and the kind of failure each one means

  Rule: Errors are JSON when my agent asks for JSON

    @status:implemented @id:cli-json-errors
    Scenario: An error is JSON when my agent asked for JSON output
      Given I have a job for "Software Engineer" at "Acme"
      When my agent tries to change the job's role to nothing, asking for JSON output
      Then my agent is told the job needs a role, as JSON
      And the JSON says what kind of failure it was
      And the command ends with the exit code for a validation error

    @status:implemented @id:cli-text-errors
    Scenario: An error is plain text otherwise
      Given I have a job for "Software Engineer" at "Acme"
      When my agent tries to change the job's role to nothing
      Then my agent is told the job needs a role, as plain text

  Rule: The command line never works on data from a newer Callback

    @status:implemented @id:cli-newer-data
    Scenario: The command line refuses data a newer Callback has updated
      Given my data was last updated by a newer version of Callback than the command line
      When my agent runs any command
      Then my data is unchanged
      And no daily backup is taken and no backup is removed
      And my agent is told my data is from a newer version of Callback, and to update Callback
      And the command ends with the exit code for a newer version of my data

    @status:implemented @id:cli-older-data
    Scenario: The command line brings older data up to date, as the app does
      Given my data was last updated by an older version of Callback
      When my agent runs any command
      Then my data is brought up to date first
      And the command runs as usual

  Rule: My agent is never asked a question

    @status:implemented @id:cli-no-questions
    Scenario: Commands finish without waiting for an answer
      Given I have a job for "Software Engineer" at "Acme"
      When my agent deletes the job
      Then the job is deleted without my agent being asked to confirm

  Rule: Long text can be given from a file or piped in

    @status:implemented @id:cli-text-from-file
    Scenario Outline: My agent gives long text from a file
      Given I have a job for "Software Engineer" at "Acme"
      And a file holds a job description with quotes, line breaks, and blank lines
      When my agent sets the job's <field> from that file
      Then the job's <field> is exactly what the file holds

      Examples:
        | field       |
        | description |
        | notes       |

    @status:implemented @id:cli-text-piped
    Scenario: My agent pipes long text in
      Given I have a job for "Software Engineer" at "Acme"
      When my agent pipes a job description with quotes and line breaks into setting the job's description
      Then the job's description is exactly what was piped in
