// using SendGrid's Go Library
// https://github.com/sendgrid/sendgrid-go
package emailprovider

import (
	"os"
	"testing"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func Test_SendgridSend(t *testing.T) {
	fromEmail := os.Getenv("SENDGRID_FROM")
	toEmail := os.Getenv("SENDGRID_TO")
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))

	from := mail.NewEmail("Example User", fromEmail)
	subject := "Sending with SendGrid is Fun"
	to := mail.NewEmail("Example User", toEmail)
	plainTextContent := "and easy to do anywhere, even with Go"
	htmlContent := "<strong>and easy to do anywhere, even with Go</strong>"
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	t.Logf("try send from %s to %s", fromEmail, toEmail)
	response, err := client.Send(message)
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("StatusCode: %#v, Body: %#v, Headers: %#v", response.StatusCode, response.Body, response.Headers)
	}
}