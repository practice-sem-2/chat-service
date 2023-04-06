package server

import (
	"context"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/practice-sem-2/auth-tools"
	"github.com/practice-sem-2/user-service/internal/models"
	"github.com/practice-sem-2/user-service/internal/pb"
	storage "github.com/practice-sem-2/user-service/internal/storages"
	usecase "github.com/practice-sem-2/user-service/internal/usecases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	NoReturn = &emptypb.Empty{}
)

type ChatServer struct {
	pb.UnimplementedChatServer
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

func (s *ChatServer) CreateChat(ctx context.Context, r *pb.CreateChatRequest) (*emptypb.Empty, error) {

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

func (s *ChatServer) DeleteChat(ctx context.Context, r *pb.DeleteChatRequest) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (s *ChatServer) AddChatMembers(ctx context.Context, r *pb.AddChatMembersRequest) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (s *ChatServer) DeleteChatMembers(ctx context.Context, r *pb.DeleteChatMembersRequest) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (s *ChatServer) SendMessage(ctx context.Context, r *pb.SendMessageRequest) (*emptypb.Empty, error) {
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

func (s *ChatServer) GetMessagesSince(ctx context.Context, r *pb.GetMessagesSinceRequest) (*pb.GetMessagesResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s *ChatServer) GetMessagesBefore(ctx context.Context, r *pb.GetMessagesBeforeRequest) (*pb.GetMessagesResponse, error) {
	//TODO implement me
	panic("implement me")
}

func wrapError(err error) error {
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

	if err == nil {
		return nil
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
