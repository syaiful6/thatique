package mailer

import (
	"github.com/emersion/go-message"
)

type JobMail struct {
	t        Transport
	messages []*message.Entity
}

func NewJobMail(t Transport, messages []*message.Entity) *JobMail {
	return &JobMail{t: t, messages: messages}
}

func (j *JobMail) GetName() string {
	return "JobMail"
}

func (j *JobMail) Fire() error {
	err := j.t.Open()
	if err != nil {
		return err
	}

	defer j.t.Close()

	_, err = j.t.SendMessages(j.messages)
	return err
}
