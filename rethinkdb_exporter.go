package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

type Exporter struct {
	sync.RWMutex

	addrs       []string
	auth        string
	user        string
	pass        string
	clusterName string

	duration     prometheus.Gauge
	scrapeError  prometheus.Gauge
	totalScrapes prometheus.Counter
	metrics      map[string]*prometheus.GaugeVec

	mux      *http.ServeMux
	registry *prometheus.Registry
	options  Options
}

type Options struct {
	namespace  string
	metricPath string

	getTableStats bool
	countRows     bool
}

func NewRethinkDBExporter(addr, auth, user, pass, clusterName string, opt Options) *Exporter {
	fmt.Printf("opts: %#v \n", opt)

	e := &Exporter{
		addrs:       strings.Split(addr, ","),
		auth:        auth,
		user:        user,
		pass:        pass,
		clusterName: clusterName,

		options: opt,

		registry: prometheus.NewRegistry(),

		duration: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: opt.namespace,
			Name:      "exporter_last_scrape_duration_seconds",
			Help:      "The last scrape duration.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: opt.namespace,
			Name:      "exporter_scrapes_total",
			Help:      "Current total rethinkdb scrapes.",
		}),
		scrapeError: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: opt.namespace,
			Name:      "exporter_last_scrape_error",
			Help:      "The last scrape error status.",
		}),
		metrics: map[string]*prometheus.GaugeVec{},
	}

	e.registry.MustRegister(e)

	e.mux = http.NewServeMux()
	e.mux.HandleFunc("/", e.indexHandler)
	e.mux.HandleFunc("/health", e.healthHandler)
	e.mux.Handle(opt.metricPath, promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{}))

	return e
}

func (e *Exporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.mux.ServeHTTP(w, r)
}

func (e *Exporter) healthHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`ok`))
}

