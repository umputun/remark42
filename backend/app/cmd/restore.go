package cmd

import (
	"time"

	log "github.com/go-pkgz/lgr"
)

// RestoreCommand set of flags and command for restore from backup
type RestoreCommand struct {
	ImportPath string `short:"p" long:"path" env:"BACKUP_PATH" default:"./var/backup" description:"export path"`
	ImportFile string `short:"f" long:"file" default:"userbackup-{{.SITE}}-{{.YYYYMMDD}}.gz" description:"file name" required:"true"`

	Site        string        `short:"s" long:"site" env:"SITE" default:"remark" description:"site name"`
	Timeout     time.Duration `long:"timeout" default:"15m" description:"import timeout"`
	AdminPasswd string        `long:"admin-passwd" env:"ADMIN_PASSWD" required:"true" description:"admin basic auth password"`
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
		InputFile:   fname,
		Site:        rc.Site,
		Provider:    "native",
		Timeout:     rc.Timeout,
		AdminPasswd: rc.AdminPasswd,
		CommonOpts:  rc.CommonOpts,
	}
	return importer.Execute(args)
}
