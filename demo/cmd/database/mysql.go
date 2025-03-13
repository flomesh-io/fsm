// Package database allows the bookwarehouse service to store
// total books data into MySQL persistent storage
package database

import (
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/flomesh-io/fsm/pkg/logger"

	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	log = logger.NewPretty("mysql")
)

// Refer to https://github.com/flomesh-io/fsm-docs/blob/main/manifests/apps/mysql.yaml for database setup
const (
	dbuser       = "root"
	dbpass       = "mypassword"
	dbport       = 3306
	dbname       = "booksdemo"
	mysqlSvcName = "mysql.bookwarehouse"
)

// GetMySQLConnection returns a MySQL connection using default configuration
func GetMySQLConnection() (*gorm.DB, error) {
	ips, err := net.LookupIP(mysqlSvcName)
	if err != nil {
		log.Error().Msgf("Failed to lookup IP for %s: %s", mysqlSvcName, err)
		return nil, err
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP found for %s", mysqlSvcName)
	}

	// no matter how many IPs are returned, we will use the first one
	ip := ips[0].String()
	log.Info().Msgf("Using IP: %s", ip)

	cfg := gomysql.NewConfig()
	cfg.User = dbuser
	cfg.Passwd = dbpass
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", ip, dbport)
	cfg.DBName = dbname
	cfg.Timeout = 20 * time.Second
	cfg.Params = map[string]string{"charset": "utf8"}

	if sqlDB, err := sql.Open("mysql", cfg.FormatDSN()); err != nil {
		return nil, err
	} else {
		return gorm.Open(mysql.New(mysql.Config{
			Conn: sqlDB,
		}), &gorm.Config{})
	}

	//connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&timeout=20s", dbuser, dbpass, "mysql.bookwarehouse", dbport, dbname)
	//db, err := gorm.Open(mysql..Open(connStr), &gorm.Config{})
	//
	//return db, err
}