func (e *Exporter) indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<html>
<head><title>RethinkDB exporter</title></head>
<body>
<h1>RethinkDB exporter</h1>
<p><a href='` + e.options.metricPath + `'>Metrics</a></p>
</body>
</html>
`))
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.metrics {
		m.Describe(ch)
	}

	ch <- e.duration.Desc()
	ch <- e.totalScrapes.Desc()
	ch <- e.scrapeError.Desc()
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	scrapes := make(chan scrapeResult)

	e.Lock()
	defer e.Unlock()

	// todo: need to clear metrics cause of eg. deleted tables.
	// but can we do better? delete selectively ?
	e.metrics = map[string]*prometheus.GaugeVec{}

	go e.scrape(scrapes)

	e.setMetrics(scrapes)
	ch <- e.duration
	ch <- e.totalScrapes
	ch <- e.scrapeError
	e.collectMetrics(ch)
}

type queryEngine struct {
	ClientConnections float64 `gorethink:"client_connections"      exporter:"cluster,server"`
	ClientsActive     float64 `gorethink:"clients_active"          exporter:"cluster,server"`
	QueriesPerSec     float64 `gorethink:"queries_per_sec"         exporter:"cluster,server"`
	QueriesTotal      float64 `gorethink:"queries_total"           exporter:"server"`
	ReadDocsPerSec    float64 `gorethink:"read_docs_per_sec"       exporter:"all"`
	ReadDocsTotal     float64 `gorethink:"read_docs_total"         exporter:"server,table_server"`
	WrittenDocsPerSec float64 `gorethink:"written_docs_per_sec"    exporter:"all"`
	WrittenDocsTotal  float64 `gorethink:"written_docs_total"      exporter:"server,table"`
}

type storageEngine struct {
	Cache struct {
		InUseBytes float64 `gorethink:"in_use_bytes"`
	}
	Disk struct {
		ReadBytesPerSec    float64 `gorethink:"read_bytes_per_sec"`
		ReadBytesTotal     float64 `gorethink:"read_bytes_total"`
		WrittenBytesPerSec float64 `gorethink:"written_bytes_per_sec"`
		WrittenBytesTotal  float64 `gorethink:"written_bytes_total"`
		SpaceUsage         struct {
			DataBytes         float64 `gorethink:"data_bytes"`
			MetadataBytes     float64 `gorethink:"metadata_bytes"`
			GarbageBytes      float64 `gorethink:"garbage_bytes"`
			PreallocatedBytes float64 `gorethink:"preallocated_bytes"`
		} `gorethink:"space_usage"`
	}
}

type scrapeResult struct {
	Name   string
	Value  float64
	Server string
	DB     string
	Table  string
}

type Stat struct {
	ID            []string      `gorethink:"id"`
	QueryEngine   queryEngine   `gorethink:"query_engine,omitempty" `
	StorageEngine storageEngine `gorethink:"storage_engine,omitempty" `

	Server string `gorethink:"server,omitempty" `
	DB     string `gorethink:"db,omitempty" `
	Table  string `gorethink:"table,omitempty" `

	Error string `gorethink:"error,omitempty" `

	scrapes chan<- scrapeResult
}

func (s *Stat) newScrapeResult(name string, val float64) scrapeResult {
	return scrapeResult{
		Name:   name,
		Value:  val,
		DB:     s.DB,
		Server: s.Server,
		Table:  s.Table}
}

func includeMetric(prefix, tag string) bool {

	if len(tag) == 0 || tag == "" || tag == "all" {
		return true
	}

	prefixes := strings.Split(tag, ",")
	for _, p := range prefixes {
		if p == prefix {
			return true
		}
	}
	return false
}

func (s *Stat) extracStructMetrics(prefix string, src interface{}, scrapes chan<- scrapeResult) {
	st := reflect.TypeOf(src)
	v := reflect.ValueOf(src)
	for pos := 0; pos < st.NumField(); pos++ {

		if !v.Field(pos).Type().ConvertibleTo(reflect.TypeOf(float64(0))) {
			continue
		}

		metric := st.Field(pos).Tag.Get("gorethink")
		tag := st.Field(pos).Tag.Get("exporter")
		if !includeMetric(prefix, tag) {
			continue
		}

		scrapes <- s.newScrapeResult(fmt.Sprintf("%s_%s", prefix, metric), v.Field(pos).Float())
	}
}

func (s *Stat) extractStorageEngineStats(scrapes chan<- scrapeResult) {
	s.extracStructMetrics("table_server_cache", s.StorageEngine.Cache, scrapes)
	s.extracStructMetrics("table_server_disk", s.StorageEngine.Disk, scrapes)
	s.extracStructMetrics("table_server_disk", s.StorageEngine.Disk.SpaceUsage, scrapes)
}

func (s *Stat) extractQueryEngineStats(scrapes chan<- scrapeResult, countRows, getTableStats bool) {
	prefix := s.ID[0]
	if (prefix == "table" || prefix == "table_server") && !getTableStats {
		return
	}
	s.extracStructMetrics(prefix, s.QueryEngine, scrapes)
}

func extractAllMetrics(sess *r.Session, scrapes chan<- scrapeResult, countRows, getTableStats bool) error {

	res, err := r.Table("stats").Run(sess)
	if err != nil {
		return err
	}

	countServers := 0
	countServerErrors := 0
	countTables := 0
	countReplicas := 0

	s := Stat{}
	for res.Next(&s) {

		if s.Error != "" {
			countServerErrors++
			continue
		}

		s.extractQueryEngineStats(scrapes, countRows, getTableStats)

		switch s.ID[0] {
		case "server":
			{
				countServers++
			}
		case "table":
			{
				countTables++
				if !countRows || !getTableStats {
					continue
				}
				res, err := r.DB(s.DB).Table(s.Table).Count().Run(sess)
				if err != nil {
					return err
				}
				var count float64
				if err = res.One(&count); err != nil {
					return err
				}
				scrapes <- s.newScrapeResult("table_docs_total", count)
			}
		case "table_server":
			{
				countReplicas++
				if !getTableStats {
					continue
				}
				s.extractStorageEngineStats(scrapes)
			}
		}
	}

	scrapes <- scrapeResult{Name: "cluster_server_errors_total", Value: float64(countServerErrors)}
	scrapes <- scrapeResult{Name: "cluster_servers_total", Value: float64(countServers)}
	scrapes <- scrapeResult{Name: "cluster_tables_total", Value: float64(countTables)}
	scrapes <- scrapeResult{Name: "cluster_replicas_total", Value: float64(countReplicas)}

	return nil
}

func (e *Exporter) scrape(scrapes chan<- scrapeResult) {

	defer close(scrapes)

	now := time.Now().UnixNano()
	e.totalScrapes.Inc()

	sess, err := r.Connect(r.ConnectOpts{
		Addresses: e.addrs,
		Database:  "rethinkdb",
		AuthKey:   e.auth,
		Username:  e.user,
		Password:  e.pass,
	})

	errCount := 0
	if err == nil {
		if err := extractAllMetrics(sess, scrapes, e.options.countRows, e.options.getTableStats); err != nil {
			errCount++
		}
		scrapes <- scrapeResult{Name: "up", Value: float64(1)}
		sess.Close()
	}

	if err != nil {
		log.Printf("scrape err: %s", err)
		scrapes <- scrapeResult{Name: "up", Value: float64(0)}
	}
	e.scrapeError.Set(float64(errCount))
	e.duration.Set(float64(time.Now().UnixNano()-now) / 1000000000)
}

func (e *Exporter) setMetrics(scrapes <-chan scrapeResult) {

	for s := range scrapes {

		name := s.Name
		value := s.Value
		var labels prometheus.Labels = map[string]string{}

		if len(e.clusterName) > 0 {
			labels["cluster"] = e.clusterName
		}
		if len(s.Server) > 0 {
			labels["server"] = s.Server
		}
		if len(s.DB) > 0 {
			labels["db"] = s.DB
		}
		if len(s.Table) > 0 {
			labels["table"] = s.Table
		}

		if _, ok := e.metrics[name]; !ok {

			asArray := make([]string, 0, len(labels))
			for k := range labels {
				asArray = append(asArray, k)
			}

			e.metrics[name] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: e.options.namespace,
				Name:      name,
			}, asArray)
		}
		e.metrics[name].With(labels).Set(float64(value))
	}
}

func (e *Exporter) collectMetrics(metrics chan<- prometheus.Metric) {
	for _, m := range e.metrics {
		m.Collect(metrics)
	}
}

func main() {
	var (
		addr = flag.String("db.addr", "localhost:28015", "Address of one or more nodes of the cluster, comma separated")
		auth = flag.String("db.auth", "", "Auth key of the RethinkDB cluster")
		user = flag.String("db.user", "", "Auth user for 2.3+ RethinkDB cluster")
		pass = flag.String("db.pass", "", "Auth pass for 2.3+ RethinkDB cluster")

		countRows     = flag.Bool("db.count-rows", true, "Count rows per table, turn off if you experience perf. issues with large tables")
		getTableStats = flag.Bool("table-stats", true, "Get stats for all tables.")

		clusterName   = flag.String("clustername", "", "Cluster Name, added as label to metrics")
		namespace     = flag.String("namespace", "rethinkdb", "Namespace for metrics")
		listenAddress = flag.String("web.listen-address", ":9123", "Address to listen on for web interface and telemetry.")
		metricPath    = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	)

	flag.Parse()

	if len(*addr) == 0 {
		log.Fatal("need parameter addr with len > 0 to connect to RethinkDB cluster")
	}

	options := Options{
		namespace:  *namespace,
		metricPath: *metricPath,

		countRows:     *countRows,
		getTableStats: *getTableStats,
	}

	exporter := NewRethinkDBExporter(*addr, *auth, *user, *pass, *clusterName, options)
	prometheus.MustRegister(exporter)

	log.Printf("listening at %s", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, exporter))
}
