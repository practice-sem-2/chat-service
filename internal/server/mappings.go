package server

import (
	"github.com/practice-sem-2/user-service/internal/models"
	"github.com/practice-sem-2/user-service/internal/pb/chats"
	"time"
)

func AttachmentToModel(a *chats.FileAttachment) *models.FileAttachment {
	return &models.FileAttachment{
		MimeType: a.MimeType,
		FileID:   a.FileId,
	}
}

func SendMessageToModel(username string, time time.Time, message *chats.SendMessageRequest) *models.Message {
	return &models.Message{
		MessageID:   message.MessageId,
		FromUser:    username,
		ChatID:      message.ChatId,
		SendingTime: time,
		Text:        message.Text,
		ReplyTo:     message.ReplyTo,
		Attachments: nil,
	}
}
