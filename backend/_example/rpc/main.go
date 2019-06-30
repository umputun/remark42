/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package main

// Opts with all cli commands and flags
var opts struct {
	API        string `long:"api" env:"API" default:"/" description:"api root url"`
	Port       int    `long:"port" env:"PORT" default:"8080" description:"rpc server port"`
	AuthUser   string `long:"auth-user" env:"AUTH_USER" default:"" description:"auth user name"`
	AuthPasswd string `long:"auth-passwd" env:"AUTH_PASSWD" default:"" description:"auth password"`
	Secret     string `long:"secret" env:"SECRET" required:"true" description:"secret key"`
	Dbg        bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
}


func main() {
	
}
