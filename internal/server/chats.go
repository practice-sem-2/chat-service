package server

import (
	"context"
	"errors"
	"github.com/practice-sem-2/user-service/internal/pb"
	usecase "github.com/practice-sem-2/user-service/internal/usecases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ChatServer struct {
	pb.UnimplementedChatServer
	chats *usecase.ChatsUsecase
}

func NewChatServer(c *usecase.ChatsUsecase) *ChatServer {
	return &ChatServer{
		chats: c,
	}
}

func (c *ChatServer) CreateChat(ctx context.Context, r *pb.CreateChatRequest) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (c *ChatServer) DeleteChat(ctx context.Context, r *pb.DeleteChatRequest) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (c *ChatServer) AddChatMembers(ctx context.Context, r *pb.AddChatMembersRequest) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (c *ChatServer) DeleteChatMembers(ctx context.Context, r *pb.DeleteChatMembersRequest) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (c *ChatServer) SendMessage(ctx context.Context, r *pb.SendMessageRequest) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (c *ChatServer) GetMessages(ctx context.Context, r *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
	//TODO implement me
	panic("implement me")
}

func wrapError(err error) error {
	errorMapper := []struct {
		from error
		to   error
	}{}

	if err == nil {
		return nil
	}

	for _, mapping := range errorMapper {
		if errors.Is(err, mapping.from) {
			return mapping.to
		}
	}
	return status.Error(codes.Internal, err.Error())
}
