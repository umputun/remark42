package cmd

import (
	"time"

	log "github.com/go-pkgz/lgr"
)

// RestoreCommand set of flags and command for restore from backup
type RestoreCommand struct {
	ImportPath string `short:"p" long:"path" env:"BACKUP_PATH" default:"./var/backup" description:"export path"`
	ImportFile string `short:"f" long:"file" default:"userbackup-{{.SITE}}-{{.YYYYMMDD}}.gz" description:"file name" required:"true"`

	SupportCmdOpts
	CommonOpts
}

// Execute runs import with RestoreCommand parameters, entry point for "restore" command
// uses ImportCommand with constructed full file name
func (rc *RestoreCommand) Execute(args []string) error {
	log.Printf("[INFO] restore %s, site %s", rc.ImportFile, rc.Site)
	resetEnv("SECRET", "ADMIN_PASSWD")

	fp := fileParser{site: rc.Site, path: rc.ImportPath, file: rc.ImportFile}
	fname, err := fp.parse(time.Now())
	if err != nil {
		return err
	}
	importer := ImportCommand{
		InputFile:      fname,
		Provider:       "native",
		SupportCmdOpts: rc.SupportCmdOpts,
		CommonOpts:     rc.CommonOpts,
	}
	return importer.Execute(args)
}
