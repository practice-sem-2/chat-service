package server

import (
	"context"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/practice-sem-2/auth-tools"
	"github.com/practice-sem-2/user-service/internal/models"
	"github.com/practice-sem-2/user-service/internal/pb/chats"
	storage "github.com/practice-sem-2/user-service/internal/storages"
	usecase "github.com/practice-sem-2/user-service/internal/usecases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

var (
	NoReturn = &emptypb.Empty{}
)

type ChatServer struct {
	chats.UnimplementedChatServer
	chats    *usecase.ChatsUsecase
	auth     *auth.VerifierService
	validate *validator.Validate
}

func NewChatServer(c *usecase.ChatsUsecase, a *auth.VerifierService, v *validator.Validate) *ChatServer {
	return &ChatServer{
		chats:    c,
		auth:     a,
		validate: v,
	}
}

func (s *ChatServer) GetUserChats(ctx context.Context, r *chats.GetChatsRequest) (*chats.GetChatsResponse, error) {
	claims, err := s.auth.GetUser(ctx)

	if err != nil {
		return nil, wrapError(err)
	}

	userChats, err := s.chats.GetUsersChats(ctx, claims)

	if err != nil {
		return nil, wrapError(err)
	}

	res := &chats.GetChatsResponse{
		Chats: make([]*chats.RichChat, len(userChats)),
	}
	// TODO: handle attachments
	for i, chat := range userChats {
		res.Chats[i] = &chats.RichChat{
			ChatId:   chat.ChatID,
			IsDirect: chat.IsDirect,
			LastMessage: &chats.Message{
				MessageId:   chat.LastMessage.MessageID,
				FromUser:    chat.LastMessage.FromUser,
				ChatId:      chat.LastMessage.ChatID,
				Timestamp:   chat.LastMessage.SendingTime.UTC().Unix(),
				Text:        chat.LastMessage.Text,
				ReplyTo:     chat.LastMessage.ReplyTo,
				Attachments: nil,
			},
		}
	}
	return res, nil
}

func (s *ChatServer) GetMessages(ctx context.Context, r *chats.GetMessagesRequest) (*chats.GetMessagesResponse, error) {
	claims, err := s.auth.GetUser(ctx)

	if err != nil {
		return nil, wrapError(err)
	}

	sel := &models.MessagesSelect{ChatID: r.ChatId}

	if r.Since != nil {
		sel.Since = new(time.Time)
		*sel.Since = time.Unix(*r.Since, 0).UTC()
	}

	if r.Until != nil {
		sel.Until = new(time.Time)
		*sel.Until = time.Unix(*r.Until, 0).UTC()
	}

	if r.Count != nil {
		count := new(int)
		*count = int(*r.Count)
		sel.Count = count
	}

	messages, err := s.chats.GetMessages(ctx, claims, sel)

	if err != nil {
		return nil, wrapError(err)
	}

	res := &chats.GetMessagesResponse{
		Messages: make([]*chats.Message, len(messages)),
	}

	for i, msg := range messages {
		res.Messages[i] = &chats.Message{
			MessageId:   msg.MessageID,
			FromUser:    msg.FromUser,
			ChatId:      msg.ChatID,
			Timestamp:   msg.SendingTime.Unix(),
			Text:        msg.Text,
			ReplyTo:     msg.ReplyTo,
			Attachments: nil,
		}
	}
	return res, nil
}

func (s *ChatServer) CreateChat(ctx context.Context, r *chats.CreateChatRequest) (*emptypb.Empty, error) {

	claims, err := s.auth.GetUser(ctx)

	if err != nil {
		return nil, wrapError(err)
	}

	err = s.validate.Var(r.ChatId, "uuid")

	if err != nil {
		return nil, wrapError(err)
	}

	err = s.chats.CreateChat(ctx, claims, models.ChatCreate{
		ChatID:   r.ChatId,
		IsDirect: r.IsDirect,
		Members:  r.Members,
	})

	if err != nil {
		return nil, wrapError(err)
	}

	return NoReturn, nil
}

func (s *ChatServer) GetChat(ctx context.Context, r *chats.GetChatRequest) (*chats.GetChatResponse, error) {
	claims, err := s.auth.GetUser(ctx)

	if err != nil {
		return nil, wrapError(err)
	}

	chat, err := s.chats.GetChatWithMembers(ctx, claims, r.ChatId)

	if err != nil {
		return nil, wrapError(err)
	}

	res := &chats.GetChatResponse{
		ChatId:       chat.ChatID,
		MembersCount: int32(chat.MembersCount),
		Members:      make([]string, len(chat.Members)),
	}

	for i, member := range chat.Members {
		res.Members[i] = member.UserID
	}

	return res, nil
}

func (s *ChatServer) DeleteChat(ctx context.Context, r *chats.DeleteChatRequest) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (s *ChatServer) AddChatMembers(ctx context.Context, r *chats.AddChatMembersRequest) (*emptypb.Empty, error) {
	claims, err := s.auth.GetUser(ctx)

	if err != nil {
		return nil, wrapError(err)
	}

	err = s.chats.AddChatMembers(ctx, claims, r.ChatId, r.Members)
	return NoReturn, wrapError(err)
}

func (s *ChatServer) DeleteChatMembers(ctx context.Context, r *chats.DeleteChatMembersRequest) (*emptypb.Empty, error) {
	claims, err := s.auth.GetUser(ctx)

	if err != nil {
		return nil, wrapError(err)
	}

	err = s.chats.DeleteChatMembers(ctx, claims, r.ChatId, r.Members)
	return NoReturn, wrapError(err)
}

func (s *ChatServer) SendMessage(ctx context.Context, r *chats.SendMessageRequest) (*emptypb.Empty, error) {
	// TODO: Handle attachments

	user, err := s.auth.GetUser(ctx)

	if err != nil {
		return nil, wrapError(err)
	}

	msg := models.MessageSend{
		MessageID:   r.MessageId,
		ChatID:      r.ChatId,
		Text:        r.Text,
		ReplyTo:     r.ReplyTo,
		Attachments: nil,
	}
	err = s.validate.Struct(msg)

	if err != nil {
		return nil, wrapError(err)
	}

	err = s.chats.SendMessage(ctx, user, msg)

	if err != nil {
		return nil, wrapError(err)
	}
	return NoReturn, err
}

func wrapError(err error) error {

	if err == nil {
		return nil
	}

	errorMapper := []struct {
		from error
		to   error
	}{
		{
			from: storage.ErrChatAlreadyExists,
			to:   status.Error(codes.AlreadyExists, err.Error()),
		},
		{
			from: storage.ErrChatNotFound,
			to:   status.Error(codes.NotFound, err.Error()),
		},
	}

	if validationErr, ok := err.(validator.ValidationErrors); ok {
		return status.Error(codes.InvalidArgument, validationErr.Error())
	}

	for _, mapping := range errorMapper {
		if errors.Is(err, mapping.from) {
			return mapping.to
		}
	}
	return status.Error(codes.Internal, err.Error())
}
