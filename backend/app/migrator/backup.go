package migrator

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
)

// AutoBackup struct handles daily backups params for siteID
type AutoBackup struct {
	Exporter       Exporter
	BackupLocation string
	SiteID         string
	KeepMax        int
	Duration       time.Duration
}

// Do writes a backup at start when the site has none for the current day, then one every Duration, and
// keeps up to KeepMax backups of the site. It returns once ctx is done; an export in progress stops at
// its next write
func (ab AutoBackup) Do(ctx context.Context) {
	log.Printf("[INFO] activate auto-backup for %s under %s, duration %s", ab.SiteID, ab.BackupLocation, ab.Duration)
	tick := time.NewTicker(ab.Duration)
	defer tick.Stop()

	// the ticker alone never fires on an instance restarted more often than every Duration; the check
	// for the file of the day keeps the second and later starts of a day from writing it again
	if now := time.Now(); ctx.Err() == nil && !ab.hasBackup(now) {
		ab.startBackup(ctx, now)
	}
	log.Printf("[DEBUG] next backup for %s at %s", ab.SiteID, time.Now().Add(ab.Duration))

	for {
		select {
		case <-tick.C:
			ab.backup(ctx, time.Now())
			log.Printf("[DEBUG] next backup for %s at %s", ab.SiteID, time.Now().Add(ab.Duration))
		case <-ctx.Done():
			log.Printf("[WARN] terminated autobackup for %s", ab.SiteID)
			return
		}
	}
}

// backup writes the backup of the day of now and, if it succeeds, removes the backups over KeepMax
func (ab AutoBackup) backup(ctx context.Context, now time.Time) {
	if _, err := ab.makeBackup(ctx, now); err != nil {
		log.Printf("[WARN] auto-backup for %s failed, %s", ab.SiteID, err)
		return
	}
	ab.removeOldBackupFiles()
}

// startBackup is the backup written at start. An export with no comments is thrown away when keeping it
// would push an existing backup out: a server started on an empty store, as it is before a restore, must
// not replace its recovery points with empty archives
func (ab AutoBackup) startBackup(ctx context.Context, now time.Time) {
	tmpFile, comments, err := ab.exportToTemp(ctx, now)
	if err != nil {
		log.Printf("[WARN] auto-backup for %s failed, %s", ab.SiteID, err)
		return
	}
	if comments == 0 && len(ab.backupFiles()) >= ab.KeepMax {
		log.Printf("[WARN] backup of %s at start has no comments and would replace an existing backup, skipped", ab.SiteID)
		if e := os.Remove(tmpFile); e != nil {
			log.Printf("[WARN] can't remove %s, %s", tmpFile, e)
		}
		return
	}
	if err := ab.publish(tmpFile, ab.backupFile(now)); err != nil {
		log.Printf("[WARN] auto-backup for %s failed, %s", ab.SiteID, err)
		return
	}
	ab.removeOldBackupFiles()
}

// hasBackup reports whether the backup of the site for the day of now exists
func (ab AutoBackup) hasBackup(now time.Time) bool {
	_, err := os.Stat(ab.backupFile(now))
	return err == nil
}

// backupFile is the name of the backup of the site for the day of now in the local time zone
func (ab AutoBackup) backupFile(now time.Time) string {
	return fmt.Sprintf("%s/backup-%s-%s.gz", ab.BackupLocation, ab.SiteID, now.Format("20060102"))
}

func (ab AutoBackup) makeBackup(ctx context.Context, now time.Time) (string, error) {
	tmpFile, _, err := ab.exportToTemp(ctx, now)
	if err != nil {
		return "", err
	}
	backupFile := ab.backupFile(now)
	if err := ab.publish(tmpFile, backupFile); err != nil {
		return "", err
	}
	return backupFile, nil
}

