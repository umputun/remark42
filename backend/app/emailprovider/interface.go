package emailprovider

import (
	"fmt"
	"time"

	"github.com/go-pkgz/auth/provider"
)

type EmailSender interface {
	provider.Sender // implement for github.com/go-pkgz/auth/provider.VerifyHandler
	fmt.Stringer
	AddHeader(header, value string)
	ResetHeaders()
	SetFrom(from string)
	SetSubject(subject string)
	SetTimeOut(timeout time.Duration)
	Name() string
}