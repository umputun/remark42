/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package main

import (
	"fmt"
	"os"

	log "github.com/go-pkgz/lgr"
	"github.com/jessevdk/go-flags"

	"github.com/umputun/remark/backend/app/rpc"

	"github.com/umputun/remark/backend/_example/memory_store/accessor"
	"github.com/umputun/remark/backend/_example/memory_store/server"
)

// opts with all cli commands and flags
var opts struct {
	API        string `long:"api" env:"API" default:"/" description:"api root url"`
	Port       int    `long:"port" env:"PORT" default:"8080" description:"rpc server port"`
	AuthUser   string `long:"auth-user" env:"AUTH_USER" default:"" description:"rpc auth user name"`
	AuthPasswd string `long:"auth-passwd" env:"AUTH_PASSWD" default:"" description:"rpc auth password"`

	Secret string `long:"secret" env:"SECRET" required:"true" description:"secret key"`
	Dbg    bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "unknown"

func main() {
	fmt.Printf("remark42-memory module %s\n", revision)

	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(2)
	}
	setupLog(opts.Dbg)

	dataStore := accessor.NewMemData()
	adminStore := accessor.NewMemAdminStore(opts.Secret)

	rpcServer := rpc.Server{
		API:        opts.API,
		AuthUser:   opts.AuthUser,
		AuthPasswd: opts.AuthPasswd,
		Version:    revision,
		AppName:    "remark42-memory",
	}
	srv := server.NewRPC(dataStore, adminStore, &rpcServer)

	admRec := accessor.AdminRec{
		SiteID: "example",
		IDs:    []string{"id1", "id2"},
		Email:  "admin@example.com",
	}
	adminStore.Set("example", admRec)

	err := srv.Run(opts.Port)
	log.Printf("[ERROR] server failed or terminated, %+v", err)
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
