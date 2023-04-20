package storage

import (
	"context"
	"database/sql"
	"errors"
	sq "github.com/Masterminds/squirrel"
	"github.com/practice-sem-2/user-service/internal/models"
	"time"
)

var (
	ErrChatAlreadyExists      = errors.New("chat with provided chat_id already exists")
	ErrChatNotFound           = errors.New("chat with provided chat_id does not exist")
	ErrEmptyMembers           = errors.New("members array can't be empty")
	ErrRepliedMessageNotFound = errors.New("message replies to a not existing message")
	ErrMessageAlreadyExists   = errors.New("message with provided message_id already exists")
	ErrMessageNotFound        = errors.New("message does not exist")
	ErrNotMember              = errors.New("user is not a member")
)

const (
	ChatsPrimaryKey             = "chats_pkey"
	ChatMembersChatIdForeignKey = "chat_members_chat_id_fkey"
	MessagesPrimaryKey          = "messages_pkey"
	MessagesReplyToForeignKey   = "messages_reply_to_fkey"
	MessagesChatIdForeignKey    = "messages_chat_id_fkey"
)

type ChatsStorage struct {
	db Scope
}

func NewChatsStorage(db Scope) *ChatsStorage {
	return &ChatsStorage{
		db: db,
	}
}

func (s *ChatsStorage) CreateChat(ctx context.Context, chatId string, isDirect bool) error {
	query, args, err := sq.Insert("chats").
		Columns("chat_id", "is_direct").
		Values(chatId, isDirect).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query, args...)

	if GetPgxConstraintName(err) == "chats_pkey" {
		return ErrChatAlreadyExists
	} else {
		return err
	}
}

func (s *ChatsStorage) AddChatMembers(ctx context.Context, chatId string, members []string) error {
	if len(members) == 0 {
		return ErrEmptyMembers
	}

	builder := sq.Insert("chat_members").
		Columns("chat_id", "user_id").
		PlaceholderFormat(sq.Dollar)

	for _, member := range members {
		builder = builder.Values(chatId, member)
	}

	query, args, err := builder.ToSql()

	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if GetPgxConstraintName(err) == "chat_members_chat_id_fkey" {
		return ErrChatAlreadyExists
	} else {
		return err
	}
}

func (s *ChatsStorage) DeleteChatMembers(ctx context.Context, chatId string, members []string) error {
	if len(members) == 0 {
		return ErrEmptyMembers
	}

	builder := sq.Delete("chat_members").
		Where(sq.Eq{"chat_id": chatId}).
		PlaceholderFormat(sq.Dollar)

	union := sq.Or{}
	for _, member := range members {
		union = append(union, sq.Eq{"user_id": member})
	}
	query, args, err := builder.Where(union).ToSql()

	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query, args...)

	if GetPgxConstraintName(err) == "chat_members_chat_id_fkey" {
		return ErrChatNotFound

	} else if GetPgxConstraintName(err) == "49cdc045-7c03-4d0e-bdc7-abcc0faa4c8b" {
		return ErrNotMember
	} else {
		return err
	}
}

func (s *ChatsStorage) GetChat(ctx context.Context, chatId string) (*models.Chat, error) {
	query, args, err := sq.Select("chats.*, count(user_id) as members_count").
		From("chats").
		Join("chat_members USING(chat_id)").
		Where(sq.Eq{"chat_id": chatId}).
		GroupBy("chats.chat_id").
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return nil, err
	}

	chat := models.Chat{}
	err = s.db.GetContext(ctx, &chat, query, args...)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrChatNotFound
	} else if err != nil {
		return nil, err
	} else {
		return &chat, nil
	}
}

