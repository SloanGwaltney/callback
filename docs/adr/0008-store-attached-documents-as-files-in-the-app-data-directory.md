# 0008. Store attached documents as files in the app data directory

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

Users attach documents (resumes, cover letters, other text-like files up to 10 MB) to jobs. The
app has to keep its own copy of each one: an attached document must still open with its original
content after the user moves or deletes the file they attached. Attached documents also have to
open in the operating system's default app for their file type, and the same file can be attached
to more than one job, with removal from one job leaving it attached to the other.

User data lives in SQLite (see [docs/adr/0003-store-user-data-in-sqlite.md]), so the copy could go
either in the database as a BLOB or on the file system with the database pointing to it.

This ADR doesn't decide where the app's data directory is on each operating system. That's its own
decision, since the database file depends on it too.

## Decisions

1. When a document is attached, copy the file into a documents folder inside the app's data
   directory, and store a reference to it in the database. Don't store file content in the
   database.
2. Name each copied file with a generated ID plus the original extension (e.g.
   `documents/<id>.pdf`). Keep the original filename in the database.
3. Store the file's path relative to the app's data directory, not as an absolute path.
4. Make one copy per attachment. Attaching the same file to two jobs makes two copies, and
   removing an attachment deletes only its own copy.

## Reasons

- **1.** Opening a document in the default app requires a real file on disk. With BLOBs, every
  open would mean writing the content out to a temporary file first. With files on disk, the app
  hands the OS the stored path.
- **1.** Storing documents as BLOBs makes every document row about the size of its file, up to
  10 MB. That grows the database file, which doesn't shrink after deletes without a `VACUUM`.
  SQLite's own measurements also show files larger than about 100 KB read faster from the file
  system than from BLOBs.
- **1.** The copy lives inside the app data directory, which users don't normally touch, so the
  risk of a stored reference breaking because someone moved the file is low.
- **1.** Trade-offs accepted: deleting a document row no longer removes its content
  automatically, so removing an attachment or deleting a job must also delete the files from
  disk. Copying a file and inserting its row aren't one transaction, so a failure between the two
  can leave an orphaned file or a dangling row. The code has to order these steps carefully. And
  backing up the user's data means copying the documents folder along with the database file.
- **2.** Generated names can't collide, even when two attachments share an original filename,
  and changing how a document is listed never requires renaming anything on disk.
- **3.** A relative path keeps working if the data directory moves or the data is restored on
  another machine.
- **4.** One copy per attachment keeps removal simple: deleting an attachment deletes its file,
  with no reference counting. The cost is duplicate copies when the same file is attached to
  several jobs, which the 10 MB limit keeps small.

## Alternatives considered

- **Store document content as BLOBs in SQLite.** Its real advantage is that deleting a row
  deletes the content with it, with no file system cleanup, and everything stays in one file for
  backups and transactions. Rejected because opening a document in the default app would require
  writing a temporary file on every open, and because of the database size and read performance
  costs above.
- **Name copied files by their original filename, in a folder per job.** Easier to read when
  browsing the folder, but two attachments with the same name on one job would collide.
  Rejected in favor of generated IDs.
- **Store absolute paths.** Simpler, but they break if the data directory moves. Rejected.
- **Deduplicate copies by content hash.** Saves disk space when one file is attached to several
  jobs, but removing an attachment would need reference counting before deleting the shared
  file. Rejected as not worth the complexity at a 10 MB limit.

## Related

- [features/job-documents.feature#attach-document] Attach a document of a given kind
- [features/job-documents.feature#attach-multiple-documents] Attach several documents to one job
- [features/job-documents.feature#document-kept-if-original-moved] An attached document still opens after the original file is moved
- [features/job-documents.feature#open-document] Open an attached document
- [features/job-documents.feature#remove-document] Remove an attached document
- [features/job-documents.feature#same-document-many-jobs] The same file can be attached to more than one job
- [features/job-cards.feature#delete-job] Delete a job after confirming
- [docs/adr/0003-store-user-data-in-sqlite.md] Store user data in SQLite
