package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Env ...
type Env struct {
	apiKey        string
	maintenanceID int
	pollInterval  int
	metricsPort   string
}

// PingdomMaintenanceSchedules ...
type PingdomMaintenanceSchedules struct {
	Maintenance []MaintenanceSchedule `json:"maintenance"`
}

// PingdomMaintenanceSchedule ...
type PingdomMaintenanceSchedule struct {
	Maintenance MaintenanceSchedule `json:"maintenance"`
}

// MaintenanceSchedule ...
type MaintenanceSchedule struct {
	ID               int    `json:"id"`
	Description      string `json:"description"`
	From             int    `json:"from"`
	To               int    `json:"to"`
	Duration         int    `json:"duration"`
	Durationunit     string `json:"durationunit"`
	Recurrencetype   string `json:"recurrencetype"`
	Repeatevery      int    `json:"repeatevery"`
	Dayofweekinmonth int    `json:"dayofweekinmonth"`
	Effectiveto      int    `json:"effectiveto"`
	Checks           struct {
		Uptime []int `json:"uptime"`
		Tms    []int `json:"tms"`
	} `json:"checks"`
}

// MaintenanceScheduleUpdate ...
type MaintenanceScheduleUpdate struct {
	Description    string `json:"description"`
	From           int    `json:"from"`
	To             int    `json:"to"`
	Recurrencetype string `json:"recurrencetype"`
	Repeatevery    int    `json:"repeatevery"`
	Effectiveto    int    `json:"effectiveto"`
	Uptimeids      string `json:"uptimeids"`
	Tmsids         string `json:"tmsids"`
}

// PingdomChecks ...
type PingdomChecks struct {
	Checks []struct {
		ID                int      `json:"id"`
		Created           int      `json:"created"`
		Name              string   `json:"name"`
		Hostname          string   `json:"hostname"`
		Resolution        int      `json:"resolution"`
		Type              string   `json:"type"`
		Ipv6              bool     `json:"ipv6"`
		VerifyCertificate bool     `json:"verify_certificate"`
		Lasterrortime     int      `json:"lasterrortime"`
		Lasttesttime      int      `json:"lasttesttime"`
		Lastresponsetime  int      `json:"lastresponsetime"`
		Status            string   `json:"status"`
		Maintenanceids    []string `json:"maintenanceids,omitempty"`
	} `json:"checks"`
	Counts struct {
		Total    int `json:"total"`
		Limited  int `json:"limited"`
		Filtered int `json:"filtered"`
	} `json:"counts"`
}

var (
	slaTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ps_pingdom_maintenance_sla_total",
			Help: "Total uptime SLA checks",
		})
	slaMaintenance = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ps_pingdom_maintenance_sla_maintenance",
			Help: "The number of SLA checks in the maintenance schedule",
		})
)

// environment variables
func newEnv(
	apiKey string,
	maintenanceID int,
	pollInterval int,
	metricsPort string) *Env {
	if apiKey == "" {
		log.Fatalf("Could not parse env API_KEY")
	}
	if maintenanceID == 0 {
		log.Fatalf("Could not parse env MAINTENANCE_ID")
	}
	if pollInterval == 0 {
		pollInterval = 300
	}
	if metricsPort == "" {
		metricsPort = "9600"
	}
	e := Env{
		apiKey:        apiKey,
		maintenanceID: maintenanceID,
		pollInterval:  pollInterval,
		metricsPort:   metricsPort,
	}
	log.Printf("\tps-pingdom-maintenance service started...")
	log.Printf("\tMaintenance ID: %d\tPoll Interval: %d\tMetrics port: %s\n\n", e.maintenanceID, e.pollInterval, e.metricsPort)
	return &e
}

// convert env var to integer
func getenvInt(key string) int {
	s := os.Getenv(key)
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

// get a list of pingdom checks tagged sla
func getPingdomChecks(e *Env) (PingdomChecks, error) {
	url := `https://api.pingdom.com/api/3.1/checks?tags=sla`
	var bearer = "Bearer " + e.apiKey
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", bearer)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return PingdomChecks{}, err
	}
	defer resp.Body.Close()
	// Success is indicated with 2xx status codes:
	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		return PingdomChecks{}, errors.New("GET Pingdom checks responded with status code: " + strconv.Itoa(resp.StatusCode))
	}
	body, _ := ioutil.ReadAll(resp.Body)
	var c = PingdomChecks{}
	err = json.Unmarshal(body, &c)
	if err != nil {
		slaTotal.Set(0)
		return PingdomChecks{}, err
	}
	slaTotal.Set(float64(len(c.Checks)))
	return c, nil
}

