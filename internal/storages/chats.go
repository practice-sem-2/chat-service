package storage

type ChatsStorage struct {
	db Scope
}

func NewChatsStorage(db Scope) ChatsStorage {
	return ChatsStorage{
		db: db,
	}
}
