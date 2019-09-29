package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

// RemapCommand set of flags and command for change linkage between comments to
// different urls based on given rules (input file)
type RemapCommand struct {
	Site        string        `short:"s" long:"site" env:"SITE" default:"remark" description:"site name"`
	InputFile   string        `short:"f" long:"file" description:"input file name" required:"true"`
	AdminPasswd string        `long:"admin-passwd" env:"ADMIN_PASSWD" required:"true" description:"admin basic auth password"`
	Timeout     time.Duration `long:"timeout" default:"15m" description:"remap timeout"`
	CommonOpts
}

func (rc *RemapCommand) Execute(args []string) error {
	log.Printf("[INFO] start remap, site %s, file with rules %s", rc.Site, rc.InputFile)
	resetEnv("SECRET", "ADMIN_PASSWD")

	rulesReader, err := os.Open(rc.InputFile)
	if err != nil {
		return errors.Wrapf(err, "cant open file %s", rc.InputFile)
	}

	client := http.Client{}
	ctx, cancel := context.WithTimeout(context.Background(), rc.Timeout)
	defer cancel()
	importURL := fmt.Sprintf("%s/api/v1/admin/remap?site=%s", rc.RemarkURL, rc.Site)
	req, err := http.NewRequest(http.MethodPost, importURL, rulesReader)
	if err != nil {
		return errors.Wrapf(err, "can't make remap request for %s", importURL)
	}
	req.SetBasicAuth("admin", rc.AdminPasswd)

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return errors.Wrapf(err, "request failed for %s", importURL)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("[WARN] failed to close response, %s", err)
		}
	}()
	if resp.StatusCode >= 300 {
		return responseError(resp)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "can't get response")
	}

	log.Printf("[INFO] completed, status=%d, %s", resp.StatusCode, string(body))
	return nil
}
