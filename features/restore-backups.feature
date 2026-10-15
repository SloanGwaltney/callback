Feature: Restoring backups in the app
  As a job seeker
  I want to restore any of my backups from inside the app
  So that I can get my job search back if something goes wrong, without digging through files

  How daily backups are taken and how many are kept is in daily-backups.feature. How my agent
  makes, lists, and restores backups is in agent-backups.feature.

  Rule: I can see all my backups in the app

    @status:implemented @id:app-backups-open
    Scenario: Go from the board to my backups
      Given I am on the board
      When I open my backups
      Then I see my backups

    @status:implemented @id:app-backups-back-to-board
    Scenario: Go back to the board from my backups
      Given I am looking at my backups
      When I go back to the board
      Then I see the board

    @status:implemented @id:app-backups-newest-first
    Scenario: Daily backups are listed newest first, by date
      Given daily backups were taken on 2026-10-12, 2026-10-14, and 2026-10-13
      When I open my backups
      Then I see the backups from 2026-10-14, 2026-10-13, and 2026-10-12, in that order

    @status:implemented @id:app-backups-none
    Scenario: Told when there are no backups yet
      Given no backups have been taken
      When I open my backups
      Then I'm told there are no backups yet, and that one is taken each day Callback runs

    @status:implemented @id:app-backups-list-every-kind
    Scenario: Daily backups, my agent's backups, and the app's backups are listed together
      Given a daily backup was taken on 2026-10-12
      And my agent made a backup named "before-cleanup" on 2026-10-13
      And I restored a backup on 2026-10-14, so a backup was made before restoring it
      When I open my backups
      Then I see all 3 backups, newest first
      And each says whether it's a daily backup, made by my agent, or made before a restore
      And the backup from my agent shows its name "before-cleanup"
      And the backups made by my agent or before a restore show the day each is kept until

  Rule: I can restore any backup after confirming

    @status:implemented @id:app-restore-daily-backup
    Scenario: Restore a daily backup
      Given a daily backup was taken on 2026-10-12 while I had a job at "Acme" in "Applied"
      And I have since moved the job to "Closed"
      When I restore the backup from 2026-10-12 and confirm
      Then I see the board
      And the job at "Acme" is in "Applied" again

    @status:implemented @id:app-restore-agent-backup
    Scenario: Restore a backup my agent made
      Given my agent made a backup named "before-cleanup" while I had a job at "Acme" in "Applied"
      And my agent then moved the job to "Closed"
      When I restore the backup "before-cleanup" and confirm
      Then I see the board
      And the job at "Acme" is in "Applied" again

    @status:implemented @id:app-restore-confirm-says-what-changes
    Scenario: The confirmation says my current data will be replaced
      Given a daily backup was taken on 2026-10-12
      When I choose to restore the backup from 2026-10-12
      Then I'm asked to confirm that my current data will be replaced with the data from 2026-10-12
      And I'm told a backup of my current data will be made first, so I can undo it

    @status:implemented @id:app-restore-cancel
    Scenario: Cancel restoring a backup
      Given a daily backup was taken on 2026-10-12 while I had a job at "Acme" in "Applied"
      And I have since moved the job to "Closed"
      When I choose to restore the backup from 2026-10-12
      And I cancel
      Then the job at "Acme" is still in "Closed"

    @status:implemented @id:app-restore-keeps-daily-backups
    Scenario: Restoring doesn't change which daily backups are kept
      Given daily backups were taken on 2026-10-12, 2026-10-13, and 2026-10-14
      When I restore the backup from 2026-10-12 and confirm
      Then I still see the backups from 2026-10-14, 2026-10-13, and 2026-10-12

  Rule: A restore in the app can be undone

    @status:implemented @id:app-restore-backs-up-first
    Scenario: Restoring backs up my current data first
      Given today is 2026-10-14
      And a daily backup was taken on 2026-10-12
      When I restore the backup from 2026-10-12 and confirm
      Then a backup of my data from just before the restore is made first
      And that backup is kept until 2026-10-21
      And it's listed with my backups as made before restoring 2026-10-12

    @status:implemented @id:app-undo-restore
    Scenario: Undo a restore
      Given a daily backup was taken on 2026-10-12 while I had a job at "Acme" in "Applied"
      And I have since moved the job to "Closed"
      And I restored the backup from 2026-10-12
      When I restore the backup made before that restore and confirm
      Then the job at "Acme" is in "Closed" again

    @status:implemented @id:app-pre-restore-backup-not-daily
    Scenario: The backup made before a restore isn't counted as a daily backup
      Given there are 15 daily backups
      And I restored one of them
      When a new daily backup is taken
      Then the backup made before that restore is still kept

  Rule: A restore works with backups from older versions of Callback, but not newer ones

    @status:implemented @id:app-restore-older-version
    Scenario: A backup from an older version is brought up to date when it's restored
      Given a daily backup was taken on 2026-10-12 while I had a job at "Acme" in "Applied"
      And I have since updated to a version of Callback that changes how my data is stored
      When I restore the backup from 2026-10-12 and confirm
      Then the job at "Acme" is in "Applied" again
      And the backup from 2026-10-12 itself is unchanged

    @status:implemented @id:app-restore-newer-version
    Scenario: A backup from a newer version can't be restored
      Given a daily backup from 2026-10-12 was taken by a newer version of Callback
      When I restore the backup from 2026-10-12 and confirm
      Then my data is unchanged
      And I'm told the backup is from a newer version of Callback, and to update Callback to restore it

  Rule: If a restore fails, my data is left as it was

    @status:implemented @id:app-restore-unreadable
    Scenario: Told when a backup can't be read
      Given the daily backup from 2026-10-12 is damaged
      When I restore the backup from 2026-10-12 and confirm
      Then my data is unchanged
      And I'm told the backup couldn't be restored

    @status:implemented @id:app-restore-missing
    Scenario: Told when a backup has been removed since I opened my backups
      Given I am looking at my backups
      And the backup from 2026-10-12 has since been removed
      When I restore the backup from 2026-10-12 and confirm
      Then my data is unchanged
      And I'm told that backup no longer exists
      And the backup from 2026-10-12 is no longer listed

  Rule: Restoring brings back my data, not the documents' files

    @status:implemented @id:app-restore-missing-document-file
    Scenario: A document removed after the backup is listed again but its file is gone
      Given a daily backup was taken on 2026-10-12 while "resume-2026.pdf" was attached to a job at "Acme"
      And I then removed "resume-2026.pdf" from the job
      When I restore the backup from 2026-10-12 and confirm
      Then the job at "Acme" lists "resume-2026.pdf" again
      But opening it tells me its file can't be found

    @status:implemented @id:app-restore-keeps-document-files
    Scenario: Restoring never deletes a document's file
      Given a daily backup was taken on 2026-10-12
      And I then attached "acme-cover.docx" to a job at "Acme"
      When I restore the backup from 2026-10-12 and confirm
      Then the job at "Acme" no longer lists "acme-cover.docx"
      But the stored copy of "acme-cover.docx" is still in the app's data directory
