package main

import (
	"context"
	"flag"
	"fmt"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/practice-sem-2/user-service/internal/pb"
	"github.com/practice-sem-2/user-service/internal/server"
	storage "github.com/practice-sem-2/user-service/internal/storages"
	usecase "github.com/practice-sem-2/user-service/internal/usecases"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
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

func initServer(address string, useCases *usecase.ChatsUsecase, logger *logrus.Logger) (*grpc.Server, net.Listener) {

	listener, err := net.Listen("tcp", address)
	logger.Infof("start listening on %s", address)

	if err != nil {
		logger.Fatalf("can't listen to address: %s", err.Error())
	}

	grpcServer := grpc.NewServer()
	pb.RegisterChatServer(grpcServer, server.NewChatServer(useCases))

	return grpcServer, listener
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

	store := storage.NewRegistry(db)
	useCases := usecase.NewChatsUsecase(store)

	address := fmt.Sprintf("%s:%d", host, port)
	srv, lis := initServer(address, useCases, logger)
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

	err := srv.Serve(lis)
	if err != nil {
		logger.Fatalf("grpc serving error: %s", err.Error())
	}
}
