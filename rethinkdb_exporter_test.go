package main

/*
  for html coverage report run
  go test -coverprofile=coverage.out  && go tool cover -html=coverage.out
*/

import (
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	r "github.com/dancannon/gorethink"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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

	chM := make(chan prometheus.Metric)
	chD := make(chan *prometheus.Desc)

	go func() {
		e.Collect(chM)
		close(chM)
	}()

	countMetrics := 0
	for m := range chM {

		descString := m.Desc().String()
		log.Printf("descString: %s", descString)
		countMetrics++

		switch m.(type) {
		case prometheus.Gauge:

			g := &dto.Metric{}
			m.Write(g)
			if g.GetGauge() == nil {
				continue
			}
			//			log.Printf("g.String: %s", g.String())
			log.Printf("g: %#v", g)

			for _, l := range g.Label {
				log.Printf("l: %s %s", *l.Name, *l.Value)

			}

		default:
			log.Printf("default: m: %#v", m)
		}

	}
	if countMetrics < 10 {
		t.Errorf("Expected at least 10 metrics, found only %d", countMetrics)
	}

	// descriptions
	go func() {
		e.Describe(chD)
		close(chD)
	}()

	allDescr := []*prometheus.Desc{}
	for d := range chD {
		allDescr = append(allDescr, d)
	}

	wants := []string{"server_client_connections", "cluster_servers_total", "cluster_client_connections", "table_server_disk_read_bytes_per_sec", "table_server_disk_garbage_bytes"}
	for _, w := range wants {
		found := false
		for _, d := range allDescr {
			if strings.Contains(d.String(), w) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("didn't find %s in descriptions", w)
		}
	}

	if len(allDescr) < 10 {
		t.Errorf("Expected at least 10 descriptions, found only %d", len(allDescr))
	}

	log.Printf("done")

}

func TestInvalidDB(t *testing.T) {

	e := NewRethinkDBExporter("localhost:1", "", "test", "")
	e.metrics = map[string]*prometheus.GaugeVec{}

	scrapes := make(chan scrapeResult)
	go e.scrape(scrapes)

	neverTrue := false
	for range scrapes {
		neverTrue = true
	}
	if neverTrue {
		t.Errorf("this shouldn't happen")
	} else {
		log.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ this is expected")
	}
}

func init() {
}
