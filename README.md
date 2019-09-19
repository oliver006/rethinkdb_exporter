# RethinkDB Prometheus Metrics Exporter
[![Circle CI](https://circleci.com/gh/oliver006/rethinkdb_exporter.svg?style=shield)](https://circleci.com/gh/oliver006/rethinkdb_exporter) [![Coverage Status](https://coveralls.io/repos/github/oliver006/rethinkdb_exporter/badge.svg?branch=master)](https://coveralls.io/github/oliver006/rethinkdb_exporter?branch=master)

Prometheus exporter for RethinkDB cluster, server and table metrics.<br>
Supports RethinkDB 2.x

## Building and running

Locally build and run:

```
    $ git clone https://github.com/oliver006/rethinkdb_exporter.git
    $ cd rethinkdb_exporter
    $ go build
    $ ./rethinkdb_exporter <flags>
```

Or via docker:

```
    $ docker pull oliver006/rethinkdb_exporter
    $ docker run -d --name rethinkdb_exporter -p 9123:9123 oliver006/rethinkdb_exporter
```


## Deploying

A [Helm](https://helm.sh/) chart is included under `helm/` for installing the rethinkdb_exporter on Kubernetes clusters.  You'll need one of these per RethinkDB cluster you run.

Installing it is as simple as:

```
$ cd helm/rethinkdb-exporter
$ helm install \
    --name=rethinkdb-exporter-for-clustername  \
    --set=rethinkdb_exporter.dbaddr=my-rethinkdb-server:28015  \
    --set=rethinkdb_exporter.dbpass=mypassword  \
    --set=rethinkdb_exporter.clustername=myclustername  \
    .
```


### Flags

Name               | Description
-------------------|------------
db_addr            | Address of one or more nodes of the cluster, comma separated.
db_auth            | Auth key of the RethinkDB cluster (for versions < 2.3)
db_user            | Username for RethinkDB connection (for versions >= 2.3) (must be `admin` if used; see below)
db_pass            | Password for RethinkDB connection (for versions >= 2.3)
db_countrows       | Count rows per table, turn off if you experience perf. issues with large tables
db_tablestats      | Get stats for all tables.
clustername        | Name of the cluster, if set it's added as a label to the metrics.
namespace          | Namespace for the metrics, defaults to "rethinkdb".
web_listenaddress  | Address to listen on for web interface and telemetry, default `:9123`
web_telemetrypath  | Path under which to expose metrics.
config             | Path to config file

Flags can either be passed on the command line 
```
$ rethinkdb_exporter --db_user foo --db_pass bar
```

Or they can be passed as (uppercase) environment variables 
```
$ export DB_USER=foo 
$ export DB_PASS=bar
$ rethinkdb_exporter
```

Or in a config file
```
$ cat > rethinkdb_exporter.conf
db_user foo
db_pass bar

$ rethinkdb_exporter --config rethinkdb_exporter.conf
```


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


### v2.3+ Auth

In v2.3 RethinkDB [moved](https://www.compose.com/articles/using-rethinkdb-2-3s-user-authentication/) to a username/password authentication system.  For compatibility with this use the `--db_user` and `--db_pass` options.

It would be good to use a dedicated read-only user for this but the RethinkDB [docs](https://rethinkdb.com/docs/system-stats/) say "the jobs table can only be accessed by the admin user account".  Thus you'll have to use `--db.user=admin`. 


### What else?

Things that can/should be added
- status metrics per shard 
- ...

Open an issue or PR if you have more suggestions or ideas about what to add.