func (s *ChatsStorage) GetChatWithMembers(ctx context.Context, chatId string) (*models.ChatWithMembers, error) {

	chat, err := s.GetChat(ctx, chatId)

	if err != nil {
		return nil, err
	}

	query, args, err := sq.Select("*").
		From("chats").
		Where(sq.Eq{"chat_id": chatId}).
		Join("chat_members USING(chat_id)").
		OrderBy("chat_id, user_id").
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryxContext(ctx, query, args...)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	members := make([]models.ChatMember, 0)
	for rows.Next() {
		member := models.ChatMember{}
		if err = rows.Scan(&chat.ChatID, &chat.IsDirect, &member.UserID); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return &models.ChatWithMembers{
		Chat:    *chat,
		Members: members,
	}, nil
}

func (s *ChatsStorage) UserIsMember(ctx context.Context, chatId string, userId string) (bool, error) {
	// Check if chat exists
	_, err := s.GetChat(ctx, chatId)
	if err != nil {
		return false, err
	}

	query, args, err := sq.Select("1").
		From("chats").
		Join("chat_members USING(chat_id)").
		Where(sq.Eq{
			"chat_id": chatId,
			"user_id": userId,
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	ok := false
	row := s.db.QueryRowxContext(ctx, query, args...)
	err = row.Scan(&ok)
	ok = ok && !errors.Is(err, sql.ErrNoRows)
	return ok, nil
}

func (s *ChatsStorage) PutMessage(ctx context.Context, message *models.Message) error {
	// TODO: check if message and reply_to message are in the same chat
	// TODO: add attachments handling
	query, args, err := sq.Insert("messages").
		Columns("message_id", "chat_id", "from_user", "reply_to", "text", "sending_time").
		Values(message.MessageID, message.ChatID, message.FromUser, message.ReplyTo, message.Text, message.SendingTime).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query, args...)

	if GetPgxConstraintName(err) == "messages_reply_to_fkey" {
		return ErrRepliedMessageNotFound
	} else if GetPgxConstraintName(err) == "messages_chat_id_fkey" {
		return ErrChatNotFound
	} else if GetPgxConstraintName(err) == "messages_pkey" {
		return ErrMessageAlreadyExists
	} else if err != nil {
		return err
	}

	return nil
}

type SelectOptions struct {
	Limit   uint64
	OrderBy []string
}

func (s *ChatsStorage) SelectMessages(ctx context.Context, selector sq.Sqlizer, options ...SelectOptions) ([]models.Message, error) {
	// TODO: handle attachments
	option := SelectOptions{}
	if len(options) > 0 {
		option = options[0]
	}

	builder := sq.Select("*").
		From("messages").
		Where(selector).
		PlaceholderFormat(sq.Dollar)

	if len(option.OrderBy) > 0 {
		builder = builder.OrderBy(option.OrderBy...)
	}

	if option.Limit > 0 {
		builder = builder.Limit(option.Limit)
	}

	query, args, err := builder.ToSql()

	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryxContext(ctx, query, args...)

	if GetPgxConstraintName(err) == ChatsPrimaryKey {
		return nil, ErrChatNotFound
	} else if err != nil {
		return nil, err
	}

	messages := make([]models.Message, 0)

	for rows.Next() {
		msg := models.Message{
			Attachments: []models.FileAttachment{},
		}

		err = rows.StructScan(&msg)

		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *ChatsStorage) GetMessagesSince(ctx context.Context, chatId string, since time.Time, count uint64) ([]models.Message, error) {
	selector := sq.And{
		sq.Eq{"chat_id": chatId},
		sq.GtOrEq{"sending_time": since.UTC()},
	}
	return s.SelectMessages(ctx, selector, SelectOptions{
		Limit:   count,
		OrderBy: []string{"sending_time"},
	})
}

func (s *ChatsStorage) GetMessagesBefore(ctx context.Context, chatId string, before time.Time, count uint64) ([]models.Message, error) {
	selector := sq.And{
		sq.Eq{"chat_id": chatId},
		sq.LtOrEq{"sending_time": before.UTC()},
	}
	return s.SelectMessages(ctx, selector, SelectOptions{
		Limit:   count,
		OrderBy: []string{"sending_time DESC"},
	})
}

func (s *ChatsStorage) GetMessagesById(ctx context.Context, ids []string) ([]models.Message, error) {
	selector := sq.Or{}
	for _, id := range ids {
		selector = append(selector, sq.Eq{"message_id": id})
	}
	return s.SelectMessages(ctx, selector, SelectOptions{
		OrderBy: []string{"sending_time DESC"},
	})
}

func (s *ChatsStorage) DeleteMessage(ctx context.Context, messageId string) error {
	query, args, err := sq.Delete("messages").
		Where(sq.Eq{"message_id": messageId}).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return err
	}

	res, err := s.db.ExecContext(ctx, query, args...)

	if err != nil {
		return err
	}

	count, err := res.RowsAffected()

	if err != nil {
		return err
	}

	if count == 0 {
		return ErrMessageNotFound
	}

	return nil
}

func (s *ChatsStorage) GetUserChats(ctx context.Context, userId string) ([]models.RichChat, error) {

	query, args, err := sq.
		Select("c.chat_id", "is_direct", "message_id", "from_user", "reply_to", "sending_time", "text").
		From("chats c").
		Join("messages msg ON c.chat_id = msg.chat_id").
		Join("chat_members mem ON c.chat_id = mem.chat_id ").
		Where(sq.Eq{
			"user_id": userId,
		}).
		Where("msg.sending_time = (SELECT max(sending_time) FROM messages WHERE c.chat_id = chat_id)").
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryxContext(ctx, query, args...)

	chats := make([]models.RichChat, 0)

	if errors.Is(err, sql.ErrNoRows) {
		return chats, nil
	} else if err != nil {
		return nil, err
	}

	for rows.Next() {
		chat := models.RichChat{}
		msg := models.Message{}
		chat.LastMessage = &msg
		err = rows.Scan(&chat.ChatID, &chat.IsDirect, &msg.MessageID, &msg.FromUser, &msg.ReplyTo, &msg.SendingTime, &msg.Text)
		msg.ChatID = chat.ChatID
		chats = append(chats, chat)
	}
	return chats, nil
}
