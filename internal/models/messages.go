package models

import "time"

type FileAttachment struct {
	MimeType string `db:"mime_type"`
	FileID   string `db:"mime_type"`
}

type Message struct {
	MessageID   string    `db:"message_id"`
	FromUser    string    `db:"from_user"`
	ChatID      string    `db:"chat_id"`
	SendingTime time.Time `db:"sending_time"`
	Text        string    `db:"text"`
	ReplyTo     *string   `db:"reply_to"`
	Attachments []FileAttachment
}
