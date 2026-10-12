Feature: Agent backups
  As a job seeker working through my own AI agent
  I want my agent to back up my data before it makes changes, and restore it if they go wrong
  So that I can let my agent change my board without worrying about losing anything

  Rule: My agent can make named backups

    @status:not-implemented @id:agent-backup
    Scenario: My agent backs up my data under a name
      Given my agent is connected to Callback
      And I have a job for "Software Engineer" at "Acme"
      When my agent makes a backup named "before-cleanup"
      Then a backup named "before-cleanup" holds my data as it is now

    @status:not-implemented @id:agent-backup-needs-name
    Scenario: A backup needs a name
      Given my agent is connected to Callback
      When my agent tries to make a backup with no name
      Then no backup is made
      And my agent is told the backup needs a name

    @status:not-implemented @id:agent-backup-name-characters
    Scenario Outline: A backup name may only use letters, numbers, hyphens, and underscores
      Given my agent is connected to Callback
      When my agent tries to make a backup named "<name>"
      Then no backup is made
      And my agent is told which characters a backup name may use

      Examples:
        | name           |
        | ../escape      |
        | before cleanup |
        | notes.db       |

    @status:not-implemented @id:agent-backup-name-taken
    Scenario: A backup can't reuse the name of an existing backup
      Given my agent made a backup named "before-cleanup" in an earlier session
      And my agent is connected to Callback in a new session
      When my agent tries to make a backup named "before-cleanup"
      Then no backup is made
      And the existing "before-cleanup" backup is unchanged
      And my agent is told that name is already taken

    @status:not-implemented @id:agent-list-session-backups
    Scenario: My agent can see the backups it made this session
      Given my agent made a backup named "old-backup" in an earlier session
      And my agent is connected to Callback in a new session
      And my agent has made backups named "before-cleanup" and "before-merge" this session
      When my agent lists its backups
      Then it sees "before-cleanup" and "before-merge"
      But it doesn't see "old-backup"

  Rule: My agent can only delete backups it made this session

    @status:not-implemented @id:agent-delete-backup
    Scenario: My agent deletes a backup it made this session
      Given my agent is connected to Callback
      And my agent made a backup named "before-cleanup"
      When my agent deletes the backup "before-cleanup"
      Then there is no backup named "before-cleanup"
      And my data is unchanged

    @status:not-implemented @id:agent-delete-other-session
    Scenario: My agent can't delete a backup from an earlier session
      Given my agent made a backup named "before-cleanup" in an earlier session
      And my agent is connected to Callback in a new session
      When my agent tries to delete the backup "before-cleanup"
      Then the "before-cleanup" backup is still kept
      And my agent is told it can only delete backups it made this session

    @status:not-implemented @id:agent-delete-daily-backup
    Scenario: My agent can't delete a daily backup
      Given a daily backup was taken today
      And my agent is connected to Callback
      When my agent tries to delete the daily backup
      Then the daily backup is still kept
      And my agent is told it can only delete backups it made this session

    @status:not-implemented @id:agent-delete-unknown
    Scenario: My agent is told when there's no backup with that name to delete
      Given my agent is connected to Callback
      When my agent tries to delete a backup named "never-made"
      Then my agent is told there's no backup it made this session with that name

  Rule: My agent can only restore backups it made this session

    @status:not-implemented @id:agent-restore
    Scenario: My agent restores a backup it made this session
      Given my agent is connected to Callback
      And my agent made a backup named "before-cleanup" while I had a job at "Acme" in "Applied"
      And my agent then moved the job to "Closed"
      When my agent restores the backup "before-cleanup"
      Then the job at "Acme" is in "Applied" again

    @status:not-implemented @id:agent-restore-other-session
    Scenario: My agent can't restore a backup from an earlier session
      Given my agent made a backup named "before-cleanup" in an earlier session
      And my agent is connected to Callback in a new session
      When my agent tries to restore the backup "before-cleanup"
      Then my data is unchanged
      And my agent is told it can only restore backups it made this session

    @status:not-implemented @id:agent-restore-daily-backup
    Scenario: My agent can't restore a daily backup
      Given a daily backup was taken today
      And my agent is connected to Callback
      When my agent tries to restore the daily backup
      Then my data is unchanged
      And my agent is told it can only restore backups it made this session

    @status:not-implemented @id:agent-restore-unknown
    Scenario: My agent is told when there's no backup with that name
      Given my agent is connected to Callback
      When my agent tries to restore a backup named "never-made"
      Then my data is unchanged
      And my agent is told there's no backup it made this session with that name

  Rule: A restore can be undone

    @status:not-implemented @id:restore-backs-up-first
    Scenario: Restoring backs up my current data first
      Given my agent is connected to Callback
      And my agent made a backup named "before-cleanup"
      When my agent restores the backup "before-cleanup"
      Then a backup of my data from just before the restore is made first
      And my agent is told that backup's name

    @status:not-implemented @id:undo-restore
    Scenario: My agent undoes a restore
      Given my agent is connected to Callback
      And my agent made a backup named "before-cleanup" while I had a job at "Acme" in "Applied"
      And my agent then moved the job to "Closed"
      And my agent restored the backup "before-cleanup"
      When my agent restores the backup made just before that restore
      Then the job at "Acme" is in "Closed" again

  Rule: Restoring brings back my data, not the documents' files

    @status:not-implemented @id:restore-missing-document-file
    Scenario: A document removed after the backup is listed again but its file is gone
      Given my agent made a backup named "before-cleanup" while "resume-2026.pdf" was attached to a job at "Acme"
      And I then removed "resume-2026.pdf" from the job
      When my agent restores the backup "before-cleanup"
      Then the job at "Acme" lists "resume-2026.pdf" again
      But opening it tells me its file can't be found

    @status:not-implemented @id:restore-app-open
    Scenario: The app shows the restored data when I come back to it
      Given the app is open and I have switched to another window
      And my agent restores a backup it made this session
      When I switch back to the app
      Then the board shows the restored data

    @status:not-implemented @id:restore-keeps-document-files
    Scenario: Restoring never deletes a document's file
      Given my agent made a backup named "before-cleanup"
      And I then attached "acme-cover.docx" to a job at "Acme"
      When my agent restores the backup "before-cleanup"
      Then the job at "Acme" no longer lists "acme-cover.docx"
      But the stored copy of "acme-cover.docx" is still in the app's data directory
