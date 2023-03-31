package storage

import (
	"context"
	"database/sql"
	"errors"
	sq "github.com/Masterminds/squirrel"
	"github.com/practice-sem-2/user-service/internal/models"
)

var (
	ErrChatAlreadyExists = errors.New("chat with provided chat_id already exists")
	ErrChatNotFound      = errors.New("chat with provided chat_id does not exist")
	ErrEmptyMembers      = errors.New("members array can't be empty")
)

type ChatsStorage struct {
	db Scope
}

func NewChatsStorage(db Scope) *ChatsStorage {
	return &ChatsStorage{
		db: db,
	}
}

func (s *ChatsStorage) CreateChat(ctx context.Context, chatId string) error {
	query, args, err := sq.Insert("chats").
		Columns("chat_id").
		Values(chatId).
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
	} else {
		return err
	}
}

func (s *ChatsStorage) GetChatWithMembers(ctx context.Context, chatId string) (*models.ChatWithMembers, error) {

	query, args, err := sq.Select("*").
		From("chats").
		Where(sq.Eq{"chat_id": chatId}).
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
	}

	query, args, err = sq.Select("*").
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
		if err = rows.Scan(&chat.ChatID, &member.UserID); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return &models.ChatWithMembers{
		Chat:    chat,
		Members: members,
	}, nil
}
