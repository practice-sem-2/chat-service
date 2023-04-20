package usecases

import (
	"context"
	"errors"
	"fmt"
	"github.com/Masterminds/squirrel"
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
	registry storage.Registry
}

func NewChatsUsecase(r storage.Registry) *ChatsUsecase {
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

	err = u.registry.Atomic(ctx, func(r storage.Registry) error {
		store := r.GetChatsStore()
		err := store.CreateChat(ctx, chat.ChatID, chat.IsDirect)
		if err != nil {
			return err
		}
		err = store.AddChatMembers(ctx, chat.ChatID, chat.Members)
		if err != nil {
			return err
		}

		upd := u.registry.GetUpdatesStore()
		err = upd.ChatCreated(&models.ChatCreated{
			UpdateMeta: models.UpdateMeta{
				Audience: chat.Members,
			},
			ChatID:   chat.ChatID,
			IsDirect: chat.IsDirect,
			Members:  chat.Members,
		})
		return err
	})

	return
}

func (u *ChatsUsecase) GetChatWithMembers(ctx context.Context, claims *auth.UserClaims, chatId string) (c *models.ChatWithMembers, err error) {
	err = u.registry.Atomic(ctx, func(r storage.Registry) error {
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

func (u *ChatsUsecase) AddChatMembers(ctx context.Context, claims *auth.UserClaims, chatId string, users []string) error {
	err := u.registry.Atomic(ctx, func(r storage.Registry) error {
		store := r.GetChatsStore()
		isMember, err := store.UserIsMember(ctx, chatId, claims.Username)
		if err != nil {
			return err
		}

		if !isMember {
			return ErrUserIsNotAChatMember
		}

		audience, err := u.getChatAudience(ctx, chatId, store)
		if err != nil {
			return err
		}

		for _, username := range audience {
			err = r.GetUpdatesStore().MemberAdded(&models.MemberAdded{
				UpdateMeta: models.UpdateMeta{
					Timestamp: time.Time{},
					Audience:  audience,
				},
				ChatID:   chatId,
				Username: username,
			})

			if err != nil {
				return err
			}
		}

		return store.AddChatMembers(ctx, chatId, users)
	})
	return err
}

func (u *ChatsUsecase) DeleteChatMembers(ctx context.Context, claims *auth.UserClaims, chatId string, users []string) error {
	err := u.registry.Atomic(ctx, func(r storage.Registry) error {
		store := r.GetChatsStore()
		isMember, err := store.UserIsMember(ctx, chatId, claims.Username)
		if err != nil {
			return err
		}

		if !isMember {
			return ErrUserIsNotAChatMember
		}

		audience, err := u.getChatAudience(ctx, chatId, store)
		if err != nil {
			return err
		}

		for _, username := range audience {
			err = r.GetUpdatesStore().MemberAdded(&models.MemberAdded{
				UpdateMeta: models.UpdateMeta{
					Timestamp: time.Time{},
					Audience:  audience,
				},
				ChatID:   chatId,
				Username: username,
			})

			if err != nil {
				return err
			}
		}

		return store.DeleteChatMembers(ctx, chatId, users)
	})
	return err
}

func (u *ChatsUsecase) SendMessage(ctx context.Context, sender *auth.UserClaims, message models.MessageSend) error {
	// TODO: Handle attachments

	return u.registry.Atomic(ctx, func(r storage.Registry) error {
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

		now := time.Now().UTC()
		err = store.PutMessage(ctx, &models.Message{
			MessageID:   message.MessageID,
			FromUser:    sender.Username,
			ChatID:      message.ChatID,
			SendingTime: now,
			Text:        message.Text,
			ReplyTo:     message.ReplyTo,
			Attachments: nil,
		})

		if err != nil {
			return err
		}

		upd := r.GetUpdatesStore()

		audience, err := u.getChatAudience(ctx, message.ChatID, store)
		if err != nil {
			return err
		}

		err = upd.MessageSent(&models.MessageSent{
			UpdateMeta: models.UpdateMeta{
				Timestamp: now,
				Audience:  audience,
			},
			MessageID:   message.MessageID,
			FromUser:    sender.Username,
			ChatID:      message.ChatID,
			Text:        message.Text,
			ReplyTo:     message.ReplyTo,
			Attachments: message.Attachments,
		})
		return err
	})
}

func (u *ChatsUsecase) getChatAudience(ctx context.Context, chatId string, store *storage.ChatsStorage) ([]string, error) {
	chat, err := store.GetChatWithMembers(ctx, chatId)
	if err != nil {
		return nil, fmt.Errorf("can't get chat members: %v", err)
	}
	audience := make([]string, len(chat.Members))
	for i, mem := range chat.Members {
		audience[i] = mem.UserID
	}
	return audience, nil
}

func (u *ChatsUsecase) GetMessages(ctx context.Context, user *auth.UserClaims, sel *models.MessagesSelect) ([]models.Message, error) {
	query := squirrel.And{squirrel.Eq{"chat_id": sel.ChatID}}
	if sel.Since != nil {
		query = append(query, squirrel.GtOrEq{"sending_time": *sel.Since})
	}
	if sel.Until != nil {
		query = append(query, squirrel.LtOrEq{"sending_time": *sel.Until})
	}
	opt := storage.SelectOptions{
		Limit:   500,
		OrderBy: []string{"sending_time ASC"},
	}
	if sel.Count != nil {
		opt.Limit = uint64(*sel.Count)
	}

	var messages []models.Message
	err := u.registry.Atomic(ctx, func(r storage.Registry) error {
		var err error

		store := r.GetChatsStore()

		isMember, err := store.UserIsMember(ctx, sel.ChatID, user.Username)
		if err != nil {
			return err
		}

		if !isMember {
			return ErrUserIsNotAChatMember
		}

		messages, err = store.SelectMessages(ctx, query, opt)
		return err
	})

	return messages, err
}

func (u *ChatsUsecase) GetUsersChats(ctx context.Context, user *auth.UserClaims) ([]models.RichChat, error) {
	return u.registry.GetChatsStore().GetUserChats(ctx, user.Username)
}
