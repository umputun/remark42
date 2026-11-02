package email

import "time"

// Option func type
type Option func(s *Sender)

// SMTP sets SMTP client. Such a client greets the server itself, so HELOHost doesn't apply to it.
func SMTP(smtp SMTPClient) Option {
	return func(s *Sender) {
		s.smtpClient = smtp
	}
}

// Log sets the logger for the email package
func Log(l Logger) Option {
	return func(s *Sender) {
		s.logger = l
	}
}

// Port sets SMTP port
func Port(port int) Option {
	return func(s *Sender) {
		s.port = port
	}
}

// ContentType sets content type of the email
func ContentType(contentType string) Option {
	return func(s *Sender) {
		s.contentType = contentType
	}
}

// Charset sets content charset of the email
func Charset(charset string) Option {
	return func(s *Sender) {
		s.contentCharset = charset
	}
}

// TLS enables TLS support
func TLS(enabled bool) Option {
	return func(s *Sender) {
		s.tls = enabled
	}
}

// STARTTLS enables STARTTLS support
func STARTTLS(enabled bool) Option {
	return func(s *Sender) {
		s.starttls = enabled
	}
}

// InsecureSkipVerify skips certificate verification
func InsecureSkipVerify(enabled bool) Option {
	return func(s *Sender) {
		s.insecureSkipVerify = enabled
	}
}

// HELOHost sets the SMTP HELO/EHLO hostname for connections created by Sender.
// Unset, the greeting stays net/smtp's "localhost". The value is passed to the server as-is,
// so an address literal like "[192.0.2.10]" works, and it has no effect on a client
// supplied with SMTP, which greets on its own.
func HELOHost(host string) Option {
	return func(s *Sender) {
		s.heloHost = host
	}
}

// Auth sets smtp username and password
func Auth(smtpUserName, smtpPasswd string) Option {
	return func(s *Sender) {
		s.smtpUserName = smtpUserName
		s.smtpPassword = smtpPasswd
	}
}

// LoginAuth sets LOGIN auth method
func LoginAuth() Option {
	return func(s *Sender) {
		s.authMethod = authMethodLogin
	}
}

// TimeOut sets smtp timeout
func TimeOut(timeOut time.Duration) Option {
	return func(s *Sender) {
		s.timeOut = timeOut
	}
}
