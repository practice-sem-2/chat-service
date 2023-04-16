package models

import "time"

type UpdateMeta struct {
	Timestamp time.Time
	Audience  []string
}

type MessageSent struct {
	UpdateMeta
	MessageID   string  `validate:"required,uuid"`
	FromUser    string  `validate:"required"`
	ChatID      string  `validate:"required,uuid"`
	Text        string  `validate:"required_without=Attachments"`
	ReplyTo     *string `validate:"uuid"`
	Attachments []FileAttachment
}

type ChatCreated struct {
	UpdateMeta
	ChatID   string `validate:"required,uuid"`
	IsDirect bool   `validate:"required"`
	Members  []string
}

type MemberAdded struct {
	UpdateMeta
	ChatID   string `validate:"required,uuid"`
	Username string `validate:"required"`
}

type MemberRemoved struct {
	UpdateMeta
	ChatID   string `validate:"required,uuid"`
	Username string `validate:"required"`
}
