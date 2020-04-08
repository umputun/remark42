package emailprovider

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"sort"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

type SMTPSender struct {
	SmtpParams SmtpParams
	creator SmtpClientCreator
	Headers map[string]string
	From string
	Subject string
	ContentType string // text/plain or text/html
}

// SmtpParams contain settings for smtp server connection
type SmtpParams struct {
	Host     string        // SMTP host
	Port     int           // SMTP port
	TLS      bool          // TLS auth
	Username string        // user name
	Password string        // password
	TimeOut  time.Duration // TCP connection timeout
}

// default email client implementation
type SmtpCreator struct{ SmtpClientCreator }

// smtpClient interface defines subset of net/smtp used by email client
type SmtpClient interface {
	Mail(string) error
	Auth(smtp.Auth) error
	Rcpt(string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Close() error
}

// smtpClientCreator interface defines function for creating new smtpClients
type SmtpClientCreator interface {
	Create(*SmtpParams) (SmtpClient, error)
}

func NewSMTPSender(params *SmtpParams, creator SmtpClientCreator) EmailSender {
	if params.TimeOut == 0 {
		params.TimeOut = DefaultEmailTimeout
	}
	if creator == nil {
		log.Printf("got empty creator, init default creator now")
		creator = &SmtpCreator{}
	}
	return &SMTPSender{
		SmtpParams: *params,
		creator: creator,
	}
}

func (s *SMTPSender) Name() string {
	return "smtp"
}

func (s *SMTPSender) Send(to, text string) error {
	if message, err := s.BuildMessage(to, text, "text/html"); err != nil {
		return err
	} else {
		return s.sendMessage(to, message)
	}
}

func (s *SMTPSender) AddHeader(header, value string) {
	if s.Headers == nil {
		s.Headers = make(map[string]string)
	}
	s.Headers[header] = value
}

func (s *SMTPSender) ResetHeaders() {
	s.Headers = nil
}

func (s *SMTPSender) SetFrom(from string) {
	s.From = from
}

func (s *SMTPSender) SetSubject(subject string) {
	s.Subject = subject
}

func (s *SMTPSender) SetTimeOut(timeout time.Duration) {
	s.SmtpParams.TimeOut = timeout
}

// buildMessage generates email message to send using net/smtp.Data()
// export BuildMessage for testing purpose
// @TODO seprate emailprovider testing from notify email send testing
func (s *SMTPSender) BuildMessage(to, body, contentType string) (message string, err error) {
	addHeader := func(msg, h, v string) string {
		msg += fmt.Sprintf("%s: %s\n", h, v)
		return msg
	}
	message = addHeader(message, "From", s.From)
	message = addHeader(message, "To", to)
	message = addHeader(message, "Subject", s.Subject)
	message = addHeader(message, "Content-Transfer-Encoding", "quoted-printable")

	if contentType != "" {
		message = addHeader(message, "MIME-version", "1.0")
		message = addHeader(message, "Content-Type", contentType+`; charset="UTF-8"`)
	}

	// https://support.google.com/mail/answer/81126 -> "Include option to unsubscribe"
	if s.Headers != nil && len(s.Headers) > 0{
		keys := make([]string, 0, len(s.Headers))
		for k := range s.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			message = addHeader(message, k, s.Headers[k])
		}
	}

	message = addHeader(message, "Date", time.Now().Format(time.RFC1123Z))

	buff := &bytes.Buffer{}
	qp := quotedprintable.NewWriter(buff)
	if _, err := qp.Write([]byte(body)); err != nil {
		return "", err
	}
	// flush now, must NOT use defer, for small body, defer may cause buff.String() got empty body
	if err := qp.Close(); err != nil {
		return "", fmt.Errorf("quotedprintable Write failed: %w", err)
	}
	m := buff.String()
	message += "\n" + m
	return message, nil
}

// sendMessage sends messages to server in a new connection, closing the connection after finishing.
// Thread safe.
func (s *SMTPSender) sendMessage(to, message string) error {
	if s.creator == nil {
		return errors.New("sendMessage called without smtpCreator set")
	}
	smtpClient, err := s.creator.Create(&s.SmtpParams)
	if err != nil {
		return errors.Wrap(err, "failed to make smtp Create")
	}

	defer func() {
		if err := smtpClient.Quit(); err != nil {
			log.Printf("[WARN] failed to send quit command to %s:%d, %v", s.SmtpParams.Host, s.SmtpParams.Port, err)
			if err := smtpClient.Close(); err != nil {
				log.Printf("[WARN] can't close smtp connection, %v", err)
			}
		}
	}()

	if err := smtpClient.Mail(s.From); err != nil {
		return errors.Wrapf(err, "bad from address %q", s.From)
	}
	if err := smtpClient.Rcpt(to); err != nil {
		return errors.Wrapf(err, "bad to address %q", to)
	}

	writer, err := smtpClient.Data()
	if err != nil {
		return errors.Wrap(err, "can't make email writer")
	}

	defer func() {
		if err = writer.Close(); err != nil {
			log.Printf("[WARN] can't close smtp body writer, %v", err)
		}
	}()

	buf := bytes.NewBufferString(message)
	if _, err = buf.WriteTo(writer); err != nil {
		return errors.Wrapf(err, "failed to send email body to %q", to)
	}

	return nil
}

// String representation of Email object
func (s *SMTPSender) String() string {
	return fmt.Sprintf("emailprovider.sender.smtp: from %q with username '%s' at server %s:%d", s.From, s.SmtpParams.Username, s.SmtpParams.Host, s.SmtpParams.Port)
}

// Create establish SMTP connection with server using credentials in smtpClientWithCreator.SmtpParams
// and returns pointer to it. Thread safe.
func (sc *SmtpCreator) Create(params *SmtpParams) (SmtpClient, error) {
	authenticate := func(c *smtp.Client) error {
		if params.Username == "" || params.Password == "" {
			return nil
		}
		auth := smtp.PlainAuth("", params.Username, params.Password, params.Host)
		if err := c.Auth(auth); err != nil {
			return errors.Wrapf(err, "failed to auth to smtp %s:%d", params.Host, params.Port)
		}
		return nil
	}

	var c *smtp.Client
	srvAddress := fmt.Sprintf("%s:%d", params.Host, params.Port)
	if params.TLS {
		tlsConf := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         params.Host,
		}
		conn, err := tls.Dial("tcp", srvAddress, tlsConf)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to dial smtp tls to %s", srvAddress)
		}
		if c, err = smtp.NewClient(conn, params.Host); err != nil {
			return nil, errors.Wrapf(err, "failed to make smtp client for %s", srvAddress)
		}
		return c, authenticate(c)
	}

	conn, err := net.DialTimeout("tcp", srvAddress, params.TimeOut)
	if err != nil {
		return nil, errors.Wrapf(err, "timeout connecting to %s", srvAddress)
	}

	// if user connect to tls port without tls param enabled, smtp.NewClient will fail after about 300s
	// SetReadDeadline here to quick fail
	if err := conn.SetReadDeadline(time.Now().Add(params.TimeOut)); err != nil {
		return nil, errors.Wrapf(err, "SetReadDeadline failed while connecting to %s", srvAddress)
	}
	c, err = smtp.NewClient(conn, srvAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}

	return c, authenticate(c)
}