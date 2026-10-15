package cli

import (
	"fmt"
	"io"
	"strconv"

	"callback/internal/backups"
)

// backupCommands are the commands for agent backups, in the order the help lists them. There's no
// command to delete a backup: each is removed once it expires. The rules are the backups
// service's.
// @scenario [#agent-cant-delete-backup]
var backupCommands = []*command{
	{
		path:      []string{"backup", "make"},
		takesName: "backup",
		summary:   "Back up the person's data under a name, kept for 1 to 90 days",
		about: fmt.Sprintf("Backs up all of the person's data, as it is now, under the name you give. Make one before "+
			"you change many jobs at once, so you can restore it if the changes go wrong. A name may only use "+
			"letters, numbers, hyphens, and underscores, and can't be the name of a backup that's still kept.\n\n"+
			"The backup is kept through the day %d to %d days from today, as --days says, and then removed. "+
			"Nothing can remove it before then. A backup holds which documents are attached, but not their "+
			"files.", backups.MinDays, backups.MaxDays),
		options: []option{
			{name: "days", value: "<days>", usage: fmt.Sprintf("how many days to keep the backup, from %d to %d", backups.MinDays, backups.MaxDays)},
		},
		run: makeBackup,
	},
	{
		path:    []string{"backup", "list"},
		summary: "List the backups agents have made, with the day each is kept until",
		about: "Lists every backup an agent made that's still kept, one line each: its name and the last day " +
			"it's kept. Callback's own daily backups aren't listed, and can't be restored from the command line.",
		run: listBackups,
	},
	{
		path:      []string{"backup", "restore"},
		takesName: "backup",
		summary:   "Restore a backup an agent made, replacing the person's data",
		about: fmt.Sprintf("Replaces all of the person's data with the backup's, without asking to confirm. Just "+
			"before, it backs up the data as it is, kept for %d days, and tells you that backup's name: restore "+
			"it to undo this restore. Restoring doesn't change which backups are kept.\n\n"+
			"It restores only a backup an agent made, never a daily backup. A backup from an older version of "+
			"Callback is brought up to date as it's restored, and the backup itself is left as it was. A backup "+
			"from a newer version can't be restored. The app can stay open: it shows the restored data when the "+
			"person comes back to it.\n\n"+
			"Documents' files aren't part of a backup. A document attached after the backup isn't listed once "+
			"it's restored, but its stored copy is kept. A document removed after the backup is listed again, "+
			"but its stored copy is gone, so it's marked missing.", backups.UndoDays),
		run: restoreBackup,
	},
}

// @scenario [#agent-backup] [#agent-backup-needs-name] [#agent-backup-name-characters] [#agent-backup-needs-retention] [#agent-backup-retention-range] [#agent-backup-name-in-use] [#agent-backup-name-freed]
func makeBackup(c *call) error {
	value, ok := c.value("days")
	if !ok {
		return usageErr("`backup make` needs --days with the number of days to keep the backup, from %d to %d", backups.MinDays, backups.MaxDays)
	}
	days, err := strconv.Atoi(value)
	if err != nil {
		return validationErr("%q isn't a number of days; use a whole number from %d to %d", value, backups.MinDays, backups.MaxDays)
	}
	b, err := c.backups.MakeAgent(c.ctx, c.name, days)
	if err != nil {
		return err
	}
	return c.output(b, func(w io.Writer) {
		fmt.Fprintf(w, "Made the backup %s, kept until %s.\n", b.Name, b.KeptUntil)
	})
}

// @scenario [#agent-list-backups]
func listBackups(c *call) error {
	listed, err := c.backups.ListAgent()
	if err != nil {
		return err
	}
	if listed == nil {
		listed = []backups.AgentBackup{}
	}
	return c.output(listed, func(w io.Writer) {
		if len(listed) == 0 {
			fmt.Fprintln(w, "No backups made by agents are kept.")
		}
		for _, b := range listed {
			fmt.Fprintf(w, "%s | kept until %s\n", b.Name, b.KeptUntil)
		}
	})
}

// restoredJSON is what a restore prints with --json: the backup restored, and the backup of the
// data from just before, which undoes the restore.
type restoredJSON struct {
	Restored string              `json:"restored"`
	Undo     backups.AgentBackup `json:"undo"`
}

// @scenario [#agent-restore] [#agent-restore-earlier-backup] [#agent-restore-daily-backup] [#agent-restore-unknown] [#restore-keeps-backups] [#restore-older-backup] [#restore-newer-backup] [#restore-app-open-safe] [#restore-backs-up-first] [#undo-restore] [#restore-missing-document-file] [#restore-app-open] [#restore-keeps-document-files]
func restoreBackup(c *call) error {
	undo, err := c.backups.Restore(c.ctx, c.name)
	if err != nil {
		return err
	}
	return c.output(restoredJSON{Restored: c.name, Undo: undo}, func(w io.Writer) {
		fmt.Fprintf(w, "Restored the backup %s.\n", c.name)
		fmt.Fprintf(w, "The data from just before is in the backup %s, kept until %s. Restore it to undo this restore.\n", undo.Name, undo.KeptUntil)
	})
}
