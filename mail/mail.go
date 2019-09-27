package mail

import (
	"github.com/labbsr0x/whisper/web/config"
	"github.com/sirupsen/logrus"
	"net/smtp"
)

type Api interface {
	Init(b *config.WebBuilder, inbox <-chan Mail) Api
	Run()
}

// Mail defines the email
type Mail struct {
	To      []string
	Content []byte
}

type DefaultApi struct {
	user string
	address string
	auth smtp.Auth
	Inbox <-chan Mail
}

// InitFromWebBuilder initializes a default email api instance
func (mh *DefaultApi) Init(b *config.WebBuilder, inbox <-chan Mail) Api {
	mh.user = b.MailUser
	mh.address = b.MailHost + ":" + b.MailPort
	mh.auth = smtp.PlainAuth(b.MailIdentity, b.MailUser, b.MailPassword, b.MailHost)
	mh.Inbox = inbox

	return mh
}

func (mh *DefaultApi) Run() {
	go func(){
		for mail := range mh.Inbox {
			err := smtp.SendMail(mh.address, mh.auth, mh.user, mail.To, mail.Content)

			if err != nil {
				logrus.Error(err)
			}
		}
	}()
}