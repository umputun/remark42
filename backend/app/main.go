package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/hashicorp/logutils"
	flags "github.com/jessevdk/go-flags"

	"github.com/umputun/remark/backend/app/cmd"
)

// Opts with all cli commands and flags
type Opts struct {
	ServerCmd  cmd.ServerCommand  `command:"server"`
	ImportCmd  cmd.ImportCommand  `command:"import"`
	BackupCmd  cmd.BackupCommand  `command:"backup"`
	RestoreCmd cmd.RestoreCommand `command:"restore"`
	AvatarCmd  cmd.AvatarCommand  `command:"avatar"`
	CleanupCmd cmd.CleanupCommand `command:"cleanup"`

	RemarkURL    string `long:"url" env:"REMARK_URL" required:"true" description:"url to remark"`
	SharedSecret string `long:"secret" env:"SECRET" required:"true" description:"shared secret key"`

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
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stdout,
	}

	log.SetFlags(log.Ldate | log.Ltime)

	if dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		filter.MinLevel = logutils.LogLevel("DEBUG")
	}
	log.SetOutput(filter)
}

func init() {
	// catch SIGQUIT and print stack traces
	sigChan := make(chan os.Signal)
	go func() {
		for range sigChan {
			log.Print("[INFO] SIGQUIT detected")
			maxSize := 5 * 1024 * 1024
			stacktrace := make([]byte, maxSize)
			length := runtime.Stack(stacktrace, true)
			if length > maxSize {
				length = maxSize
			}
			fmt.Println(string(stacktrace[:length]))
		}
	}()
	signal.Notify(sigChan, syscall.SIGQUIT)
}
