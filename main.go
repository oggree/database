package database

import (
	"fmt"
	"sync"
	"time"

	"github.com/oggree/logger"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

type DatabaseClient struct {
	DB *gorm.DB
}

var singleton *DatabaseClient

func GetInstance() *DatabaseClient {
	if singleton == nil {
		var once sync.Once
		once.Do(func() {
			singleton = &DatabaseClient{DB: GetPostgreSQLClient()}
		})
	}
	return singleton
}

func GetPostgreSQLClient() *gorm.DB {
	host := viper.GetString("database.host")
	//hostRead := viper.GetString("database.hostRead")
	database := viper.GetString("database.database")

	port := viper.GetString("database.port")

	username := viper.GetString("database.username")
	password := viper.GetString("database.password")

	sslMode := viper.GetString("database.sslMode")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", username, password, host, port, database, sslMode)
	//dsnRead := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", username, password, hostRead, port, database)

	logger.Info(":::Database Details::: user: " + username + " host : " + host + " port: " + port + " dbname: " + database)
	//logger.Info(":::Database Details For Read::: user: " + username + " host : " + hostRead + " port: " + port + " dbname: " + database)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})

	if err != nil {
		logger.Error("Error while connecting to PostgreSQL", err)
		return nil
	}

	db.Use(dbresolver.Register(dbresolver.Config{
		// use `db2` as sources, `db3`, `db4` as replicas
		//Sources:  []gorm.Dialector{mysql.Open("db2_dsn")},
		Replicas: []gorm.Dialector{
			//postgres.Open(dsnRead)
		},
		// sources/replicas load balancing policy
		Policy: dbresolver.RandomPolicy{},
		// print sources/replicas mode in logger
		TraceResolverMode: true,
	}).
		SetConnMaxIdleTime(10 * time.Minute).
		SetConnMaxLifetime(10 * time.Minute).
		SetMaxIdleConns(10).
		SetMaxOpenConns(30))

	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("Error while getting database/sql interface from gorm instance", err)
		return nil
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(30)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return db
}
