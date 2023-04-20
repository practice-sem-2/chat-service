package models

import "time"

type FileAttachment struct {
	MimeType string `validate:"required" db:"mime_type"`
	FileID   string `validate:"required,uuid" db:"file_id"`
}

type MessageSend struct {
	MessageID   string           `validate:"required,uuid"`
	ChatID      string           `validate:"required,uuid"`
	Text        string           `validate:"max=2048,required_without=Attachments"`
	ReplyTo     *string          `validate:"omitempty,uuid"`
	Attachments []FileAttachment `validate:"required_without=Text"`
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

type MessagesSelect struct {
	ChatID string     `validate:"required,uuid"`
	Count  *int       `validate:"omitempty,min=0,max=512"`
	Since  *time.Time `validate:"omitempty"`
	Until  *time.Time `validate:"omitempty"`
}
