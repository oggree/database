package database

import (
	"fmt"
	"sync"
	"time"

	"github.com/oggree/logger"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

type DatabaseClient struct {
	DB *gorm.DB
}

var singletonInstanceList = make(map[string]*DatabaseClient)

func GetInstance(connectionName string) *DatabaseClient {
	if singletonInstanceList[connectionName] == nil {
		var once sync.Once
		once.Do(func() {
			singletonInstanceList[connectionName] = &DatabaseClient{DB: GetSQLClient(connectionName)}
		})
	}
	return singletonInstanceList[connectionName]
}

func GetSQLClient(connectionName string) *gorm.DB {
	connectionType := viper.GetString(fmt.Sprintf("database.%s.type", connectionName))

	host := viper.GetString(fmt.Sprintf("database.%s.host", connectionName))
	//hostRead := viper.GetString("database.hostRead")

	database := viper.GetString(fmt.Sprintf("database.%s.database", connectionName))

	port := viper.GetString(fmt.Sprintf("database.%s.port", connectionName))

	username := viper.GetString(fmt.Sprintf("database.%s.username", connectionName))
	password := viper.GetString(fmt.Sprintf("database.%s.password", connectionName))

	sslMode := viper.GetString(fmt.Sprintf("database.%s.sslMode", connectionName))

	var dbDial gorm.Dialector

	replicas := []gorm.Dialector{}

	if connectionType == "postgres" {
		dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", username, password, host, port, database, sslMode)
		//dsnRead := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", username, password, hostRead, port, database)

		logger.Info(":::Database Details::: user: " + username + " host : " + host + " port: " + port + " dbname: " + database)
		//logger.Info(":::Database Details For Read::: user: " + username + " host : " + hostRead + " port: " + port + " dbname: " + database)

		dbDial = postgres.Open(dsn)
	} else if connectionType == "mysql" {
		dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/" + database + "?charset=utf8mb4&parseTime=True&loc=Local"

		logger.Info(":::Database Details::: user: " + username + " host : " + host + " port: " + port + " dbname: " + database)

		dbDial = mysql.Open(dsn)
	} else {

	}

	db, err := gorm.Open(dbDial, &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})

	if err != nil {
		logger.Error("Error while connecting to PostgreSQL", err)
		return nil
	}

	db.Use(dbresolver.Register(dbresolver.Config{
		// use `db2` as sources, `db3`, `db4` as replicas
		//Sources:  []gorm.Dialector{mysql.Open("db2_dsn")},
		Replicas: replicas,
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
