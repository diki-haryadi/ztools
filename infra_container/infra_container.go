package infraContainer

import (
	"context"
	"fmt"
	"time"

	sentry "github.com/getsentry/sentry-go"
	kafka "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/diki-haryadi/ztools/config"
	"github.com/diki-haryadi/ztools/grpc"
	echoHttp "github.com/diki-haryadi/ztools/http/echo"
	kafkaConsumer "github.com/diki-haryadi/ztools/kafka/consumer"
	kafkaProducer "github.com/diki-haryadi/ztools/kafka/producer"
	"github.com/diki-haryadi/ztools/logger"
	"github.com/diki-haryadi/ztools/postgres"
)

type IContainer struct {
	Config         *config.Config
	Logger         *zap.Logger
	Postgres       *postgres.Postgres
	GrpcServer     grpc.Server
	EchoHttpServer echoHttp.ServerInterface
	KafkaWriter    *kafkaProducer.Writer
	KafkaReader    *kafkaConsumer.Reader
}

func NewIC(ctx context.Context) (*IContainer, func(), error) {
	var downFns []func()
	down := func() {
		for _, df := range downFns {
			df()
		}
	}

	se := sentry.Init(sentry.ClientOptions{
		Dsn:              config.BaseConfig.Sentry.Dsn,
		TracesSampleRate: 1.0,
		EnableTracing:    config.IsDevEnv(),
	})
	if se != nil {
		_ = fmt.Errorf("can not initialize sentry with error:  %s", se)
	}
	downFns = append(downFns, func() {
		sentry.Flush(2 * time.Second)
	})

	grpcServerConfig := &grpc.Config{
		Port:        config.BaseConfig.Grpc.Port,
		Host:        config.BaseConfig.Grpc.Host,
		Development: config.IsDevEnv(),
	}
	grpcServer := grpc.NewGrpcServer(grpcServerConfig)
	downFns = append(downFns, func() {
		grpcServer.GracefulShutdown()
	})

	echoServerConfig := &echoHttp.ServerConfig{
		Port:     config.BaseConfig.Http.Port,
		BasePath: "/api/v1",
		IsDev:    config.IsDevEnv(),
	}
	echoServer := echoHttp.NewServer(echoServerConfig)
	echoServer.SetupDefaultMiddlewares()
	downFns = append(downFns, func() {
		_ = echoServer.GracefulShutdown(ctx)
	})

	pg, err := postgres.NewConnection(ctx, &postgres.Config{
		Host:    config.BaseConfig.Postgres.Host,
		Port:    config.BaseConfig.Postgres.Port,
		User:    config.BaseConfig.Postgres.User,
		Pass:    config.BaseConfig.Postgres.Pass,
		DBName:  config.BaseConfig.Postgres.DBName,
		SslMode: config.BaseConfig.Postgres.SslMode,
	})
	if err != nil {
		return nil, down, fmt.Errorf("can not connect to database using sqlx with error: %s", err)
	}
	downFns = append(downFns, func() {
		pg.Close()
	})

	var kw *kafkaProducer.Writer
	var kr *kafkaConsumer.Reader

	if config.BaseConfig.Kafka.Enabled {
		kwc := &kafkaProducer.WriterConfig{
			Brokers:      config.BaseConfig.Kafka.ClientBrokers,
			Topic:        config.BaseConfig.Kafka.Topic,
			RequiredAcks: kafka.RequireAll,
		}
		kw = kafkaProducer.NewKafkaWriter(kwc)
		downFns = append(downFns, func() {
			_ = kw.Client.Close()
		})

		krc := &kafkaConsumer.ReaderConfig{
			Brokers: config.BaseConfig.Kafka.ClientBrokers,
			Topic:   config.BaseConfig.Kafka.Topic,
			GroupID: config.BaseConfig.Kafka.ClientGroupId,
		}
		kr = kafkaConsumer.NewKafkaReader(krc)
		downFns = append(downFns, func() {
			_ = kr.Client.Close()
		})
	}

	ic := &IContainer{
		Config:         config.BaseConfig,
		Logger:         logger.Zap,
		Postgres:       pg,
		GrpcServer:     grpcServer,
		EchoHttpServer: echoServer,
		KafkaWriter:    kw,
		KafkaReader:    kr,
	}

	return ic, down, nil
}
