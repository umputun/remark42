package notify

import (
	"context"
	"fmt"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
	"gopkg.in/gomail.v2"
)

// Email implements notify.Destination for email
type Email struct {
	server    string
	port      int
	username  string
	password  string
	keepAlive time.Duration
	ch        chan *gomail.Message
}

const emailConnectionKeepAlive = 30 * time.Second

//NewEmail makes email object for notifications and run sending daemon
func NewEmail(server string, port int, username string, password string, keepAlive time.Duration) (*Email, error) {

	res := Email{
		server:    server,
		port:      port,
		username:  username,
		password:  password,
		keepAlive: keepAlive,
		ch:        make(chan *gomail.Message),
	}

	if res.keepAlive == 0 {
		res.keepAlive = emailConnectionKeepAlive
	}

	log.Printf(
		"[DEBUG] create new email notifier for server %s with user %s, keepalive=%s",
		res.server, res.username, res.keepAlive)

	// test connection before starting a daemon
	tmpConn, err := gomail.NewDialer(res.server, res.port, res.username, res.password).Dial()
	if err != nil {
		return &res, errors.Errorf(
			"[WARN] error connecting to '%s':%d with username '%s': %s",
			res.server, res.port, res.username, err)
	}
	err = tmpConn.Close()
	if err != nil {
		return &res, errors.Errorf(
			"[WARN] error closing connection to %s:%d: %s",
			res.server, res.port, err)
	}

	go func() {
		d := gomail.NewDialer(res.server, res.port, res.username, res.password)

		var s gomail.SendCloser
		var err error
		open := false
		for {
			select {
			case m, ok := <-res.ch:
				if !ok {
					return
				}
				if !open {
					if s, err = d.Dial(); err != nil {
						log.Printf(
							"[WARN] error connecting to %s:%d with username %s: %s",
							res.server, res.port, res.username, err)
					}
					open = true
				}
				if err := gomail.Send(s, m); err != nil {
					log.Printf(
						"[INFO] error sending to %s:%d with username %s: %s",
						res.server, res.port, res.username, err)
				}
			// Close the connection to the SMTP server if no email was sent in the keepAlive period.
			case <-time.After(res.keepAlive):
				if open {
					if err := s.Close(); err != nil {
						log.Printf(
							"[WARN] error closing connection with %s:%d with username %s: %s",
							res.server, res.port, res.username, err)
					}
					open = false
				}
			}
		}
	}()

	// TODO: do we need to close this?
	// Close the channel to stop the mail daemon.
	//close(res.ch)

	return &res, nil
}

// Send email
func (e *Email) Send(ctx context.Context, req request) error {
	log.Printf("[DEBUG] send notification via %s, comment id %s", e, req.comment.ID)
	// TODO: write actual code
	return nil
}

func (e *Email) String() string {
	return fmt.Sprintf("email: '%s'@'%s':%d", e.username, e.server, e.port)
}
