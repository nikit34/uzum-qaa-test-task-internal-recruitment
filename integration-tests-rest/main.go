package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type jsonError struct {
	Code string `json:"code"`
	Msg string `json:"msg"`
}

type Book struct {
	ISBN string `json:"isbn"`
	Title string `json:"title"`
	Image string `json:"image"`
	Genre string `json:"genre"`
	YearPublished int `json:"year_published"`
}

type health struct {
	Status string `json:"status"`
	Messages []string `json:"messages"`
}

func main() {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error making DB connected: %s", err.Error())
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
	    log.Println(err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations/",
		"music",
		driver,
	)
	if err != nil {
		log.Println(err)
	}
	if err := m.Steps(2); err != nil {
		log.Println(err)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS book (id serial PRIMARY KEY, isbn VARCHAR ( 255 ) NOT NULL, name VARCHAR ( 255 ) NOT NULL, image VARCHAR ( 255 ) NOT NULL, genre VARCHAR ( 255 ) NOT NULL, year_published int NOT NULL)")

    if err != nil {
        log.Fatal(err)
    }

	r := mux.NewRouter()

	r.HandleFunc(
		"/healthcheck",
		func(w http.ResponseWriter, r *http.Request) {
			h := health{
				Status:   "OK",
				Messages: []string{},
			}

			b, _ := json.Marshal(h)

			w.WriteHeader(http.StatusOK)
			w.Write(b)
		},
	)
	r.HandleFunc(
		"/book/{isbn}",
		func(w http.ResponseWriter, r *http.Request) {
			v := mux.Vars(r)

			isbn := v["isbn"]

			b := Book{}
			row := db.QueryRow("SELECT isbn, name, image, genre, year_published FROM book WHERE isbn = $1", isbn)
			err := row.Scan(
				&b.ISBN,
				&b.Title,
				&b.Image,
				&b.Genre,
				&b.YearPublished,
				)

			if err != nil {
				e := jsonError{
					Code: "001",
					Msg:  fmt.Sprintf("No book with ISBN %s", isbn),
				}

				body, _ := json.Marshal(e)

				w.WriteHeader(http.StatusNotFound)
				w.Write(body)
				return
			}

			body, _ := json.Marshal(b)

			w.WriteHeader(http.StatusOK)
			w.Write(body)
		},
	)

	s := http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	s.ListenAndServe()
}
