package usecases

import storage "github.com/practice-sem-2/user-service/internal/storages"

type ChatsUsecase struct {
	registry *storage.Registry
}

func NewChatsUsecase(r *storage.Registry) *ChatsUsecase {
	return &ChatsUsecase{
		registry: r,
	}
}
