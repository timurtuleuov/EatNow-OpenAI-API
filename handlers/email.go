package handlers

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/spf13/viper"
)

type EmailSender struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func NewEmailSender() *EmailSender {
	return &EmailSender{
		Host:     viper.GetString("smtp.host"),
		Port:     viper.GetInt("smtp.port"),
		Username: viper.GetString("smtp.username"),
		Password: viper.GetString("smtp.password"),
		From:     viper.GetString("smtp.from"),
	}
}

func (e *EmailSender) fromAddress() string {
	if i := strings.Index(e.From, "<"); i >= 0 {
		if j := strings.LastIndex(e.From, ">"); j > i {
			return e.From[i+1 : j]
		}
	}
	return e.From
}

func (e *EmailSender) Send(to, subject, body string) error {
	if e.Username == "" || e.Password == "" {
		return nil
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		e.From, to, subject, body)

	addr := fmt.Sprintf("%s:%d", e.Host, e.Port)
	auth := smtp.PlainAuth("", e.Username, e.Password, e.Host)

	return smtp.SendMail(addr, auth, e.fromAddress(), []string{to}, []byte(msg))
}
