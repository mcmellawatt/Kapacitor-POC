package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/influxdata/kapacitor/client/v1"
)

var (
	monotonicityAlertFired bool
	nonnegativeAlertFired  bool
)

func main() {
	go func() {
		http.HandleFunc("/kapacitor-alert", kapacitorAlertHandler)
		log.Fatal(http.ListenAndServe("0.0.0.0:9001", nil))
	}()

	fmt.Println("Simulated time-series data:")

	now := time.Now()
	writeStream(now, streamSpec{
		name:      "test",
		tag:       "valid",
		valueFunc: func(i int) float64 { return float64(i + 1) },
	})
	writeStream(now, streamSpec{
		name:      "test",
		tag:       "monotonicity_violation",
		valueFunc: func(i int) float64 { return float64(10 - i) },
	})
	writeStream(now, streamSpec{
		name:      "test",
		tag:       "nonnegative_violation",
		valueFunc: func(i int) float64 { return float64(i - 5) },
	})

	cli, err := client.New(client.Config{URL: "http://localhost:9092"})
	if err != nil {
		log.Fatalf("Kapacitor client error: %v", err)
	}
	if cli == nil {
		log.Fatalf("Kapacitor client is nil after creation. Check configuration and Kapacitor status.")
	}

	tickScriptBytes, err := os.ReadFile("invariants.tick")
	if err != nil {
		log.Fatalf("Failed to read invariants.tick: %v", err)
	}
	tickScript := string(tickScriptBytes)

	taskOpts := client.CreateTaskOptions{
		ID:         "invariant-check",
		Type:       client.BatchTask,
		TICKscript: tickScript,
		Status:     client.Enabled,
		DBRPs: []client.DBRP{
			{Database: "testdb", RetentionPolicy: "autogen"},
		},
	}
	_, err = cli.CreateTask(taskOpts)
	if err != nil {
		log.Printf("Could not create invariant-check batch task: %v", err)
	} else {
		fmt.Println("Invariant-check Kapacitor batch task created.")
	}

	fmt.Println("\nWaiting for alerts.")
	select {}
}

func writeToInflux(points []map[string]interface{}, measurement, streamTag string) error {
	influxURL := "http://localhost:8086/write?db=testdb"
	for _, point := range points {
		timestamp := point["time"].(time.Time).UnixNano()
		value := point["value"].(float64)
		line := fmt.Sprintf("%s,stream=%s value=%f %d", measurement, streamTag, value, timestamp)
		resp, err := http.Post(influxURL, "text/plain", strings.NewReader(line))
		if err != nil {
			return fmt.Errorf("failed to write to InfluxDB: %w", err)
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("influxdb returned status %d", resp.StatusCode)
		}
	}
	return nil
}

func kapacitorAlertHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Println("Raw alert body:", string(body))

	var alertPayload map[string]interface{}
	if err := json.Unmarshal(body, &alertPayload); err == nil {
		log.Println("Parsed JSON alert payload:")
		for k, v := range alertPayload {
			log.Printf("%s: %v", k, v)
		}

		if msg, ok := alertPayload["details"].(string); ok && strings.Contains(msg, "monotonicity violation") {
			monotonicityAlertFired = true
		}
		if msg, ok := alertPayload["details"].(string); ok && strings.Contains(msg, "non-negative violation") {
			nonnegativeAlertFired = true
		}
	}

	if monotonicityAlertFired && nonnegativeAlertFired {
		log.Println("Both invariant alerts have fired. Exiting.")
		os.Exit(0)
	}
	w.WriteHeader(http.StatusOK)
}

type streamSpec struct {
	name      string
	tag       string
	valueFunc func(i int) float64
}

func writeStream(now time.Time, spec streamSpec) {
	stream := make([]map[string]interface{}, 0)
	for i := range 10 {
		stream = append(stream, map[string]any{
			"time":  now.Add(time.Duration(i-9) * time.Minute),
			"value": spec.valueFunc(i),
		})
	}
	fmt.Printf("%s (%s): %v\n", spec.name, spec.tag, stream)
	writeToInflux(stream, spec.name, spec.tag)
}
