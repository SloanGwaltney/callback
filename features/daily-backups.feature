Feature: Daily backups
  As a job seeker
  I want my job search data backed up automatically
  So that I can recover it if something goes wrong, even if I haven't opened the app in a while

  Rule: Callback backs up my data once on each day it runs

    @status:implemented @id:backup-on-first-start
    Scenario Outline: The first start of the day takes a backup
      Given today is 2026-10-14
      And no backup has been taken today
      When <start>
      Then a backup of my data from 2026-10-14 is taken

      Examples:
        | start                            |
        | I open the app                   |
        | my agent runs a Callback command |

    @status:implemented @id:backup-once-per-day
    Scenario: Starting Callback again the same day doesn't take another backup
      Given today is 2026-10-14
      And a backup was taken earlier today
      When I open the app
      Then no new backup is taken

    @status:implemented @id:backup-no-catch-up
    Scenario: Days when Callback didn't run get no backup
      Given the last backup was taken on 2026-10-01
      And Callback hasn't run since
      And today is 2026-10-14
      When I open the app
      Then one backup is taken, for 2026-10-14

    @status:implemented @id:backup-holds-data-not-documents
    Scenario: A backup holds my data but not the attached documents' files
      Given I have a job for "Software Engineer" at "Acme" with "resume-2026.pdf" attached
      When a daily backup is taken
      Then the backup holds the job, its details, and that "resume-2026.pdf" is attached
      But the backup doesn't hold a copy of the "resume-2026.pdf" file

  Rule: Only the most recent daily backups are kept

    @status:implemented @id:backup-keeps-last-15
    Scenario: Taking a 16th daily backup removes the oldest
      Given there are 15 daily backups
      When a new daily backup is taken
      Then there are still 15 daily backups
      And the oldest one has been removed

    @status:implemented @id:backup-count-not-days
    Scenario: Backups are kept by count, not by age
      Given there are 3 daily backups, the oldest from 60 days ago
      When a new daily backup is taken
      Then all 4 daily backups are kept

    @status:implemented @id:backup-rotation-keeps-agent-backups
    Scenario: Removing old daily backups never removes a backup my agent made
      Given there are 15 daily backups
      And my agent has made a backup named "before-cleanup"
      When a new daily backup is taken
      Then the "before-cleanup" backup is still kept

  Rule: A failed backup doesn't stop Callback from working

    @status:implemented @id:backup-failure-still-starts
    Scenario: Callback still starts when the daily backup fails
      Given no backup has been taken today
      And the backup can't be written
      When I open the app
      Then the board opens as usual
      And no partial backup is left behind
