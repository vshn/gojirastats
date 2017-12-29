package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func runJqlQuery(config map[string]interface{}, jql string) int {
	host := os.Getenv("JIRA_URL")
	username := os.Getenv("JIRA_USER")
	password := os.Getenv("JIRA_PASSWORD")

	// Create the authenticated  HTTP request
	client := &http.Client{}
	params := url.Values{}
	params.Add("jql", jql)
	// we actually only care about the total atm.
	// specifying one field reduces the amount of useless data in the response
	params.Add("fields", "key")
	req, err := http.NewRequest("GET", host+"/rest/api/latest/search?"+params.Encode(), nil)
	req.SetBasicAuth(username, password)
	resp, err := client.Do(req)
	checkError(err)

	// Read and parse JSON body
	defer resp.Body.Close()
	rawBody, err := ioutil.ReadAll(resp.Body)
	checkError(err)
	var jsonResult interface{}
	err = json.Unmarshal(rawBody, &jsonResult)
	checkError(err)
	m := jsonResult.(map[string]interface{})

	// extract the interesting data
	return int(m["total"].(float64))
}

func createInfluxClient(config map[string]interface{}) client.Client {
	host := os.Getenv("INFLUX_URL")
	username := ""
	password := ""
	if len(os.Getenv("INFLUX_USER")) != 0 {
		username = os.Getenv("INFLUX_USER")
		password = os.Getenv("INFLUX_PASSWORD")
	}
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:               host,
		Username:           username,
		Password:           password,
		InsecureSkipVerify: true,
	})
	checkError(err)
	return c
}

func createBatchPoints(config map[string]interface{}, c client.Client) client.BatchPoints {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  os.Getenv("INFLUX_DB"),
		Precision: "s",
	})
	checkError(err)
	return bp
}

// addPoint adds a point with tags to a BatchPoints object for sending them later
func addPoint(batchPoints client.BatchPoints, rawTags map[string]interface{}, count int) {
	// put tags from config into the right type of map
	tags := map[string]string{}
	for key, value := range rawTags {
		tags[key] = value.(string)
	}
	fields := map[string]interface{}{
		"count": count,
	}

	// for now, the measurement name is fixed
	pt, err := client.NewPoint("issue_count", tags, fields)
	checkError(err)
	fmt.Printf("Prepared for sending: %v: %d issues\n", tags, count)
	batchPoints.AddPoint(pt)
}

func main() {
	// Read queries from config file
	rawConfig, configErr := ioutil.ReadFile(os.Getenv("QUERIES_FILE"))
	checkError(configErr)
	var jsonConfig interface{}
	configErr = json.Unmarshal(rawConfig, &jsonConfig)
	checkError(configErr)
	config := jsonConfig.(map[string]interface{})
	queries := config["queries"].([]interface{})

	// create influx client and batch points (only one send operation at the end)
	influxClient := createInfluxClient(config)
	batchPoints := createBatchPoints(config, influxClient)
	durationBetweenJiraQueries := time.Duration(500) * time.Millisecond

	for _, queryObject := range queries {
		q := queryObject.(map[string]interface{})
		// run jira query
		jql := q["jql"].(string)
		count := runJqlQuery(config, jql)

		// create influx point and save for later
		addPoint(batchPoints, q["tags"].(map[string]interface{}), count)
		time.Sleep(durationBetweenJiraQueries)
	}

	// write the points
	fmt.Println("Writing data to InfluxDB")
	influxErr := influxClient.Write(batchPoints)
	checkError(influxErr)
}
