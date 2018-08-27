package cmd

import (
	"log"
	"time"
)

// RestoreCommand set of flags and command for restore from backup
type RestoreCommand struct {
	ImportPath string `short:"p" long:"path" env:"BACKUP_PATH" default:"./var/backup" description:"export path"`
	ImportFile string `short:"f" long:"file" default:"userbackup-{{.SITE}}-{{.YYYYMMDD}}.gz" description:"file name" required:"true"`

	Site         string        `long:"site" env:"SITE" default:"remark" description:"site name"`
	SharedSecret string        `long:"secret" env:"SECRET" description:"shared secret key" required:"true"`
	Timeout      time.Duration `long:"timeout" default:"15m" description:"import timeout"`
	URL          string        `long:"url" default:"http://127.0.0.1:8081" description:"migrator base url"`
}

// Execute runs import with RestoreCommand parameters, entry point for "restore" command
// uses ImportCommand with constructed full file name
func (rc *RestoreCommand) Execute(args []string) error {
	log.Printf("[INFO] restore %s, site %s", rc.ImportFile, rc.Site)
	resetEnv("SECRET")

	fp := fileParser{site: rc.Site, path: rc.ImportPath, file: rc.ImportFile}
	fname, err := fp.parse(time.Now())
	if err != nil {
		return err
	}
	importer := ImportCommand{
		InputFile:    fname,
		Site:         rc.Site,
		Provider:     "native",
		SharedSecret: rc.SharedSecret,
		Timeout:      rc.Timeout,
		URL:          rc.URL,
	}
	return importer.Execute(args)
}
