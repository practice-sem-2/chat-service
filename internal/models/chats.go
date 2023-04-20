package models

type Chat struct {
	ChatID       string `json:"chat_id" db:"chat_id"`
	MembersCount int    `json:"members_count" db:"members_count"`
	IsDirect     bool   `json:"is_direct" db:"is_direct"`
}

type ChatCreate struct {
	ChatID   string   `json:"chat_id" validate:"required,uuid" db:"chat_id"`
	IsDirect bool     `json:"is_direct" validate:"required" db:"is_direct"`
	Members  []string `json:"members" validate:"required,uuid"`
}

type ChatMember struct {
	UserID string `json:"user_id" db:"user_id"`
}

type ChatWithMembers struct {
	Chat
	Members []ChatMember `json:"members"`
}

type RichChat struct {
	ChatID      string   `json:"chat_id" db:"chat_id"`
	IsDirect    bool     `json:"is_direct" db:"is_direct"`
	LastMessage *Message `json:"last_message"`
}
