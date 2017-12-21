package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
	"github.com/umputun/remark/app/store"

	"github.com/umputun/remark/app/rest"
)

var opts struct {
	DBFile string `long:"db" env:"BOLTDB_FILE" default:"/tmp/remark.db" description:"bolt file name"`
	Dbg    bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "unknown"

func main() {
	fmt.Printf("remark %s\n", revision)
	if _, err := flags.Parse(&opts); err != nil {
		log.Fatal(err)
	}
	setupLog(opts.Dbg)
	log.Print("[INFO] started remark")

	dataStore, err := store.NewBoltDB(opts.DBFile)
	if err != nil {
		log.Fatalf("[ERROR] can't initialize data store, %+v", err)
	}

	srv := rest.Server{
		Version: revision,
		Store:   dataStore,
	}
	srv.Run()
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
