// SendGrid(https://sendgrid.com) Trial Plan provides 40,000 emails for 30 days
// After your trial ends, you can send 100 emails/day for free

package emailprovider

import (
	"fmt"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// MailgunConfig contain settings for mailgun API
type SendgridSender struct {
	sg		   *sendgrid.Client
	APIKey 	string // the SendGrid API key
	TimeOut  time.Duration // TCP connection timeout
	From string
	Subject string
	Headers map[string]string
	ContentType string // text/plain or text/html
}

func NewSendgridSender(APIKey string, TimeOut time.Duration) EmailSender {
	if TimeOut == 0 {
		TimeOut = DefaultEmailTimeout
	}
	sender := &SendgridSender {
		APIKey: APIKey,
		TimeOut: TimeOut,
	}

	// Create an instance of the sendgrid Client
	sender.sg = sendgrid.NewSendClient(APIKey)
	return sender
}

func (s *SendgridSender) Name() string {
	return "sendgrid"
}

func (s *SendgridSender) Send(to, text string) error {
	fromEmail := mail.NewEmail("", s.From)
	toEmail := mail.NewEmail("", to)
	sgmail := mail.NewSingleEmail(fromEmail, s.Subject, toEmail, text, text)

	// extra headers used mainly for List-Unsubscribe feature
	// see more info via https://sendgrid.com/docs/ui/sending-email/list-unsubscribe/
	if s.Headers != nil && len(s.Headers) > 0{
		sgmail.Headers = s.Headers
	}
	// Send the message	with a 10 second timeout
	sendgrid.DefaultClient.HTTPClient.Timeout = s.TimeOut
	resp, err := s.sg.Send(sgmail)
	if err != nil {
		return fmt.Errorf("sendgrid: send failed: %w", err)
	}
	fmt.Printf("sendgrid: send to %s success, StatusCode: %d\n", to, resp.StatusCode)
	return nil
}

func (s *SendgridSender) AddHeader(header, value string) {
	if s.Headers == nil {
		s.Headers = make(map[string]string)
	}
	s.Headers[header] = value
}

func (s *SendgridSender) ResetHeaders() {
	s.Headers = nil
}

func (s *SendgridSender) SetFrom(from string) {
	s.From = from
}

func (s *SendgridSender) SetSubject(subject string) {
	s.Subject = subject
}

func (s *SendgridSender) SetTimeOut(timeout time.Duration) {
	s.TimeOut = timeout
	sendgrid.DefaultClient.HTTPClient.Timeout = s.TimeOut
}

// String representation of Email object
func (s *SendgridSender) String() string {
	return fmt.Sprintf("emailprovider.sender.sendgrid: API %s", "v3")
}