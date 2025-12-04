package email

import (
	"errors"
	"net/smtp"
)

// authMethod is SMTP authentication method
type authMethod string

// List of supported authentication methods
const (
	authMethodPlain authMethod = "PLAIN"
	authMethodLogin authMethod = "LOGIN"
)

// newLoginAuth returns smtp.Auth that implements the LOGIN authentication
// mechanism as defined in the LOGIN SASL Mechanism document,
// https://www.ietf.org/archive/id/draft-murchison-sasl-login-00.txt.
// The returned smtp.Auth uses the given username and password to authenticate
// to the host.
//
// LOGIN will only send the credentials if the connection is using TLS
// or is connected to localhost. Otherwise authentication will fail with an
// error, without sending the credentials.
//
// LOGIN is described as obsolete in the SASL Mechanisms document
// but the mechanism is still in use, e.g. in Office 365 and Outlook.com.
func newLoginAuth(usr, pwd, host string) smtp.Auth {
	return &loginAuth{usr, pwd, host}
}

type loginAuth struct {
	user     string
	password string
	host     string
}

func isLocalhost(name string) bool {
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (proto string, toServer []byte, err error) {
	if !server.TLS && !isLocalhost(server.Name) {
		return "", nil, errors.New("unencrypted connection")
	}
	if server.Name != a.host {
		return "", nil, errors.New("wrong host name")
	}

	return "LOGIN", []byte(a.user), nil
}

func (a *loginAuth) Next(_ []byte, more bool) (toServer []byte, err error) {
	if more {
		return []byte(a.password), nil
	}

	return nil, nil
}
