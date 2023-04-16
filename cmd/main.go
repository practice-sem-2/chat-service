package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
	"github.com/practice-sem-2/auth-tools"
	"github.com/practice-sem-2/user-service/internal/pb/chats"
	"github.com/practice-sem-2/user-service/internal/server"
	storage "github.com/practice-sem-2/user-service/internal/storages"
	usecase "github.com/practice-sem-2/user-service/internal/usecases"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func initLogger(level string) *logrus.Logger {

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint: true,
	})

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logger.SetLevel(logrus.InfoLevel)
		logger.
			WithField("log_level", level).
			Warning("specified invalid log level")
	} else {
		logger.SetLevel(logLevel)
		logger.
			WithField("log_level", level).
			Infof("specified %s log level", logLevel.String())
	}

	return logger
}

func initDB(dsn string, logger *logrus.Logger) *sqlx.DB {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		logger.Fatalf("can't connect to database: %s", err.Error())
	}

	err = db.Ping()

	if err != nil {
		logger.Fatalf("database ping failed: %s", err.Error())
	}

	logger.Info("successfully connected to database")
	return db
}

func initServer(address string, c *usecase.ChatsUsecase, a *auth.VerifierService, v *validator.Validate, logger *logrus.Logger) (*grpc.Server, net.Listener) {

	listener, err := net.Listen("tcp", address)
	logger.Infof("start listening on %s", address)

	if err != nil {
		logger.Fatalf("can't listen to address: %s", err.Error())
	}

	grpcServer := grpc.NewServer()
	chats.RegisterChatServer(grpcServer, server.NewChatServer(c, a, v))

	return grpcServer, listener
}

func initProducer(logger *logrus.Logger) sarama.SyncProducer {
	brokers := viper.GetString("KAFKA_BROKERS")
	if len(brokers) == 0 {
		logger.Fatal("KAFKA_BROKERS environment variable must be defined")
	}

	addrs := strings.Split(brokers, ",")
	config := sarama.NewConfig()
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Timeout = 10 * time.Second
	config.Producer.Return.Successes = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Offsets.AutoCommit.Enable = false
	producer, err := sarama.NewSyncProducer(addrs, config)

	if err != nil {
		logger.WithError(err).Fatalf("can't create producer")
	}

	return producer
}

func main() {
	viper.AutomaticEnv()
	ctx := context.Background()
	defer ctx.Done()

	var host string
	var port int
	var logLevel string

	flag.IntVar(&port, "port", 80, "port on which server will be started")
	flag.StringVar(&host, "host", "0.0.0.0", "host on which server will be started")
	flag.StringVar(&logLevel, "log", "info", "log level")

	flag.Parse()

	logger := initLogger(logLevel)

	db := initDB(viper.GetString("DB_DSN"), logger)
	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			logger.Fatalf("during db connection close an error occurred: %s", err.Error())
		}
	}(db)

	producer := initProducer(logger)

	store := storage.NewRegistry(db, producer, &storage.UpdatesStoreConfig{
		UpdatesTopic: viper.GetString("UPDATES_TOPIC"),
	})

	chatsUsecase := usecase.NewChatsUsecase(store)
	verifier, err := auth.NewVerifierFromFile(viper.GetString("JWT_PUBLIC_KEY_PATH"))

	if err != nil {
		logger.Fatalf("verifier can't read public key: %s", err.Error())
	}

	validate := validator.New()
	address := fmt.Sprintf("%s:%d", host, port)
	srv, lis := initServer(address, chatsUsecase, verifier, validate, logger)
	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func(ctx context.Context) {
		select {
		case sig := <-osSignal:
			srv.GracefulStop()
			logger.Infof("%s caught. Gracefully shutdown", sig.String())
		case <-ctx.Done():
			return
		}
	}(ctx)

	err = srv.Serve(lis)
	if err != nil {
		logger.Fatalf("grpc serving error: %s", err.Error())
	}
}
