package main

/*
  for html coverage report run
  go test -coverprofile=coverage.out && go tool cover -html=coverage.out
*/

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

var (
	names = []string{"john", "paul", "ringo", "george"}
)

func setupDB(t *testing.T) (sess *r.Session, dbName string, err error) {

	dbName = fmt.Sprintf("db%d", int32(time.Now().Unix()))

	sess, err = r.Connect(r.ConnectOpts{
		Address:  os.Getenv("RETHINKDB_URI"),
		Database: dbName,
	})
	if err != nil {
		return
	}

	_, err = r.DBCreate(dbName).Run(sess)
	if err != nil {
		t.Errorf("couldn't create table, err: %s ", err)
		return
	}
	r.DB(dbName).Wait().Run(sess)

	r.DB(dbName).TableCreate("test1", r.TableCreateOpts{PrimaryKey: "id"}).Exec(sess)
	r.DB(dbName).TableCreate("test2", r.TableCreateOpts{PrimaryKey: "id"}).Exec(sess)

	res, err := r.DB(dbName).TableList().Run(sess)
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
		t.Errorf("table list off, %d    %v ", len(tables), tables)
		return
	}

	for idx, n := range names {

		var rec = struct {
			Name string
			Age  int
		}{Name: n, Age: 56 + idx}

		r.DB(dbName).Table("test1").Insert(rec).RunWrite(sess)
	}

	sess.Use(dbName)
	return
}

func getTestExporter() *Exporter {
	e := NewRethinkDBExporter(
		os.Getenv("RETHINKDB_URI"),
		"", "", "", "test",
		Options{countRows: true, getTableStats: true, metricPath: "/metrics"},
	)
	e.metrics = map[string]*prometheus.GaugeVec{}
	return e
}

func TestMetrics(t *testing.T) {
	sess, dbName, err := setupDB(t)
	if err != nil {
		t.Errorf("DB setup borked")
		return
	}
	defer r.DBDrop(dbName).Run(sess)

	e := getTestExporter()

	chM := make(chan prometheus.Metric)
	chD := make(chan *prometheus.Desc)

	go func() {
		e.Collect(chM)
		close(chM)
	}()

	countMetrics := 0
	countMetricsForDB := 0
	for m := range chM {

		// descString := m.Desc().String()
		// log.Printf("descString: %s", descString)
		countMetrics++

		switch m.(type) {
		case prometheus.Gauge:

			g := &dto.Metric{}
			m.Write(g)
			if g.GetGauge() == nil {
				continue
			}

			if len(g.Label) == 4 && *g.Label[1].Name == "db" && *g.Label[1].Value == dbName {
				countMetricsForDB++
			}

		default:
			log.Printf("default: m: %#v", m)
		}

	}

	expectedCountMetrics := 53
	if countMetrics != expectedCountMetrics {
		t.Errorf("Expected %d metrics, found %d", expectedCountMetrics, countMetrics)
	}

	expectedCountMetricsForDB := 24
	if countMetricsForDB != expectedCountMetricsForDB {
		t.Errorf("Expected %d metrics, found %d", expectedCountMetricsForDB, countMetricsForDB)
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

	wants := []string{"server_client_connections", "cluster_servers_total", "cluster_client_connections", "table_server_disk_read_bytes_per_sec", "table_server_disk_garbage_bytes", "up"}
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
}

func TestMetricsNoRowCounting(t *testing.T) {
	sess, dbName, err := setupDB(t)
	if err != nil {
		t.Errorf("DB setup borked")
		return
	}
	defer r.DBDrop(dbName).Run(sess)

	e := getTestExporter()
	e.options.countRows = false

	chM := make(chan prometheus.Metric)
	chD := make(chan *prometheus.Desc)

	go func() {
		e.Collect(chM)
		close(chM)
	}()

	countMetrics := 0
	countMetricsForDB := 0
	for m := range chM {

		countMetrics++

		switch m.(type) {
		case prometheus.Gauge:

			g := &dto.Metric{}
			m.Write(g)
			if g.GetGauge() == nil {
				continue
			}

			if len(g.Label) == 4 && *g.Label[1].Name == "db" && *g.Label[1].Value == dbName {
				countMetricsForDB++
			}

		default:
			log.Printf("default: m: %#v", m)
		}
	}

	expectedCountMetrics := 51
	if countMetrics != expectedCountMetrics {
		t.Errorf("Expected %d metrics, found %d", expectedCountMetrics, countMetrics)
	}

	expectedCountMetricsForDB := 24
	if countMetricsForDB != expectedCountMetricsForDB {
		t.Errorf("Expected %d metrics, found %d", expectedCountMetricsForDB, countMetricsForDB)
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
}

func TestInvalidDB(t *testing.T) {
	e := getTestExporter()
	e.addrs = []string{"localhost:1"}

	scrapes := make(chan scrapeResult)
	go e.scrape(scrapes)

	neverTrue := false
	for x := range scrapes {
		if x.Name != "up" {
			neverTrue = true
		}
	}
	if neverTrue {
		t.Errorf("this shouldn't happen")
	} else {
		log.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ this is expected")
	}
}

func TestHttpHandlers(t *testing.T) {
	e := getTestExporter()

	ts := httptest.NewServer(e)
	defer ts.Close()

	for _, tst := range []struct {
		path string
		want string
	}{
		{
			path: "/",
			want: `<head><title>RethinkDB exporter`,
		},
		{
			path: "/health",
			want: `ok`,
		},
		{
			path: "/metrics",
			want: `up 1`,
		},
	} {
		t.Run(fmt.Sprintf("path: %s", tst.path), func(t *testing.T) {
			body := downloadURL(t, ts.URL+tst.path)
			if !strings.Contains(body, tst.want) {
				t.Fatalf(`error, expected string "%s" in body, got body: \n\n%s`, tst.want, body)
			}
		})
	}
}

func downloadURL(t *testing.T, u string) string {
	_, res := downloadURLWithStatusCode(t, u)
	return res
}

func downloadURLWithStatusCode(t *testing.T, u string) (int, string) {
	t.Logf("downloadURL() %s", u)

	resp, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	return resp.StatusCode, string(body)
}
