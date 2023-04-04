package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/practice-sem-2/user-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type ChatsStorageTestSuite struct {
	PostgresTestSuite
}

func (s *ChatsStorageTestSuite) TearDownTest() {
	_, err := s.db.Exec("TRUNCATE messages, chat_members, chats, attachments")
	require.NoError(s.T(), err, "can't teardown test")
}

func TestChatsStorageTestSuite(t *testing.T) {
	suite.Run(t, &ChatsStorageTestSuite{})
}

func (s *ChatsStorageTestSuite) Test_CreateChat() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store := NewChatsStorage(s.db)
	err := store.CreateChat(ctx, chatId)
	assert.NoError(s.T(), err, "should correctly create chat")

	// Check if chat was actually created
	row := s.db.QueryRow("SELECT count(*) FROM chats WHERE chat_id=$1::uuid", chatId)
	count := 0
	err = row.Scan(&count)
	assert.NoError(s.T(), err, "should be scanned correctly")
	assert.Equal(s.T(), 1, count, "should be exactly 1 row")
}

func (s *ChatsStorageTestSuite) Test_Create_CorrectErrorIfChatExists() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store := NewChatsStorage(s.db)
	err := store.CreateChat(ctx, chatId)
	assert.NoError(s.T(), err, "should correctly create chat")

	assert.ErrorIs(s.T(), store.CreateChat(ctx, chatId), ErrChatAlreadyExists)

}

func (s *ChatsStorageTestSuite) Test_AddMember() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store := NewChatsStorage(s.db)
	err := store.CreateChat(ctx, chatId)
	assert.NoError(s.T(), err, "should correctly create chat")

	members := []string{
		"74cccd17-9c56-490b-b721-88c027976863",
		"67f85047-09d0-42a2-a5ee-9ce8db28cb07",
	}

	err = store.AddChatMembers(ctx, chatId, members)
	assert.NoError(s.T(), err, "should correctly add members chat")

	row := s.db.QueryRow(`
		SELECT count(*) 
		FROM chat_members 
		WHERE user_id IN(
		    '74cccd17-9c56-490b-b721-88c027976863',
		    '67f85047-09d0-42a2-a5ee-9ce8db28cb07'
		)`,
	)
	count := 0
	err = row.Scan(&count)
	assert.NoError(s.T(), err, "rows count should be correctly scanned")
	assert.Equal(s.T(), 2, count, "there should be exactly 2 members in a chat")
}

func (s *ChatsStorageTestSuite) Test_AddMember_Atomic() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	registry := NewRegistry(s.db)

	err := registry.Atomic(ctx, func(registry *Registry) error {
		store := registry.GetChatsStore()
		err := store.CreateChat(ctx, chatId)
		assert.NoError(s.T(), err, "should correctly create chat")

		err = store.AddChatMembers(ctx, chatId, []string{"74cccd17-9c56-490b-b721-88c027976863"})
		return errors.New("bang")
	})

	assert.Error(s.T(), err, "should return error")

	row := s.db.QueryRow(`
		SELECT count(*) FROM chats WHERE chat_id=$1
	`, chatId)
	count := 0
	err = row.Scan(&count)
	assert.NoError(s.T(), err, "rows count should be correctly scanned")
	assert.Equal(s.T(), 0, count, "whole transaction should be rolled back")
}

func (s *ChatsStorageTestSuite) Test_DeleteMember() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store := NewChatsStorage(s.db)
	err := store.CreateChat(ctx, chatId)
	assert.NoError(s.T(), err, "should correctly create chat")

	members := []string{
		"74cccd17-9c56-490b-b721-88c027976863",
		"67f85047-09d0-42a2-a5ee-9ce8db28cb07",
	}

	err = store.AddChatMembers(ctx, chatId, members)
	assert.NoError(s.T(), err, "should correctly add members chat")

	err = store.DeleteChatMembers(ctx, chatId, []string{"74cccd17-9c56-490b-b721-88c027976863"})
	assert.NoError(s.T(), err, "should correctly delete member from chat")

	row := s.db.QueryRow(`
		SELECT count(*) 
		FROM chat_members 
		WHERE user_id = '74cccd17-9c56-490b-b721-88c027976863'`,
	)
	count := 0
	err = row.Scan(&count)
	assert.NoError(s.T(), err, "rows count should be correctly scanned")
	assert.Equal(s.T(), 0, count, "member should be correctly deleted from chat")
}

