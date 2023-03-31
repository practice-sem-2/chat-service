package models

type Chat struct {
	ChatID string `json:"chat_id" db:"chat_id"`
}

type ChatMember struct {
	UserID string `json:"user_id" db:"user_id"`
}

type ChatWithMembers struct {
	Chat
	Members []ChatMember `json:"members"`
}
