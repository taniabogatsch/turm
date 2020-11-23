package app

import (
	"encoding/base64"
	"net/smtp"
	"time"

	"github.com/k3a/html2text"
	"github.com/revel/revel"
)

/*MailerConn contains all Mailer connection fields. */
type MailerConn struct {
	EMail    string
	Server   string
	URL      string
	User     string
	Suffix   string
	Password string
}

/*EMail contains all fields required to send an e-mail. */
type EMail struct {
	Recipient string
	Subject   string
	ReplyTo   string
	Body      string
}

var (
	//EMailQueue holds all currently queued e-mails
	EMailQueue chan EMail

	//Mailer holds all mailer connection data
	Mailer MailerConn
)

func init() {
	EMailQueue = make(chan EMail, 1000)
}

//sendEMails sends e-mails from the queue
type sendEMails struct{}

/*Run is a job that sends one e-mail from the queue. */
func (e sendEMails) Run() {

	select {
	case email := <-EMailQueue:
		mailer(&email)
		revel.AppLog.Debug("sending email", "recipient", email.Recipient,
			"subject", email.Subject, "replyTo", email.ReplyTo)

	case <-time.After(1 * time.Second):
		//no e-mail in queue
	}
}

//mailer sends an e-mail
func mailer(email *EMail) {

	//set the subject and the body
	subjectb64 := base64.StdEncoding.EncodeToString([]byte(email.Subject))
	subjectutf8 := "=?utf-8?B?" + subjectb64 + "?=" //workaround for e-mail servers to not confuse uft-8 encoding in the subject
	msg := "From: " + Mailer.EMail + "\n" +
		"Reply-To: " + email.ReplyTo + "\n" +
		"To: " + email.Recipient + "\n" +
		"Subject: " + subjectutf8 + "\n" +
		"MIME-version: 1.0;\nContent-Type: multipart/alternative; boundary=\"Nldui6qoTs4F=_?:\"\n\n" +
		email.Body

	//localhost -> local e-mail server (postfix)
	c, err := smtp.Dial("127.0.0.1:25")
	if err != nil {
		revel.AppLog.Error("failed dialing localhost", "error", err.Error())
		return
	}
	defer c.Close()

	if err = c.Mail(Mailer.EMail); err != nil {
		revel.AppLog.Error("failed setting the service e-mail as the sender",
			"error", err.Error())
		return
	}

	if err = c.Rcpt(email.Recipient); err != nil {
		revel.AppLog.Error("failed setting the recipient of the e-mail",
			"error", err.Error())
		return
	}

	w, err := c.Data()
	if err != nil {
		revel.AppLog.Error("failed to issue data command to server",
			"error", err.Error())
		return
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		revel.AppLog.Error("failed to write e-mail body", "error", err.Error())
		return
	}

	err = w.Close()
	if err != nil {
		revel.AppLog.Error("failed to close writer", "error", err.Error())
		return
	}
	err = c.Quit()
	if err != nil {
		revel.AppLog.Error("failed to quit client", "error", err.Error())
		return
	}
}

/*SendErrorNote sends an error notification e-mail to the mailer. */
func SendErrorNote() {

	if !revel.DevMode {
		email := EMail{
			Recipient: Mailer.EMail,
			Subject:   "application error",
			ReplyTo:   "",
			Body:      "",
		}
		EMailQueue <- email
	}
}

/*HTMLToMimeFormat takes a HTML string and doubles it into a e-mail body that contains the HTML and a plaintext (divided by a string sequence).
This format of sending a HTML or plaintext e-mail is called MIME format. */
func HTMLToMimeFormat(html *string) (mimeBody string) {

	//HTML2Text takes reader and readability-setting as bool and returns plain text
	//NOTE: text looks pretty dull, but is correct
	plaintext := html2text.HTML2Text(*html)

	mimeBody = "¿This is a multi-part message in MIME format.\n\n--Nldui6qoTs4F=_?:\nContent-Type: text/plain;\n\tcharset=\"utf-8\"\nContent-Transfer-Encoding: 8bit\n\n"
	mimeBody += plaintext
	mimeBody += "\n\n--Nldui6qoTs4F=_?:\nContent-Type: text/html;\n\tcharset=\"utf-8\"\nContent-Transfer-Encoding: 8bit\n\n\n<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\" \"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\"><html xmlns=\"http://www.w3.org/1999/xhtml\"><head><meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" /><title>Individuelle Willkommensmail</title><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\" /></head><body>"
	mimeBody += *html
	mimeBody += "</body></html>\n\n--Nldui6qoTs4F=_?:--"

	return
}

//initMailerData initializes all Mailer config variables
func initMailerData() {

	var found bool
	if Mailer.EMail, found = revel.Config.String("email.email"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "email.email")
	}
	if Mailer.Server, found = revel.Config.String("email.server"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "email.server")
	}
	var port string
	if port, found = revel.Config.String("email.port"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "email.port")
	}
	Mailer.URL = Mailer.Server + ":" + port
	if Mailer.User, found = revel.Config.String("email.user"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "email.user")
	}
	if Mailer.Suffix, found = revel.Config.String("email.suffix"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "email.suffix")
	}
}