// Get pingdom maintenance schedule by id
func getPingdomMainenanceSchedule(e *Env) (PingdomMaintenanceSchedule, error) {
	url := fmt.Sprintf(`https://api.pingdom.com/api/3.1/maintenance/%d`, e.maintenanceID)
	var bearer = "Bearer " + e.apiKey
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", bearer)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slaMaintenance.Set(0)
		return PingdomMaintenanceSchedule{}, err
	}
	defer resp.Body.Close()
	// Success is indicated with 2xx status codes:
	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		return PingdomMaintenanceSchedule{}, errors.New("GET Pingdom maintenance responded with status code: " + strconv.Itoa(resp.StatusCode))
	}
	body, _ := ioutil.ReadAll(resp.Body)
	var m = PingdomMaintenanceSchedule{}
	err = json.Unmarshal(body, &m)
	if err != nil {
		slaMaintenance.Set(0)
		return PingdomMaintenanceSchedule{}, err
	}
	slaMaintenance.Set(float64(len(m.Maintenance.Checks.Uptime)))
	return m, nil
}

// convert []int to comma separated string
func intSliceToString(v []int) string {
	valuesText := []string{}
	for i := range v {
		number := v[i]
		text := strconv.Itoa(number)
		valuesText = append(valuesText, text)
	}
	result := strings.Join(valuesText, ",")
	return result
}

// Update pingdom maintenance schedule
func updatePingdomMaintenanceSchedule(e *Env, m PingdomMaintenanceSchedule) error {
	t := time.Now()
	from := time.Date(t.Year(), t.Month(), t.Day(), 15, 0, 0, 0, time.UTC)
	to := time.Date(t.Year(), t.Month(), t.Day()+1, 6, 0, 0, 0, time.UTC)
	schedule := MaintenanceScheduleUpdate{
		Description:    m.Maintenance.Description,
		From:           int(from.Unix()),
		To:             int(to.Unix()),
		Recurrencetype: m.Maintenance.Recurrencetype,
		Repeatevery:    m.Maintenance.Repeatevery,
		Effectiveto:    m.Maintenance.Effectiveto,
		Uptimeids:      intSliceToString(m.Maintenance.Checks.Uptime),
		Tmsids:         intSliceToString(m.Maintenance.Checks.Tms),
	}
	url := fmt.Sprintf(`https://api.pingdom.com/api/3.1/maintenance/%d`, e.maintenanceID)
	var bearer = "Bearer " + e.apiKey
	// marshal MaintenanceScheduleUpdate to json
	json, err := json.Marshal(schedule)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(json))
	req.Header.Add("Authorization", bearer)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Success is indicated with 2xx status codes:
	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		return errors.New("UPDATE Pingdom maintenance responded with status code: " + strconv.Itoa(resp.StatusCode))
	}
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Printf("\tPUT: %s", json)
	log.Printf("\tRESPONSE: %s", response)
	return nil
}

// get a list of pingdom check id's
func getUptimeIds(c PingdomChecks) []int {
	var i []int
	for _, check := range c.Checks {
		i = append(i, check.ID)
	}
	return i
}

// compare two []int
func compareSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func checkMaintenanceSchedule(m PingdomMaintenanceSchedule, u []int) (bool, PingdomMaintenanceSchedule) {
	upToDate := true

	if !compareSlice(m.Maintenance.Checks.Uptime, u) {
		upToDate = false
		m.Maintenance.Checks.Uptime = u
	}
	return upToDate, m
}

func pollAPI(e *Env) {
	ticker := time.NewTicker(time.Second * time.Duration(e.pollInterval)).C
	for {
		select {
		case <-ticker:
			// get uptime checks
			c, err := getPingdomChecks(e)
			if err != nil {
				log.Printf("\tPingdom checks: [ERROR] - %s", err)
				break
			}
			// get uptime check id's
			u := getUptimeIds(c)
			// get maintenance window
			m, err := getPingdomMainenanceSchedule(e)
			if err != nil {
				log.Printf("\tPingdom maintenance: [ERROR] - %s", err)
				break
			}
			// update maintenance schedule if necessary
			upToDate, schedule := checkMaintenanceSchedule(m, u)
			if !upToDate {
				err := updatePingdomMaintenanceSchedule(e, schedule)
				if err != nil {
					log.Printf("\tPingdom update maintenance schedule: [ERROR] - %s", err)
					break
				}
			} else {
				log.Printf("\tMaintenance schedule up to date")
				// get schedule again to update metric
				_, _ = getPingdomMainenanceSchedule(e)
			}
		}
	}
}

func mainloop() {
	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal
	systemTeardown()
}

func systemTeardown() {
	log.Printf("Shutting down...")
}

func main() {
	e := newEnv(
		os.Getenv("API_KEY"),
		getenvInt("MAINTENANCE_ID"),
		getenvInt("POLL_INTERVAL"),
		os.Getenv("METRICS_PORT"),
	)
	go pollAPI(e)
	// prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%s", e.metricsPort), nil)
	mainloop()
}
