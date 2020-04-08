package emailprovider

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test_SMTPSend(t *testing.T) {
	fromEmail := os.Getenv("SMTP_FROM")
	toEmail := os.Getenv("SMTP_TO")
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	useTls, _ := strconv.ParseBool(os.Getenv("SMTP_TLS"))

	sndr := NewSMTPSender(&SmtpParams{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     port,
		TLS:      useTls,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		TimeOut:  3 * time.Second,
	}, nil)

	subject := "Sending with SMTP is Not safe"

	sndr.SetSubject(subject)
	sndr.SetFrom(fromEmail)

	htmlContent := "<strong>and easy to do anywhere, even with Go</strong>"
	t.Logf("try send via SMTP from %s to %s", fromEmail, toEmail)

	msg, err := sndr.(*SMTPSender).BuildMessage(toEmail, htmlContent, "text/html")
	if err != nil {
		t.Error(err)
	}
	t.Logf("mail msg: %s", msg)
	if !strings.Contains(msg, htmlContent) {
		t.Errorf("BuildMessage lost body")
	}
	err = sndr.Send(toEmail, htmlContent)
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("mail send sucess")
	}
}