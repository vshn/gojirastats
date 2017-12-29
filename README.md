# gojirastatus - JIRA metrics collector

Read statistics from JIRA and write the result to InfluxDB.

## Configuration

### Environment Variables

* `QUERIES_FILE`: Path to query configuration
* `INFLUX_URL`: URL of InfluxDB
* `INFLUX_USER`: InfluxDB Username
* `INFLUX_PASSWORD`: InfluxDB Password
* `JIRA_URL`: JIRA URL
* `JIRA_USER`: JIRA Username
* `JIRA_PASSWORD`: JIRA Password

### Queries

See `queries.example.json` for an example configuration.

## Credit

https://github.com/cbonitz/jira-influx