package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oggree/logger"
	"github.com/oggree/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
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
	connectionType := config.GetString(fmt.Sprintf("database.%s.type", connectionName))

	host := config.GetString(fmt.Sprintf("database.%s.host", connectionName))
	//hostRead := config.GetString("database.hostRead")

	database := config.GetString(fmt.Sprintf("database.%s.database", connectionName))

	port := config.GetString(fmt.Sprintf("database.%s.port", connectionName))

	username := config.GetString(fmt.Sprintf("database.%s.username", connectionName))
	password := config.GetString(fmt.Sprintf("database.%s.password", connectionName))

	sslMode := config.GetString(fmt.Sprintf("database.%s.sslMode", connectionName))

	var dbDial gorm.Dialector

	var replicas []gorm.Dialector
	var sources []gorm.Dialector

	type NodeConfig struct {
		Host    string `mapstructure:"host"`
		Purpose string `mapstructure:"purpose"`
	}
	var nodes []NodeConfig
	config.UnmarshalKey(fmt.Sprintf("database.%s.replicas", connectionName), &nodes)

	if connectionType == "postgres" {
		dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", username, password, host, port, database, sslMode)
		//dsnRead := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", username, password, hostRead, port, database)

		logger.Info(":::Database Details::: user: " + username + " host : " + host + " port: " + port + " dbname: " + database)
		//logger.Info(":::Database Details For Read::: user: " + username + " host : " + hostRead + " port: " + port + " dbname: " + database)

		dbDial = postgres.Open(dsn)

		for _, node := range nodes {
			repDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", username, password, node.Host, port, database, sslMode)
			dialector := postgres.Open(repDSN)
			if node.Purpose == "write" {
				sources = append(sources, dialector)
				logger.Info(":::Added Postgres Write Node::: host : " + node.Host)
			} else {
				replicas = append(replicas, dialector)
				logger.Info(":::Added Postgres Read Replica::: host : " + node.Host)
			}
		}
	} else if connectionType == "mysql" {
		dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/" + database + "?charset=utf8mb4&parseTime=True&loc=Local"

		logger.Info(":::Database Details::: user: " + username + " host : " + host + " port: " + port + " dbname: " + database)

		dbDial = mysql.Open(dsn)

		for _, node := range nodes {
			repDSN := username + ":" + password + "@tcp(" + node.Host + ":" + port + ")/" + database + "?charset=utf8mb4&parseTime=True&loc=Local"
			dialector := mysql.Open(repDSN)
			if node.Purpose == "write" {
				sources = append(sources, dialector)
				logger.Info(":::Added MySQL Write Node::: host : " + node.Host)
			} else {
				replicas = append(replicas, dialector)
				logger.Info(":::Added MySQL Read Replica::: host : " + node.Host)
			}
		}
	} else if connectionType == "sqlite" {
		dbPath := filepath.Join("data", "db", fmt.Sprintf("%s.db", database))

		if err := os.MkdirAll(filepath.Dir(dbPath), os.ModePerm); err != nil {
			logger.Error("Error creating directory:", err)
		}

		dbDial = sqlite.Open(dbPath)
	} else {
		logger.Error("Invalid connection type: "+connectionType, nil)
	}

	db, err := gorm.Open(dbDial, &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})

	if err != nil {
		logger.Error("Error while connecting to database", err)
		return nil
	}

	if len(replicas) > 0 || len(sources) > 0 {
		db.Use(dbresolver.Register(dbresolver.Config{
			Sources:           sources,
			Replicas:          replicas,
			Policy:            dbresolver.RandomPolicy{},
			TraceResolverMode: true,
		}).
			SetConnMaxIdleTime(10 * time.Minute).
			SetConnMaxLifetime(10 * time.Minute).
			SetMaxIdleConns(10).
			SetMaxOpenConns(30))
	} else {
		// Even if no replicas, keeping the dbresolver helps with uniformity or future dynamic extensions,
		// but it's optional. It was present before, so we can keep it for sources/replicas structure.
		db.Use(dbresolver.Register(dbresolver.Config{
			Sources:           sources,
			Replicas:          replicas,
			Policy:            dbresolver.RandomPolicy{},
			TraceResolverMode: true,
		}).
			SetConnMaxIdleTime(10 * time.Minute).
			SetConnMaxLifetime(10 * time.Minute).
			SetMaxIdleConns(10).
			SetMaxOpenConns(30))
	}

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