// exportToTemp writes the export of the site to the temporary file of the backup of the day of now and
// returns its name with the number of comments exported. The export goes to a temporary file first, so a
// failed one leaves an earlier backup of the day in place
func (ab AutoBackup) exportToTemp(ctx context.Context, now time.Time) (tmpFile string, comments int, err error) {
	log.Printf("[DEBUG] make backup for %s", ab.SiteID)
	tmpFile = ab.backupFile(now) + ".tmp"
	if comments, err = ab.writeBackup(ctx, tmpFile); err != nil {
		if e := os.Remove(tmpFile); e != nil && !errors.Is(e, fs.ErrNotExist) {
			log.Printf("[WARN] can't remove %s, %s", tmpFile, e)
		}
		return "", 0, err
	}
	return tmpFile, comments, nil
}

// publish gives the finished temporary export its backup name
func (ab AutoBackup) publish(tmpFile, backupFile string) error {
	if err := os.Rename(tmpFile, backupFile); err != nil {
		return fmt.Errorf("can't rename %s to %s: %w", tmpFile, backupFile, err)
	}
	log.Printf("[DEBUG] created backup file %s", backupFile)
	return nil
}

// writeBackup writes the gzipped export of the site to fname, failing once ctx is done, and returns the
// number of comments exported
func (ab AutoBackup) writeBackup(ctx context.Context, fname string) (comments int, err error) {
	fh, err := os.Create(fname) //nolint:gosec // harmless
	if err != nil {
		return 0, fmt.Errorf("can't create backup file %s: %w", fname, err)
	}
	gz := gzip.NewWriter(fh)

	if comments, err = ab.Exporter.Export(ctxWriter{ctx: ctx, w: gz}, ab.SiteID); err != nil {
		_ = fh.Close()
		return 0, fmt.Errorf("export failed for %s: %w", ab.SiteID, err)
	}
	if err = gz.Close(); err != nil {
		_ = fh.Close()
		return 0, fmt.Errorf("can't close gz for %s: %w", fname, err)
	}
	if err = fh.Close(); err != nil {
		return 0, fmt.Errorf("can't close file handler for %s: %w", fname, err)
	}
	return comments, nil
}

// backupFiles returns the names of the backups of the site, oldest first
func (ab AutoBackup) backupFiles() []string {
	files, err := os.ReadDir(ab.BackupLocation)
	if err != nil {
		log.Printf("[WARN] can't read files in backup directory %s, %s", ab.BackupLocation, err)
		return nil
	}
	var backFiles []string // ReadDir sorts by name, and the date in the name sorts the same way
	for _, file := range files {
		if ab.isBackupFile(file.Name()) {
			backFiles = append(backFiles, file.Name())
		}
	}
	return backFiles
}

// removeOldBackupFiles keeps the newest KeepMax backups of the site and removes the rest
func (ab AutoBackup) removeOldBackupFiles() {
	backFiles := ab.backupFiles()
	for i := 0; i < len(backFiles)-ab.KeepMax; i++ {
		fpath := ab.BackupLocation + "/" + backFiles[i]
		if e := os.Remove(fpath); e != nil {
			log.Printf("[WARN] can't delete %s, %s", fpath, e)
			continue
		}
		log.Printf("[DEBUG] removed %s", fpath)
	}
}

// isBackupFile reports whether name is an auto-backup of this site, backup-<site>-<yyyymmdd>.gz.
// A prefix alone is not enough: backup-radio- is also the start of the backups of site radio-t
func (ab AutoBackup) isBackupFile(name string) bool {
	date, ok := strings.CutPrefix(name, "backup-"+ab.SiteID+"-")
	if !ok {
		return false
	}
	date, ok = strings.CutSuffix(date, ".gz")
	if !ok || len(date) != len("20060102") {
		return false
	}
	for _, c := range date {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ctxWriter fails every write once ctx is done, which stops an export the Exporter can't cancel
type ctxWriter struct {
	ctx context.Context
	w   io.Writer
}

func (cw ctxWriter) Write(p []byte) (int, error) {
	if err := cw.ctx.Err(); err != nil {
		return 0, err
	}
	return cw.w.Write(p)
}
