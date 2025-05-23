// Package main implements the bookwarehouse application
// This create a service which has both inbound as well as outbound service policies
// i.e. bookbuyer makes a GET call to bookstore, bookstore makes a POST call to bookwarehouse
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"

	"github.com/flomesh-io/fsm/demo/cmd/common"
	"github.com/flomesh-io/fsm/demo/cmd/database"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log      = logger.NewPretty("bookwarehouse")
	identity = flag.String("ident", "unidentified", "the identity of the container where this demo app is running (VM, K8s, etc)")
	port     = flag.Int("port", 14001, "port on which this app is listening for incoming HTTP")
	db       *gorm.DB
)

// Record stores key value pairs
type Record struct {
	Key      string `gorm:"primaryKey"`
	ValueInt int64
}

const (
	keyTotalBooks = "total-books"
)

func getIdentity() string {
	ident := os.Getenv("IDENTITY")
	if ident == "" {
		if identity != nil {
			ident = *identity
		}
	}
	return ident
}

func getBooksStockedRecord() Record {
	var record Record
	db.Where(&Record{Key: keyTotalBooks}).First(&record)
	return record
}

func setHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.IdentityHeader, getIdentity())

	if r == nil {
		return
	}

	for _, header := range common.GetTracingHeaderKeys() {
		if v := r.Header.Get(header); v != "" {
			w.Header().Set(header, v)
		}
	}
}

// restockBooks decreases the balance of the given bookwarehouse account.
func restockBooks(w http.ResponseWriter, r *http.Request) {
	setHeaders(w, r)
	var numberOfBooks int
	err := json.NewDecoder(r.Body).Decode(&numberOfBooks)
	if err != nil {
		log.Error().Err(err).Msg("Could not decode request body")
		numberOfBooks = 0
	}

	record := getBooksStockedRecord()
	record.ValueInt += int64(numberOfBooks)
	totalBooks := int(record.ValueInt)
	db.Save(record)

	_, _ = fmt.Fprintf(w, "{\"restocked\":%d}", numberOfBooks)
	log.Info().Msgf("Restocking bookstore with %d new books; Total so far: %d", numberOfBooks, totalBooks)
	if totalBooks >= 3 {
		fmt.Println(common.Success)
		log.Info().Msgf(common.Success)
	}
}

func initDb() {
	var err error
	for {
		db, err = database.GetMySQLConnection()

		if err != nil {
			log.Error().Msgf("Booksdemo database is not ready: %v. Wait for 10s ...", err)
			time.Sleep(10 * time.Second)
		} else {
			break
		}
	}

	log.Info().Msg("Booksdemo database is connected.")
	err = db.Migrator().AutoMigrate(&Record{})
	if err != nil {
		log.Fatal().Msgf("Database migration failed. %v", err)
	}

	var record Record
	if result := db.Where(&Record{Key: keyTotalBooks}).First(&record); result.RowsAffected == 0 {
		// initialize record
		record = Record{
			Key:      keyTotalBooks,
			ValueInt: 0,
		}

		result = db.Create(&record)
		log.Info().Msgf("Initial %s record created. %v, %v, %v", keyTotalBooks, record, result.RowsAffected, result.Error)
	}
}

func main() {
	flag.Parse()

	initDb()

	//initializing router
	router := mux.NewRouter()

	router.HandleFunc(fmt.Sprintf("/%s", common.RestockWarehouseURL), restockBooks).Methods("POST")
	router.HandleFunc("/", restockBooks).Methods("POST")
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
	log.Info().Msgf("Starting BookWarehouse HTTP server on port %d", *port)
	//#nosec G114: Use of net/http serve function that has no support for setting timeouts
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), router)
	log.Fatal().Err(err).Msgf("Failed to start BookWarehouse HTTP server on port %d", *port)
}
