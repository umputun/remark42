package migrator

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
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

// Do runs daily export to local files, keeps up to keepMax backups for given siteID
func (ab AutoBackup) Do(ctx context.Context) {
	log.Printf("[INFO] activate auto-backup for %s under %s, duration %s", ab.SiteID, ab.BackupLocation, ab.Duration)
	tick := time.NewTicker(ab.Duration)
	defer tick.Stop()
	log.Printf("[DEBUG] first backup for %s at %s", ab.SiteID, time.Now().Add(ab.Duration))

	for {
		select {
		case <-tick.C:
			if _, err := ab.makeBackup(); err != nil {
				log.Printf("[WARN] auto-backup for %s failed, %s", ab.SiteID, err)
				continue
			}
			ab.removeOldBackupFiles()
			log.Printf("[DEBUG] next backup for %s at %s", ab.SiteID, time.Now().Add(ab.Duration))
		case <-ctx.Done():
			log.Printf("[WARN] terminated autobackup for %s", ab.SiteID)
			return
		}
	}
}

func (ab AutoBackup) makeBackup() (string, error) {
	log.Printf("[DEBUG] make backup for %s", ab.SiteID)
	backupFile := fmt.Sprintf("%s/backup-%s-%s.gz", ab.BackupLocation, ab.SiteID, time.Now().Format("20060102"))
	// the export goes to a temporary file first, so a failed one leaves an earlier backup of the day in place
	tmpFile := backupFile + ".tmp"
	if err := ab.writeBackup(tmpFile); err != nil {
		if e := os.Remove(tmpFile); e != nil && !errors.Is(e, fs.ErrNotExist) {
			log.Printf("[WARN] can't remove %s, %s", tmpFile, e)
		}
		return "", err
	}
	if err := os.Rename(tmpFile, backupFile); err != nil {
		return "", fmt.Errorf("can't rename %s to %s: %w", tmpFile, backupFile, err)
	}
	log.Printf("[DEBUG] created backup file %s", backupFile)
	return backupFile, nil
}

// writeBackup writes the gzipped export of the site to fname
func (ab AutoBackup) writeBackup(fname string) error {
	fh, err := os.Create(fname) //nolint:gosec // harmless
	if err != nil {
		return fmt.Errorf("can't create backup file %s: %w", fname, err)
	}
	gz := gzip.NewWriter(fh)

	if _, err = ab.Exporter.Export(gz, ab.SiteID); err != nil {
		_ = fh.Close()
		return fmt.Errorf("export failed for %s: %w", ab.SiteID, err)
	}
	if err = gz.Close(); err != nil {
		_ = fh.Close()
		return fmt.Errorf("can't close gz for %s: %w", fname, err)
	}
	if err = fh.Close(); err != nil {
		return fmt.Errorf("can't close file handler for %s: %w", fname, err)
	}
	return nil
}

// removeOldBackupFiles keeps the newest KeepMax backups of the site and removes the rest
func (ab AutoBackup) removeOldBackupFiles() {
	files, err := os.ReadDir(ab.BackupLocation)
	if err != nil {
		log.Printf("[WARN] can't read files in backup directory %s, %s", ab.BackupLocation, err)
		return
	}
	var backFiles []string // ReadDir sorts by name, and the date in the name sorts the same way
	for _, file := range files {
		if ab.isBackupFile(file.Name()) {
			backFiles = append(backFiles, file.Name())
		}
	}

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
