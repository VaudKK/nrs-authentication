package mailer

import (
	"bytes"
	"context"
	"embed"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
	"github.com/resend/resend-go/v3"
	"github.com/sirupsen/logrus"
)

//go:embed "templates"
var templateFS embed.FS // embedded file system

// Define a Mailer struct which contains a mail.Dialer instance (used to connect to a
// SMTP server) and the sender information for your emails (the name and address you
// want the email to be from, such as "Alice Smith <alice@example.com>").
type Mailer struct {
	dialer *mail.Dialer
	sender string
	log    *logrus.Logger
}

func New(host string, port int, username, password, sender string, log *logrus.Logger) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return Mailer{
		dialer: dialer,
		sender: sender,
		log:    log,
	}
}

func (m *Mailer) Send(recipient, templateFile string, data interface{}, useSmtp bool) error {
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

	// Use the mail.NewMessage() function to initialize a new mail.Message instance.
	// Then we use the SetHeader() method to set the email recipient, sender and subject
	// headers, the SetBody() method to set the plain-text body, and the AddAlternative()
	// method to set the HTML body. It's important to note that AddAlternative() should
	// always be called *after* SetBody().
	msg := mail.NewMessage()

	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())
	msg.Embed("internal/mailer/templates/images/logo.png")

	// Call the DialAndSend() method on the dialer, passing in the message to send. This
	// opens a connection to the SMTP server, sends the message, then closes the
	// connection. If there is a timeout, it will return a "dial tcp: i/o timeout"
	// error.
	if useSmtp {
		err = m.dialer.DialAndSend(msg)
		if err != nil {
			return err

		}
	} else {
		err = sendViaHttp(msg, htmlBody.String(), m.dialer.Password, m.log)
		if err != nil {
			return err
		}
	}

	return nil

}

func sendViaHttp(message *mail.Message, htmlBody, clientKey string, log *logrus.Logger) error {
	ctx := context.TODO()
	client := resend.NewClient(clientKey)

	params := &resend.SendEmailRequest{
		From:    message.GetHeader("From")[0],
		To:      message.GetHeader("To"),
		Subject: message.GetHeader("Subject")[0],
		Html:    htmlBody,
		ReplyTo: "noreply@mail.kcsda.or.ke",
	}

	sent, err := client.Emails.SendWithContext(ctx, params)

	if err != nil {
		log.WithError(err).Error("Error while sending mail via http")
		return err
	}

	log.Info("Sent mail with send id " + sent.Id)
	return nil
}
