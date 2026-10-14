Feature: Agent backups
  As a job seeker working through my own AI agent
  I want my agent to back up my data before it makes changes, and restore it if they go wrong
  So that I can let my agent change my board without worrying about losing anything

  Rule: My agent can make named backups that are kept for a number of days it chooses

    @status:implemented @id:agent-backup
    Scenario: My agent backs up my data under a name, kept for a number of days
      Given today is 2026-10-14
      And I have a job for "Software Engineer" at "Acme"
      When my agent makes a backup named "before-cleanup" kept for 7 days
      Then a backup named "before-cleanup" holds my data as it is now
      And the "before-cleanup" backup is kept until 2026-10-21

    @status:implemented @id:agent-backup-needs-name
    Scenario: A backup needs a name
      When my agent tries to make a backup with no name, kept for 7 days
      Then no backup is made
      And my agent is told the backup needs a name

    @status:implemented @id:agent-backup-name-characters
    Scenario Outline: A backup name may only use letters, numbers, hyphens, and underscores
      When my agent tries to make a backup named "<name>" kept for 7 days
      Then no backup is made
      And my agent is told which characters a backup name may use

      Examples:
        | name           |
        | ../escape      |
        | before cleanup |
        | notes.db       |

    @status:implemented @id:agent-backup-needs-retention
    Scenario: A backup needs to say how many days it's kept
      When my agent tries to make a backup named "before-cleanup" without saying how many days to keep it
      Then no backup is made
      And my agent is told the backup needs a number of days from 1 to 90

    @status:implemented @id:agent-backup-retention-range
    Scenario Outline: A backup is kept for 1 to 90 days
      Given today is 2026-10-14
      When my agent tries to make a backup named "before-cleanup" kept for <days> days
      Then <outcome>

      Examples:
        | days | outcome                                                        |
        | 1    | the "before-cleanup" backup is kept until 2026-10-15           |
        | 90   | the "before-cleanup" backup is kept until 2027-01-12           |
        | 0    | no backup is made, and my agent is told to use 1 to 90 days    |
        | 91   | no backup is made, and my agent is told to use 1 to 90 days    |

    @status:implemented @id:agent-backup-name-in-use
    Scenario: A backup can't reuse the name of a backup that's still kept
      Given another agent made a backup named "before-cleanup" that is still kept
      When my agent tries to make a backup named "before-cleanup" kept for 7 days
      Then no backup is made
      And the existing "before-cleanup" backup is unchanged
      And my agent is told that name is already taken

    @status:implemented @id:agent-backup-name-freed
    Scenario: An expired backup's name can be used again
      Given today is 2026-10-22
      And my agent made a backup named "before-cleanup" that was kept until 2026-10-21
      When my agent makes a backup named "before-cleanup" kept for 7 days
      Then a backup named "before-cleanup" holds my data as it is now
      And the "before-cleanup" backup is kept until 2026-10-29

  Rule: My agent can see every backup an agent made, but not the daily backups

    @status:implemented @id:agent-list-backups
    Scenario: My agent lists the backups agents have made
      Given an agent made a backup named "old-backup" last week that is still kept
      And my agent has just made a backup named "before-cleanup"
      And a daily backup was taken today
      When my agent lists the backups
      Then it sees "old-backup" and "before-cleanup", each with the day it's kept until
      But it doesn't see the daily backup

  Rule: Expired backups are removed whenever Callback runs

    @status:implemented @id:agent-backup-kept-through-last-day
    Scenario: A backup is still kept on the last day it's kept until
      Given today is 2026-10-21
      And my agent made a backup named "before-cleanup" kept until 2026-10-21
      When I open the app
      Then the "before-cleanup" backup is still kept

    @status:implemented @id:agent-backup-expires
    Scenario Outline: An expired backup is removed when Callback runs
      Given today is 2026-10-22
      And my agent made a backup named "before-cleanup" kept until 2026-10-21
      When <run>
      Then there is no backup named "before-cleanup"

      Examples:
        | run                              |
        | I open the app                   |
        | my agent runs a Callback command |

    @status:implemented @id:agent-backup-expiry-retries
    Scenario: An expired backup that can't be removed yet is removed on a later run
      Given today is 2026-10-22
      And my agent made a backup named "before-cleanup" kept until 2026-10-21
      And the "before-cleanup" backup can't be removed right now
      When I open the app
      Then the app opens as usual
      And the "before-cleanup" backup is removed the next time Callback runs and can remove it

    @status:implemented @id:agent-backup-expiry-keeps-daily
    Scenario: Removing expired backups never removes a daily backup
      Given there is a daily backup from 60 days ago
      When my agent runs a Callback command
      Then the daily backup from 60 days ago is still kept

  Rule: My agent can't delete backups

    @status:implemented @id:agent-cant-delete-backup
    Scenario: My agent has no way to delete a backup
      Given my agent made a backup named "before-cleanup"
      When my agent looks at what it can do with backups
      Then it can make, list, and restore backups
      But it can't delete one

  Rule: My agent can restore any backup an agent made, but not the daily backups

    @status:implemented @id:agent-restore
    Scenario: My agent restores a backup
      Given my agent made a backup named "before-cleanup" while I had a job at "Acme" in "Applied"
      And my agent then moved the job to "Closed"
      When my agent restores the backup "before-cleanup"
      Then the job at "Acme" is in "Applied" again

    @status:implemented @id:agent-restore-earlier-backup
    Scenario: My agent restores a backup another agent made earlier
      Given an agent made a backup named "last-week" last week, while I had a job at "Acme" in "Applied"
      And the job at "Acme" has since moved to "Interviewing"
      When my agent restores the backup "last-week"
      Then the job at "Acme" is in "Applied" again

    @status:implemented @id:agent-restore-daily-backup
    Scenario: My agent can't restore a daily backup
      Given a daily backup was taken today
      When my agent tries to restore the daily backup
      Then my data is unchanged
      And my agent is told it can only restore backups an agent made

    @status:implemented @id:agent-restore-unknown
    Scenario: My agent is told when there's no backup with that name
      When my agent tries to restore a backup named "never-made"
      Then my data is unchanged
      And my agent is told there's no backup with that name

    @status:implemented @id:restore-keeps-backups
    Scenario: Restoring doesn't change which backups are kept
      Given my agent made a backup named "first" and then a backup named "second"
      When my agent restores the backup "first"
      Then my agent still sees "first" and "second" when it lists the backups
      And each is still kept until the same day as before

  Rule: A restore works with backups from older versions of Callback, but not newer ones

    @status:implemented @id:restore-older-backup
    Scenario: A backup from an older version is brought up to date when it's restored
      Given my agent made a backup named "before-cleanup" while I had a job at "Acme" in "Applied"
      And I have since updated to a version of Callback that changes how my data is stored
      When my agent restores the backup "before-cleanup"
      Then the job at "Acme" is in "Applied" again
      And my data is stored the way the current version stores it
      And the "before-cleanup" backup itself is unchanged

    @status:implemented @id:restore-newer-backup
    Scenario: A backup from a newer version can't be restored
      Given a backup named "from-newer" was made by a newer version of Callback than the command line
      When my agent tries to restore the backup "from-newer"
      Then my data is unchanged
      And my agent is told the backup is from a newer version of Callback
      And the command ends with the exit code for a newer version of my data

    @status:implemented @id:restore-app-open-safe
    Scenario: Restoring while the app is open doesn't need the app closed
      Given the app is open
      And my agent made a backup named "before-cleanup"
      When my agent restores the backup "before-cleanup"
      Then the restore finishes without the app being closed
      And the app keeps working with the restored data

  Rule: A restore can be undone

    @status:implemented @id:restore-backs-up-first
    Scenario: Restoring backs up my current data first
      Given today is 2026-10-14
      And my agent made a backup named "before-cleanup"
      When my agent restores the backup "before-cleanup"
      Then a backup of my data from just before the restore is made first
      And that backup is kept until 2026-10-21
      And it's listed with the other backups agents made
      And my agent is told that backup's name

    @status:implemented @id:undo-restore
    Scenario: My agent undoes a restore
      Given my agent made a backup named "before-cleanup" while I had a job at "Acme" in "Applied"
      And my agent then moved the job to "Closed"
      And my agent restored the backup "before-cleanup"
      When my agent restores the backup made just before that restore
      Then the job at "Acme" is in "Closed" again

  Rule: Restoring brings back my data, not the documents' files

    @status:implemented @id:restore-missing-document-file
    Scenario: A document removed after the backup is listed again but its file is gone
      Given my agent made a backup named "before-cleanup" while "resume-2026.pdf" was attached to a job at "Acme"
      And I then removed "resume-2026.pdf" from the job
      When my agent restores the backup "before-cleanup"
      Then the job at "Acme" lists "resume-2026.pdf" again
      But opening it tells me its file can't be found

    @status:implemented @id:restore-app-open
    Scenario: The app shows the restored data when I come back to it
      Given the app is open and I have switched to another window
      And my agent restores a backup an agent made
      When I switch back to the app
      Then the board shows the restored data

    @status:implemented @id:restore-keeps-document-files
    Scenario: Restoring never deletes a document's file
      Given my agent made a backup named "before-cleanup"
      And I then attached "acme-cover.docx" to a job at "Acme"
      When my agent restores the backup "before-cleanup"
      Then the job at "Acme" no longer lists "acme-cover.docx"
      But the stored copy of "acme-cover.docx" is still in the app's data directory
