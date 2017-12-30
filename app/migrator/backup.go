package migrator

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Do runs daily export to local files, keeps up to keepMax backups for given siteID
func (ab AutoBackup) Do() {
	log.Print("[INFO] activate auto-backup")
	tick := time.NewTicker(24 * time.Hour)
	for range tick.C {
		_, err := ab.makeBackup()
		if err != nil {
			log.Printf("[WARN] auto-backup for %s failed, %s", ab.SiteID, err)
			continue
		}
		ab.removeOldBackupFiles()
	}
}

func (ab AutoBackup) makeBackup() (string, error) {
	log.Printf("[DEBUG] make backup for %s", ab.SiteID)
	backupFile := fmt.Sprintf("%s/backup-%s-%s.gz", ab.BackupLocation, ab.SiteID, time.Now().Format("20060102"))
	fh, err := os.Create(backupFile)
	if err != nil {
		return "", errors.Wrapf(err, "can't create backup file %s", backupFile)
	}
	gz := gzip.NewWriter(fh)

	if err = ab.Exporter.Export(gz, ab.SiteID); err != nil {
		return "", errors.Wrapf(err, "export failed for %s", ab.SiteID)
	}
	if err = gz.Close(); err != nil {
		return "", errors.Wrapf(err, "can't close gz for %s", backupFile)
	}
	if err = fh.Close(); err != nil {
		return "", errors.Wrapf(err, "can't close file handler for %s", backupFile)
	}
	log.Printf("[DEBUG] created backup file %s", backupFile)
	return backupFile, nil
}

func (ab AutoBackup) removeOldBackupFiles() {
	files, err := ioutil.ReadDir(ab.BackupLocation)
	if err != nil {
		log.Printf("[WARN] can't read files in backup directory %s, %s", ab.BackupLocation, err)
		return
	}
	backFiles := []os.FileInfo{}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "backup-"+ab.SiteID) {
			backFiles = append(backFiles, file)
		}
	}
	sort.Slice(backFiles, func(i int, j int) bool { return backFiles[i].Name() < backFiles[j].Name() })

	if len(backFiles) > ab.KeepMax {
		for i := 0; i < len(backFiles)-ab.KeepMax; i++ {
			fpath := ab.BackupLocation + "/" + backFiles[i].Name()
			if e := os.Remove(fpath); e != nil {
				log.Printf("[WARN] can't delete %s, %s", fpath, err)
				continue
			}
			log.Printf("[DEBUG] removed %s", fpath)
		}
	}
}
