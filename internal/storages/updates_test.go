package storage

import (
	"context"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/practice-sem-2/user-service/internal/models"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"strings"
	"testing"
	"time"
)

type EventsTestSuite struct {
	suite.Suite
	p sarama.SyncProducer
	c sarama.Consumer
}

func (s *EventsTestSuite) TearDownSuite() {
	err := s.p.Close()
	require.NoError(s.T(), err, "Sarama producer should be closed correctly")
}

func (s *EventsTestSuite) SetupSuite() {
	viper.AutomaticEnv()
	brokers := viper.GetString("KAFKA_BROKERS")

	if len(brokers) == 0 {
		require.FailNow(s.T(), "KAFKA_BROKERS must be defined")
	}

	addrs := strings.Split(brokers, ",")
	config := sarama.NewConfig()
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Timeout = 10 * time.Second
	config.Producer.Return.Successes = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Offsets.AutoCommit.Enable = false

	var err error
	s.p, err = sarama.NewSyncProducer(addrs, config)
	require.NoError(s.T(), err, fmt.Sprintf("can't create kafka producer: %v", err))

	s.c, err = sarama.NewConsumer(addrs, config)
	require.NoError(s.T(), err, fmt.Sprintf("can't create kafka producer: %v", err))
}

func TestEventsSuite(t *testing.T) {
	suite.Run(t, &EventsTestSuite{})
}

func (s *EventsTestSuite) Test_EventsStorage_MemberAdded() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	consumer, err := s.c.ConsumePartition("test", 0, sarama.OffsetNewest)
	require.NoError(s.T(), err, "create consume partition")
	defer consumer.Close()

	update := models.MemberAdded{
		UpdateMeta: models.UpdateMeta{
			Timestamp: time.Now().UTC(),
			Audience: []string{
				"253becbb-76b1-4471-9ff3-529462925899",
				"1230cadb-899e-4710-8cdd-0a2f83882712",
			},
		},
		ChatID:   "256e3354-8263-4913-8bdd-345bd04d962e",
		Username: "johndoe",
	}
	store := NewUpdatesStore(s.p, &UpdatesStoreConfig{UpdatesTopic: "test"})
	err = store.MemberAdded(&update)
	assert.NoError(s.T(), err, "event should be pushed without error")

	select {
	case msg := <-consumer.Messages():
		body, err := proto.Marshal(store.memberAddedToProtobuf(&update))
		require.NoError(s.T(), err)

		assert.Equal(s.T(), update.ChatID, string(msg.Key))
		assert.Equal(s.T(), body, msg.Value)
	case _ = <-ctx.Done():
		assert.FailNow(s.T(), "Timeout")
	}

}

func (s *EventsTestSuite) Test_EventsStorage_MemberRemoved() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	consumer, err := s.c.ConsumePartition("test", 0, sarama.OffsetNewest)
	require.NoError(s.T(), err, "create consume partition")
	defer consumer.Close()

	update := models.MemberRemoved{
		UpdateMeta: models.UpdateMeta{
			Timestamp: time.Now().UTC(),
			Audience: []string{
				"253becbb-76b1-4471-9ff3-529462925899",
				"1230cadb-899e-4710-8cdd-0a2f83882712",
			},
		},
		ChatID:   "256e3354-8263-4913-8bdd-345bd04d962e",
		Username: "johndoe",
	}
	store := NewUpdatesStore(s.p, &UpdatesStoreConfig{UpdatesTopic: "test"})
	err = store.MemberRemoved(&update)
	assert.NoError(s.T(), err, "event should be pushed without error")

	select {
	case msg := <-consumer.Messages():
		body, err := proto.Marshal(store.memberRemovedToProtobuf(&update))
		require.NoError(s.T(), err)

		assert.Equal(s.T(), update.ChatID, string(msg.Key))
		assert.Equal(s.T(), body, msg.Value)
	case _ = <-ctx.Done():
		assert.FailNow(s.T(), "Timeout")
	}

}
