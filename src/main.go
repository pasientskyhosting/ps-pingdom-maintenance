package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Env ...
type Env struct {
	apiKey        string
	maintenanceID int
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

// environment variables
func newEnv(
	apiKey string,
	maintenanceID int) *Env {
	if apiKey == "" {
		log.Fatalf("Could not parse env API_KEY")
	}
	if maintenanceID == 0 {
		log.Fatalf("Could not parse env MAINTENANCE_ID")
	}
	e := Env{
		apiKey:        apiKey,
		maintenanceID: maintenanceID,
	}
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
		log.Fatal("Pingdom checks:\n[ERROR] - ", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var c = PingdomChecks{}
	err = json.Unmarshal(body, &c)
	if err != nil {
		log.Fatal("Pingdom checks:\n[ERROR] - ", err)
	}
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
		log.Fatal("Pingdom maintenance:\n[ERROR] - ", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var m = PingdomMaintenanceSchedule{}
	err = json.Unmarshal(body, &m)
	if err != nil {
		log.Fatal("Pingdom maintenance:\n[ERROR] - ", err)
	}
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
func updatePingdomMaintenanceSchedule(e *Env, m PingdomMaintenanceSchedule) {
	schedule := MaintenanceScheduleUpdate{
		Description:    m.Maintenance.Description,
		From:           m.Maintenance.From,
		To:             m.Maintenance.To,
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
		log.Fatal("Pingdom update maintenance schedule:\n[ERROR] - ", err)
	}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(json))
	req.Header.Add("Authorization", bearer)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Pingdom update maintenance schedule:\n[ERROR] - ", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatal("Pingdom update maintenance schedule:\n[ERROR] - ", fmt.Sprintf("Pingdom responded with status code %d", resp.StatusCode))
	}
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Pingdom update maintenance schedule:\n[ERROR] - ", err)
	}
	fmt.Println(string(response))
}

// get a list of pingdom check id's
func getUptimeIds(c PingdomChecks) []int {
	var i []int
	for _, check := range c.Checks {
		i = append(i, check.ID)
	}
	return i
}

// check if a []int contains a value
func contains(intSlice []int, searchInt int) bool {
	for _, value := range intSlice {
		if value == searchInt {
			return true
		}
	}
	return false
}

func checkMaintenanceSchedule(m PingdomMaintenanceSchedule, u []int) (bool, PingdomMaintenanceSchedule) {
	upToDate := true
	for _, check := range u {
		if !contains(m.Maintenance.Checks.Uptime, check) {
			upToDate = false
			m.Maintenance.Checks.Uptime = append(m.Maintenance.Checks.Uptime, check)
		}
	}
	return upToDate, m
}

func main() {
	e := newEnv(
		os.Getenv("API_KEY"),
		getenvInt("MAINTENANCE_ID"),
	)

	// get uptime checks
	c, err := getPingdomChecks(e)
	if err != nil {
		log.Fatal(err)
	}

	// get uptime check id's
	u := getUptimeIds(c)

	// get maintenance window
	m, err := getPingdomMainenanceSchedule(e)
	if err != nil {
		log.Fatal(err)
	}
	// update maintenance schedule if necessary
	upToDate, schedule := checkMaintenanceSchedule(m, u)
	if !upToDate {
		updatePingdomMaintenanceSchedule(e, schedule)
	} else {
		fmt.Println("Nothing to do... Exiting.")
	}
}