func (s *ChatsStorageTestSuite) Test_GetChatWithMembers() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"
	members := []string{
		"74cccd17-9c56-490b-b721-88c027976863",
		"67f85047-09d0-42a2-a5ee-9ce8db28cb07",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	store := NewChatsStorage(s.db)
	err := store.CreateChat(ctx, chatId)
	assert.NoError(s.T(), err, "should correctly create chat")

	err = store.AddChatMembers(ctx, chatId, members)
	assert.NoError(s.T(), err, "should correctly add members chat")

	chat, err := store.GetChatWithMembers(ctx, chatId)
	assert.NoError(s.T(), err, "should correctly return chat with members")
	assert.Equal(s.T(), chatId, chat.ChatID)

	expectedMembers := []models.ChatMember{
		{UserID: "67f85047-09d0-42a2-a5ee-9ce8db28cb07"},
		{UserID: "74cccd17-9c56-490b-b721-88c027976863"},
	}
	assert.Equal(s.T(), expectedMembers, chat.Members, "should contain all chat members")
}

func (s *ChatsStorageTestSuite) Test_GetChatWithMembers_CorrectErrorIfChatDoesNotExist() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	store := NewChatsStorage(s.db)
	_, err := store.GetChatWithMembers(ctx, chatId)
	assert.ErrorIs(s.T(), err, ErrChatNotFound)
}

func (s *ChatsStorageTestSuite) Test_UserIsMember() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"
	const userId = "74cccd17-9c56-490b-b721-88c027976863"
	const userIdNotMember = "67f85047-09d0-42a2-a5ee-9ce8db28cb07"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	store := NewChatsStorage(s.db)
	err := store.CreateChat(ctx, chatId)
	assert.NoError(s.T(), err, "should correctly create chat")
	err = store.AddChatMembers(ctx, chatId, []string{userId})
	assert.NoError(s.T(), err, "should correctly add members")

	isMember, err := store.UserIsMember(ctx, chatId, userId)
	assert.NoError(s.T(), err, "should return no error")
	assert.True(s.T(), isMember, "user is member")

	isMember, err = store.UserIsMember(ctx, chatId, userIdNotMember)
	assert.NoError(s.T(), err, "should return no error")
	assert.False(s.T(), isMember, "user is not member")
}

func (s *ChatsStorageTestSuite) Test_UserIsMember_IfChatNotExist() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"
	const userId = "74cccd17-9c56-490b-b721-88c027976863"
	const userIdNotMember = "67f85047-09d0-42a2-a5ee-9ce8db28cb07"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	store := NewChatsStorage(s.db)

	_, err := store.UserIsMember(ctx, chatId, userId)
	assert.ErrorIs(s.T(), ErrChatNotFound, err, "should return no error")
}
func (s *ChatsStorageTestSuite) Test_PutMessage() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"
	const userId = "74cccd17-9c56-490b-b721-88c027976863"
	const messageId = "67f85047-09d0-42a2-a5ee-9ce8db28cb07"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	store := NewChatsStorage(s.db)
	err := store.CreateChat(ctx, chatId)
	assert.NoError(s.T(), err, "should correctly create chat")
	err = store.AddChatMembers(ctx, chatId, []string{userId})
	assert.NoError(s.T(), err, "should correctly add member to chat")

	expectedMsg := models.Message{
		MessageID:   messageId,
		FromUser:    userId,
		ChatID:      chatId,
		SendingTime: time.Now().UTC(),
		Text:        "Hello, world!",
		ReplyTo:     nil,
		Attachments: nil,
	}
	err = store.PutMessage(ctx, &expectedMsg)
	assert.NoError(s.T(), err, "should correctly send message to chat")

	row := s.db.QueryRow("SELECT * FROM messages WHERE message_id = $1", messageId)
	msg := models.Message{}
	err = row.Scan(&msg.MessageID, &msg.ChatID, &msg.FromUser, &msg.ReplyTo, &msg.SendingTime, &msg.Text)
	assert.NoError(s.T(), err, "should return row from db")

	assert.Equal(s.T(), expectedMsg, msg)
}

