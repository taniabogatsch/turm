package app

import (
	"encoding/base64"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/k3a/html2text"
	"github.com/revel/revel"
)

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
)

func init() {
	EMailQueue = make(chan EMail, 1000)
}

//initEMailTemplates parses all e-mail template names
func initEMailTemplates() {

	//TODO: is this function necessary?
	folder := filepath.Join(revel.BasePath, "app", "views", "EMails")
	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, "_") {
			EMailTemplates = append(EMailTemplates, filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		revel.AppLog.Fatal("error parsing e-mail templates", "error", err.Error())
	}
}

//sendEMails sends e-mails from the queue.
type sendEMails struct{}

/*Run is a job that sends one e-mail from the queue. */
func (e sendEMails) Run() {
	email := <-EMailQueue
	mailer(&email)
	revel.AppLog.Debug("sending email", "recipient", email.Recipient,
		"subject", email.Subject, "replyTo", email.ReplyTo)

	//wait before sending the next e-mail
	time.Sleep(time.Second * 15) //necessary to not spam the e-mail server too much
}

/*AddEMailToQueue adds an e-mail to the e-mail queue. */
func AddEMailToQueue(email *EMail) {
	EMailQueue <- *email
}

//mailer sends an e-mail.
func mailer(email *EMail) {

	//NOTE: receipts must look like this: []string{"some.mail@tu-ilmenau.de", "second.mail@web.de"}

	//set up authentication information
	auth := smtp.PlainAuth("", EMailUser, Passwords["email.pw"], EMailServer)

	//connect to the server, authenticate, set the sender and recipient and send the e-mail
	subjectb64 := base64.StdEncoding.EncodeToString([]byte(email.Subject))
	subjectutf8 := "=?utf-8?B?" + subjectb64 + "?=" //workaround for e-mail servers to not confuse uft-8 encoding in the subject
	msg := "From: " + ServiceEMail + "\n" +
		"Reply-To: " + email.ReplyTo + "\n" +
		"To: " + email.Recipient + "\n" +
		"Subject: " + subjectutf8 + "\n" +
		"MIME-version: 1.0;\nContent-Type: multipart/alternative; boundary=\"Nldui6qoTs4F=_?:\"\n\n" +
		email.Body

	err := smtp.SendMail(EMailURL, auth, ServiceEMail, []string{email.Recipient}, []byte(msg))
	if err != nil {
		revel.AppLog.Error("error sending e-mail", "recipient", email.Recipient,
			"subject", email.Subject, "replyTo", email.ReplyTo, "error", err.Error())
	}
	return
}

/*HTMLToMimeFormat takes a HTML string and doubles it into a e-mail body that contains the HTML and a plaintext (divided by a string sequence).
This format of sending a HTML or plaintext e-mail is called MIME format. */
func HTMLToMimeFormat(html *string) (mimeBody string) {
	//HTML2Text takes reader and readability-setting as bool and returns plain text
	//NOTE: text looks pretty dull, but is correct
	plaintext := html2text.HTML2Text(*html)

	mimeBody = ""
	mimeBody = "¿This is a multi-part message in MIME format.\n\n--Nldui6qoTs4F=_?:\nContent-Type: text/plain;\n\tcharset=\"utf-8\"\nContent-Transfer-Encoding: 8bit\n\n"
	mimeBody += plaintext
	mimeBody += "\n\n--Nldui6qoTs4F=_?:\nContent-Type: text/html;\n\tcharset=\"utf-8\"\nContent-Transfer-Encoding: 8bit\n\n\n<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\" \"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\"><html xmlns=\"http://www.w3.org/1999/xhtml\"><head><meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" /><title>Individuelle Willkommensmail</title><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\" /></head><body>"
	mimeBody += *html
	mimeBody += "</body></html>\n\n--Nldui6qoTs4F=_?:--"

	return mimeBody
}
