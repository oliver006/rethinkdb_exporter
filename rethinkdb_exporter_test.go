package main

/*
  for html coverage report run
  go test -coverprofile=coverage.out  && go tool cover -html=coverage.out
*/

import (
	"fmt"
	"testing"
	"time"

	r "github.com/dancannon/gorethink"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	names = []string{"john", "paul", "ringo", "george"}
)

func setupDB(t *testing.T) (sess *r.Session, dbName string, err error) {

	dbName = fmt.Sprintf("db%d", int32(time.Now().Unix()))

	sess, err = r.Connect(r.ConnectOpts{
		Address:  "localhost:28015",
		Database: dbName,
	})
	if err != nil {
		return
	}

	_, err = r.DbCreate(dbName).Run(sess)
	if err != nil {
		t.Errorf("couldn't create table, err: %s ", err)
		return
	}

	sess.Use(dbName)

	r.Db(dbName).TableCreate("test1", r.TableCreateOpts{PrimaryKey: "id"}).Run(sess)
	r.Db(dbName).TableCreate("test2", r.TableCreateOpts{PrimaryKey: "id"}).Run(sess)

	res, err := r.Db(dbName).TableList().Run(sess)
	if err != nil {
		t.Errorf("couldn't load table list, err: %s ", err)
		return
	}

	var tables []interface{}
	if err = res.All(&tables); err != nil {
		t.Errorf("couldn't load table list, err: %s ", err)
		return
	}

	if len(tables) != 2 {
		t.Errorf("table list off, %v ", tables)
		return
	}

	for idx, n := range names {

		var rec = struct {
			Name string
			Age  int
		}{Name: n, Age: 56 + idx}

		r.Db(dbName).Table("test1").Insert(rec).RunWrite(sess)
	}

	return
}

func TestMetrics(t *testing.T) {

	sess, dbName, err := setupDB(t)
	if err != nil {
		t.Errorf("DB setup borked")
		return
	}
	defer r.DbDrop(dbName).Run(sess)

	e := NewRethinkDBExporter("localhost:28015", "", "test", "")
	e.metrics = map[string]*prometheus.GaugeVec{}

	scrapes := make(chan scrapeResult)
	go e.scrape(scrapes)

	for s := range scrapes {

		metric := s.Name
		value := s.Value
		table := s.Table
		db := s.DB

		if db != dbName {
			continue
		}

		// cluster and server wide metrics
		switch metric {
		case "cluster_client_connections", "cluster_clients_active", "cluster_servers_total", "cluster_tables_total", "cluster_replicas_total":
			{
				if int(value) < 1 {
					t.Errorf("value for %s should >0", metric)
					return
				}
			}
		case "server_client_connections", "server_clients_active":
			{
				if int(value) < 1 {
					t.Errorf("value for %s should >0", metric)
					return
				}
			}
		}

		if table != "test1" {
			continue
		}

		// table wide metrics
		switch metric {
		case "table_docs_total":
			{
				if int(value) != len(names) {
					t.Errorf("missing records from test1, should be $5, is %d", len(names), int(value))
					return
				}
			}
		}
	}

	scrapes = make(chan scrapeResult)
	go e.scrape(scrapes)
	e.setMetrics(scrapes)

	if len(e.metrics) < 10 {
		t.Errorf("len(e.metrics) should be > 10, is %d", len(e.metrics))
		return
	}
}

func TestInvalidDB(t *testing.T) {

	e := NewRethinkDBExporter("localhost:1111", "", "test", "")
	e.metrics = map[string]*prometheus.GaugeVec{}

	scrapes := make(chan scrapeResult)
	go e.scrape(scrapes)

	neverTrue := false
	for _ = range scrapes {
		neverTrue = true
	}
	if neverTrue {
		t.Errorf("this shouldn't happen")
	}
}

func init() {
}
