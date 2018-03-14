# RethinkDB Metrics Exporter
[![Circle CI](https://circleci.com/gh/oliver006/rethinkdb_exporter.svg?style=shield)](https://circleci.com/gh/oliver006/rethinkdb_exporter) [![Coverage Status](https://coveralls.io/repos/github/oliver006/rethinkdb_exporter/badge.svg?branch=master)](https://coveralls.io/github/oliver006/rethinkdb_exporter?branch=master)

Prometheus exporter for RethinkDB cluster, server and table metrics.<br>
Supports RethinkDB 2.x and 1.6.x (and possibly older versions)

## Building and running

    go build
    ./rethinkdb_exporter <flags>

### Flags

Name               | Description
-------------------|------------
db.addr            | Address of one or more nodes of the cluster, comma separated.
db.auth            | Auth key of the RethinkDB cluster.
db.count-rows      | Count rows per table, turn off if you experience perf. issues with large tables
db.table-stats     | Get stats for all tables.
clustername        | Name of the cluster, if set it's added as a label to the metrics.
namespace          | Namespace for the metrics, defaults to "rethinkdb".
web.listen-address | Address to listen on for web interface and telemetry.
web.telemetry-path | Path under which to expose metrics.


### What's exported?

All entries from the `stats` table of the internal database `rethinkdb` are exported,
see http://rethinkdb.com/docs/system-stats/ for details.<br>
In addition, for every table there is a gauge with the number of items of said table.<br> 
Metric name is `rethinkdb_table_items_total{db="...",table="..."}`
There are also total counters for numer of servers, tables and replicas as well as number of 
errors returned from the `stats` table.<br>
Metric names are `rethinkdb_cluster_[servers|server_errors|tables|replicas]_total`


### What does it look like?
[Grafana](https://github.com/grafana) dashboard is available [here](https://grafana.com/dashboards/5043):<br>
![rethink_exporter_dashboard](https://grafana.com/api/dashboards/5043/images/3108/image)



### What else?

Things that can/should be added
- status metrics per shard 
- ...

Open an issue or PR if you have more suggestions or ideas about what to add.
