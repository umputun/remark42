package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	log "github.com/go-pkgz/lgr"
	"github.com/umputun/go-flags"

	"github.com/umputun/remark42/backend/app/cmd"
)

// Opts with all cli commands and flags
type Opts struct {
	ServerCmd  cmd.ServerCommand  `command:"server"`
	ImportCmd  cmd.ImportCommand  `command:"import"`
	BackupCmd  cmd.BackupCommand  `command:"backup"`
	RestoreCmd cmd.RestoreCommand `command:"restore"`
	AvatarCmd  cmd.AvatarCommand  `command:"avatar"`
	CleanupCmd cmd.CleanupCommand `command:"cleanup"`
	RemapCmd   cmd.RemapCommand   `command:"remap"`

	RemarkURL    string `long:"url" env:"REMARK_URL" required:"true" description:"url to remark"`
	SharedSecret string `long:"secret" env:"SECRET" required:"true" description:"shared secret key used to sign JWT, should be a random, long, hard-to-guess string"`

	Dbg bool `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "unknown"

func main() {
	fmt.Printf("remark42 %s\n", revision)

	var opts Opts
	p := flags.NewParser(&opts, flags.Default)
	p.CommandHandler = func(command flags.Commander, args []string) error {
		setupLog(opts.Dbg)
		// commands implements CommonOptionsCommander to allow passing set of extra options defined for all commands
		c := command.(cmd.CommonOptionsCommander)
		c.SetCommon(cmd.CommonOpts{
			RemarkURL:    opts.RemarkURL,
			SharedSecret: opts.SharedSecret,
			Revision:     revision,
		})
		for _, entry := range c.HandleDeprecatedFlags() {
			deprecationNote := fmt.Sprintf("[WARN] --%s is deprecated since v%s and will be removed in the future", entry.Old, entry.Version)
			if entry.New != "" {
				deprecationNote += fmt.Sprintf(", please use --%s instead", entry.New)
			}
			log.Print(deprecationNote)
		}
		err := c.Execute(args)
		if err != nil {
			log.Printf("[ERROR] failed with %+v", err)
		}
		return err
	}

	if _, err := p.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}

// getDump reads runtime stack and returns as a string
func getDump() string {
	maxSize := 5 * 1024 * 1024
	stacktrace := make([]byte, maxSize)
	length := runtime.Stack(stacktrace, true)
	if length > maxSize {
		length = maxSize
	}
	return string(stacktrace[:length])
}

// nolint:gochecknoinits // can't avoid it in this place
func init() {
	// catch SIGQUIT and print stack traces
	sigChan := make(chan os.Signal)
	go func() {
		for range sigChan {
			log.Printf("[INFO] SIGQUIT detected, dump:\n%s", getDump())
		}
	}()
	signal.Notify(sigChan, syscall.SIGQUIT)
}
