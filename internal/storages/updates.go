package storage

import (
	"github.com/Shopify/sarama"
	"github.com/practice-sem-2/user-service/internal/models"
	"github.com/practice-sem-2/user-service/internal/pb/chats/updates"
	"google.golang.org/protobuf/proto"
	"time"
)

type UpdatesStorage struct {
	cfg      *UpdatesStoreConfig
	producer sarama.SyncProducer
}

type UpdatesStoreConfig struct {
	UpdatesTopic string
}

func NewUpdatesStore(p sarama.SyncProducer, cfg *UpdatesStoreConfig) *UpdatesStorage {
	return &UpdatesStorage{
		producer: p,
		cfg:      cfg,
	}
}

func (s *UpdatesStorage) putUpdate(topic, key string, event *updates.Update) error {
	bytes, err := proto.Marshal(event)
	if err != nil {
		return err
	}

	_, _, err = s.producer.SendMessage(&sarama.ProducerMessage{
		Topic:     topic,
		Key:       sarama.StringEncoder(key),
		Value:     sarama.ByteEncoder(bytes),
		Timestamp: time.Time{},
	})

	return err
}

func (s *UpdatesStorage) chatCreatedToProtobuf(chat *models.ChatCreated) *updates.Update {
	return &updates.Update{
		Meta: &updates.UpdateMeta{
			Timestamp: chat.Timestamp.UTC().Unix(),
			Audience:  chat.Audience,
		},
		Update: &updates.Update_CreatedChat{
			CreatedChat: &updates.ChatCreated{
				ChatId:   chat.ChatID,
				IsDirect: chat.IsDirect,
				Members:  chat.Members,
			},
		},
	}
}

func (s *UpdatesStorage) messageSentToProtobuf(msg *models.MessageSent) *updates.Update {
	l := 0
	if msg.Attachments != nil {
		l = len(msg.Attachments)
	}
	attachments := make([]*updates.FileAttachment, l)
	for i, att := range msg.Attachments {
		attachments[i] = &updates.FileAttachment{
			MimeType: att.MimeType,
			FileId:   att.FileID,
		}
	}
	return &updates.Update{
		Meta: &updates.UpdateMeta{
			Timestamp: msg.Timestamp.UTC().Unix(),
			Audience:  msg.Audience,
		},
		Update: &updates.Update_Message{
			Message: &updates.MessageSent{
				MessageId:   msg.MessageID,
				FromUser:    msg.FromUser,
				ChatId:      msg.ChatID,
				Text:        msg.Text,
				ReplyTo:     msg.ReplyTo,
				Attachments: attachments,
			},
		},
	}
}

func (s *UpdatesStorage) memberAddedToProtobuf(member *models.MemberAdded) *updates.Update {
	return &updates.Update{
		Meta: &updates.UpdateMeta{
			Timestamp: member.Timestamp.UTC().Unix(),
			Audience:  member.Audience,
		},
		Update: &updates.Update_MemberAdded{
			MemberAdded: &updates.MemberAdded{
				ChatId:   member.ChatID,
				Username: member.Username,
			},
		},
	}
}

func (s *UpdatesStorage) memberRemovedToProtobuf(member *models.MemberRemoved) *updates.Update {
	return &updates.Update{
		Meta: &updates.UpdateMeta{
			Timestamp: member.Timestamp.UTC().Unix(),
			Audience:  member.Audience,
		},
		Update: &updates.Update_MemberRemoved{
			MemberRemoved: &updates.MemberRemoved{
				ChatId:   member.ChatID,
				Username: member.Username,
			},
		},
	}
}

func (s *UpdatesStorage) ChatCreated(chat *models.ChatCreated) error {
	update := s.chatCreatedToProtobuf(chat)
	return s.putUpdate(s.cfg.UpdatesTopic, chat.ChatID, update)
}

func (s *UpdatesStorage) MessageSent(msg *models.MessageSent) error {
	update := s.messageSentToProtobuf(msg)
	return s.putUpdate(s.cfg.UpdatesTopic, msg.ChatID, update)
}

func (s *UpdatesStorage) MemberAdded(member *models.MemberAdded) error {
	update := s.memberAddedToProtobuf(member)
	return s.putUpdate(s.cfg.UpdatesTopic, member.ChatID, update)
}

func (s *UpdatesStorage) MemberRemoved(member *models.MemberRemoved) error {
	update := s.memberRemovedToProtobuf(member)
	return s.putUpdate(s.cfg.UpdatesTopic, member.ChatID, update)
}
