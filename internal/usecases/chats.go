package usecases

import (
	"context"
	"errors"
	"fmt"
	"github.com/practice-sem-2/auth-tools"
	"github.com/practice-sem-2/user-service/internal/models"
	storage "github.com/practice-sem-2/user-service/internal/storages"
	"time"
)

var (
	ErrPermissionDenied       = errors.New("user is not authorized to this action")
	ErrAuthenticationRequired = fmt.Errorf("%w: Authentication required", ErrPermissionDenied)
	ErrUserIsNotAChatMember   = fmt.Errorf("%w: User is not a chat member", ErrPermissionDenied)
	ErrBusinessLogicViolation = errors.New("business logic violation")
)

type ChatsUsecase struct {
	registry *storage.Registry
}

func NewChatsUsecase(r *storage.Registry) *ChatsUsecase {
	return &ChatsUsecase{
		registry: r,
	}
}

func (u *ChatsUsecase) CreateChat(ctx context.Context, claims *auth.UserClaims, chat models.ChatCreate) (err error) {
	if claims == nil {
		return ErrAuthenticationRequired
	}

	found := false
	for _, mem := range chat.Members {
		if mem == claims.Username {
			found = true
			break
		}
	}

	if !found {
		chat.Members = append(chat.Members, claims.Username)
	}

	if chat.IsDirect && len(chat.Members) != 2 {
		return fmt.Errorf("%w: direct chat must have exactly two members", ErrBusinessLogicViolation)
	}

	err = u.registry.Atomic(ctx, func(r *storage.Registry) error {
		store := r.GetChatsStore()
		err := store.CreateChat(ctx, chat.ChatID, chat.IsDirect)
		if err != nil {
			return err
		}
		err = store.AddChatMembers(ctx, chat.ChatID, chat.Members)
		return err
	})
	return
}

func (u *ChatsUsecase) GetChatWithMembers(ctx context.Context, claims *auth.UserClaims, chatId string) (c *models.ChatWithMembers, err error) {
	err = u.registry.Atomic(ctx, func(r *storage.Registry) error {
		store := r.GetChatsStore()

		isMember, err := store.UserIsMember(ctx, chatId, claims.Username)

		if err != nil {
			return err
		}

		if !isMember {
			err = ErrUserIsNotAChatMember
			return err
		}

		c, err = store.GetChatWithMembers(ctx, chatId)

		return err
	})
	return
}

func (u *ChatsUsecase) SendMessage(ctx context.Context, sender *auth.UserClaims, message models.MessageSend) error {
	// TODO: Handle attachments

	return u.registry.Atomic(ctx, func(r *storage.Registry) error {
		store := r.GetChatsStore()

		// Check if user is a chat member
		isMember, err := store.UserIsMember(ctx, message.ChatID, sender.Username)
		if err != nil {
			return err
		} else if !isMember {
			return ErrUserIsNotAChatMember
		}

		// If ReplyTo is not nil, check weather replied message exists and is in the same chat
		if message.ReplyTo != nil {
			msgs, err := store.GetMessagesById(ctx, []string{*message.ReplyTo})

			if err != nil {
				return err
			}

			repliedMsg := msgs[0]
			if repliedMsg.ChatID != message.ChatID {
				return fmt.Errorf("%w: replied message must be in the same chat", ErrBusinessLogicViolation)
			}
		}

		err = store.PutMessage(ctx, &models.Message{
			MessageID:   message.MessageID,
			FromUser:    sender.Username,
			ChatID:      message.ChatID,
			SendingTime: time.Now().UTC(),
			Text:        message.Text,
			ReplyTo:     message.ReplyTo,
			Attachments: nil,
		})
		return err
	})
}