func (s *ChatsStorageTestSuite) Test_DeleteMessage() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"
	const userId = "74cccd17-9c56-490b-b721-88c027976863"
	const messageId = "67f85047-09d0-42a2-a5ee-9ce8db28cb07"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	store := NewChatsStorage(s.db)
	err := store.CreateChat(ctx, chatId)
	assert.NoError(s.T(), err, "should correctly create chat")
	err = store.AddChatMembers(ctx, chatId, []string{userId})
	assert.NoError(s.T(), err, "should correctly add member to chat")

	expectedMsg := models.Message{
		MessageID:   messageId,
		FromUser:    userId,
		ChatID:      chatId,
		SendingTime: time.Now().UTC(),
		Text:        "Hello, world!",
		ReplyTo:     nil,
		Attachments: nil,
	}
	err = store.PutMessage(ctx, &expectedMsg)
	assert.NoError(s.T(), err, "should correctly send message to chat")

	err = store.DeleteMessage(ctx, messageId)

	assert.NoError(s.T(), err, "should not return any error")
	count := -1
	err = s.db.GetContext(ctx, &count, "SELECT count(1) FROM messages WHERE message_id = $1", messageId)
	assert.NoError(s.T(), err, "should not return any error")
	assert.Equal(s.T(), count, 0, "row should be deleted")
}

func (s *ChatsStorageTestSuite) Test_DeleteMessage_IfMessageDoesNotExists() {
	const messageId = "67f85047-09d0-42a2-a5ee-9ce8db28cb07"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	store := NewChatsStorage(s.db)

	err := store.DeleteMessage(ctx, messageId)

	assert.ErrorIs(s.T(), err, ErrMessageNotFound, "should not return any error")
}

func (s *ChatsStorageTestSuite) Test_SelectMessage() {
	const chatId = "694a909e-bec7-4dbe-bf38-935a99d848cc"
	const userId = "74cccd17-9c56-490b-b721-88c027976863"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	store := NewChatsStorage(s.db)
	err := store.CreateChat(ctx, chatId)
	assert.NoError(s.T(), err, "should correctly create chat")
	err = store.AddChatMembers(ctx, chatId, []string{userId})
	assert.NoError(s.T(), err, "should correctly add member to chat")

	beginTime := time.Now().UTC().Add(-15 * time.Hour)
	timeSent := beginTime

	insertedMessages := make([]models.Message, 0, 10)
	for i := 0; i < 10; i++ {
		id, _ := uuid.NewRandom()
		msg := models.Message{
			MessageID:   id.String(),
			FromUser:    userId,
			ChatID:      chatId,
			SendingTime: timeSent,
			Text:        fmt.Sprintf("Hello, world! (%d)", i),
			ReplyTo:     nil,
			Attachments: []models.FileAttachment{},
		}
		if i != 0 {
			msg.ReplyTo = &insertedMessages[i-1].MessageID
		}
		insertedMessages = append(insertedMessages, msg)
		timeSent = timeSent.Add(time.Hour)
		err = store.PutMessage(ctx, &msg)
		assert.NoError(s.T(), err, "should correctly send message to chat")
	}

	actualMessages, err := store.GetMessagesSince(ctx, chatId, beginTime, 3)
	assert.NoError(s.T(), err, "should not return any error")
	assert.Equal(s.T(), insertedMessages[:3], actualMessages, "query should return first three messages")

	actualMessages, err = store.GetMessagesSince(ctx, chatId, beginTime.Add(6*time.Hour), 3)
	assert.NoError(s.T(), err, "should not return any error")
	assert.Equal(s.T(), insertedMessages[6:9], actualMessages, "query should return first three messages")

	expectedMessages := make([]models.Message, 5)
	copy(expectedMessages, insertedMessages[5:])
	for i := 0; i < len(expectedMessages)/2; i++ {
		tmp := expectedMessages[i]
		expectedMessages[i] = expectedMessages[len(expectedMessages)-i-1]
		expectedMessages[len(expectedMessages)-i-1] = tmp
	}

	actualMessages, err = store.GetMessagesBefore(ctx, chatId, timeSent, 5)
	assert.NoError(s.T(), err, "should not return any error")
	assert.Equal(s.T(), expectedMessages, actualMessages, "query should return first three messages")
}
