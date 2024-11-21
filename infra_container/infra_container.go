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
	DownFns        []func()
	Down           func()
	Context        context.Context
}

func (ic *IContainer) IContext(ctx context.Context) *IContainer {
	ic.Context = ctx
}
func (ic *IContainer) ICDown() *IContainer {
	var downFns []func()
	down := func() {
		for _, df := range downFns {
			df()
		}
	}
	ic.Down = down
	return ic
}

func (ic *IContainer) ICPostgres() *IContainer {
	pg, err := postgres.NewConnection(ic.Context, &postgres.Config{
		Host:    config.BaseConfig.Postgres.Host,
		Port:    config.BaseConfig.Postgres.Port,
		User:    config.BaseConfig.Postgres.User,
		Pass:    config.BaseConfig.Postgres.Pass,
		DBName:  config.BaseConfig.Postgres.DBName,
		SslMode: config.BaseConfig.Postgres.SslMode,
	})
	if err != nil {
		return nil
	}
	ic.Postgres = pg
	ic.DownFns = append(ic.DownFns, func() {
		ic.Postgres.Close()
	})
	return ic
}

func (ic *IContainer) ICGrpc() *IContainer {
	grpcServerConfig := &grpc.Config{
		Port:        config.BaseConfig.Grpc.Port,
		Host:        config.BaseConfig.Grpc.Host,
		Development: config.IsDevEnv(),
	}
	ic.GrpcServer = grpc.NewGrpcServer(grpcServerConfig)
	ic.DownFns = append(ic.DownFns, func() {
		ic.GrpcServer.GracefulShutdown()
	})
	return ic
}

func (ic *IContainer) ICEcho() *IContainer {
	echoServerConfig := &echoHttp.ServerConfig{
		Port:     config.BaseConfig.Http.Port,
		BasePath: "/api/v1",
		IsDev:    config.IsDevEnv(),
	}
	ic.EchoHttpServer = echoHttp.NewServer(echoServerConfig)
	ic.EchoHttpServer.SetupDefaultMiddlewares()
	ic.DownFns = append(ic.DownFns, func() {
		_ = ic.EchoHttpServer.GracefulShutdown(ic.Context)
	})
	return ic
}

func (ic *IContainer) ICKafka() *IContainer {
	var kw *kafkaProducer.Writer
	var kr *kafkaConsumer.Reader

	kwc := &kafkaProducer.WriterConfig{
		Brokers:      config.BaseConfig.Kafka.ClientBrokers,
		Topic:        config.BaseConfig.Kafka.Topic,
		RequiredAcks: kafka.RequireAll,
	}
	ic.KafkaWriter = kafkaProducer.NewKafkaWriter(kwc)
	ic.DownFns = append(ic.DownFns, func() {
		_ = kw.Client.Close()
	})

	krc := &kafkaConsumer.ReaderConfig{
		Brokers: config.BaseConfig.Kafka.ClientBrokers,
		Topic:   config.BaseConfig.Kafka.Topic,
		GroupID: config.BaseConfig.Kafka.ClientGroupId,
	}
	ic.KafkaReader = kafkaConsumer.NewKafkaReader(krc)
	ic.DownFns = append(ic.DownFns, func() {
		_ = kr.Client.Close()
	})
	return ic
}

func (ic *IContainer) NewIC() (*IContainer, func(), error) {
	//var downFns []func()
	//down := func() {
	//	for _, df := range downFns {
	//		df()
	//	}
	//}
	se := sentry.Init(sentry.ClientOptions{
		Dsn:              config.BaseConfig.Sentry.Dsn,
		TracesSampleRate: 1.0,
		EnableTracing:    config.IsDevEnv(),
	})
	if se != nil {
		_ = fmt.Errorf("can not initialize sentry with error:  %s", se)
	}
	ic.DownFns = append(ic.DownFns, func() {
		sentry.Flush(2 * time.Second)
	})

	//grpcServerConfig := &grpc.Config{
	//	Port:        config.BaseConfig.Grpc.Port,
	//	Host:        config.BaseConfig.Grpc.Host,
	//	Development: config.IsDevEnv(),
	//}
	//grpcServer := grpc.NewGrpcServer(grpcServerConfig)
	//downFns = append(downFns, func() {
	//	ic.GrpcServer.GracefulShutdown()
	//})

	//echoServerConfig := &echoHttp.ServerConfig{
	//	Port:     config.BaseConfig.Http.Port,
	//	BasePath: "/api/v1",
	//	IsDev:    config.IsDevEnv(),
	//}
	//echoServer := echoHttp.NewServer(echoServerConfig)
	//echoServer.SetupDefaultMiddlewares()
	//downFns = append(downFns, func() {
	//	_ = ic.EchoHttpServer.GracefulShutdown(ctx)
	//})

	//pg, err := postgres.NewConnection(ctx, &postgres.Config{
	//	Host:    config.BaseConfig.Postgres.Host,
	//	Port:    config.BaseConfig.Postgres.Port,
	//	User:    config.BaseConfig.Postgres.User,
	//	Pass:    config.BaseConfig.Postgres.Pass,
	//	DBName:  config.BaseConfig.Postgres.DBName,
	//	SslMode: config.BaseConfig.Postgres.SslMode,
	//})
	//if err != nil {
	//	return nil, down, fmt.Errorf("can not connect to database using sqlx with error: %s", err)
	//}
	//downFns = append(downFns, func() {
	//	ic.Postgres.Close()
	//})

	//var kw *kafkaProducer.Writer
	//var kr *kafkaConsumer.Reader

	//if config.BaseConfig.Kafka.Enabled {
	//	kwc := &kafkaProducer.WriterConfig{
	//		Brokers:      config.BaseConfig.Kafka.ClientBrokers,
	//		Topic:        config.BaseConfig.Kafka.Topic,
	//		RequiredAcks: kafka.RequireAll,
	//	}
	//	kw = kafkaProducer.NewKafkaWriter(kwc)
	//	downFns = append(downFns, func() {
	//		_ = kw.Client.Close()
	//	})
	//
	//	krc := &kafkaConsumer.ReaderConfig{
	//		Brokers: config.BaseConfig.Kafka.ClientBrokers,
	//		Topic:   config.BaseConfig.Kafka.Topic,
	//		GroupID: config.BaseConfig.Kafka.ClientGroupId,
	//	}
	//	kr = kafkaConsumer.NewKafkaReader(krc)
	//	downFns = append(downFns, func() {
	//		_ = kr.Client.Close()
	//	})
	//}

	nic := &IContainer{
		Config:         config.BaseConfig,
		Logger:         logger.Zap,
		Postgres:       ic.Postgres,
		GrpcServer:     ic.GrpcServer,
		EchoHttpServer: ic.EchoHttpServer,
		KafkaWriter:    ic.KafkaWriter,
		KafkaReader:    ic.KafkaReader,
	}

	return nic, ic.Down, nil
}
