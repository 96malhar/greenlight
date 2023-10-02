package email

import (
	"bytes"
	"embed"
	"github.com/go-mail/mail/v2"
	"html/template"
	"time"
)

//go:embed "templates"
var templateFS embed.FS

type MailerInterface interface {
	Send(recipient, templateFile string, data any) error
}

// Mailer represents a mailer that can be used to send emails.
type Mailer struct {
	dialer *mail.Dialer
	sender string
}

// NewMailer returns a new Mailer instance, containing a default mail.Dialer and the sender information.
func NewMailer(host string, port int, username, password, sender string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

// Send takes the recipient email address ,the email template filename, and any
// dynamic data for the templates as an any parameter.
func (m Mailer) Send(recipient, templateFile string, data any) error {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	for i := 1; i <= 3; i++ {
		err = m.dialer.DialAndSend(msg)
		if nil == err {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return err
}
