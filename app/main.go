package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"

	"github.com/umputun/remark/app/rest"
)

var opts struct {
	Mongo       []string `short:"m" long:"mongo" env:"MONGO" default:"mongo" description:"mongo host:port" env-delim:","`
	MongoPasswd string   `short:"p" long:"mongo-password" env:"MONGO_PASSWD" default:"" description:"mongo password"`
	MongoDelay  int      `long:"mongo-delay" env:"MONGO_DELAY" default:"0" description:"mongo initial delay"`

	Dbg bool `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "unknown"

func main() {
	fmt.Printf("remark %s\n", revision)
	if _, err := flags.Parse(&opts); err != nil {
		log.Fatal(err)
	}
	setupLog(opts.Dbg)
	log.Print("[INFO] started remark")

	srv := rest.Server{
		Version: revision,
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
