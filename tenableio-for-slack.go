package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

const baseTenablePath = "https://cloud.tenable.com"

func main() {
	// Command line arguments for YAML config
	var configPath string
	flag.StringVar(&configPath, "config", "config.yml", "Full Path to the YAML configuration file")
	flag.Parse()

	// Read in file
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Configuration file does not exists: %v\n", err)
		os.Exit(0)
	}

	// Read in YAML config
	yamlData, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Unable to read YAML configuration file: %v\n", err)
		os.Exit(0)
	}
	var config Config

	yamlErr := yaml.Unmarshal(yamlData, &config)
	if yamlErr != nil {
		fmt.Printf("Unable to unmarshal YAML config: %v\n", yamlErr)
	}
	// Create Tenable.io Client with configured keys
	TenableClient := NewTenableIOClient(config.APIAccessKey, config.APISecretKey)

	// Build the audit log date filter
	layout := "2006-01-02"
	//dateFilter, err := time.Parse(layout, dateTime.String())
	dateFilter := time.Now().Format(layout)

	// Build GET request
	req := TenableClient.NewRequest("GET", "audit-log/v1/events", nil)
	GetParams := req.URL.Query()
	GetParams.Add("f", fmt.Sprintf("date.gt:%v", dateFilter))
	fmt.Printf("Requesting Logs with filter: %v\n", dateFilter)
	req.URL.RawQuery = GetParams.Encode()

	// Request Logs
	LogResponse := TenableClient.Do(req)
	ResponseBytes, _ := ioutil.ReadAll(LogResponse.Body)
	var Logs AuditLogResponse

	// Unmarshal API response to AuditLogResponse struct
	responseError := json.Unmarshal(ResponseBytes, &Logs)
	if responseError != nil {
		fmt.Printf("Unable to unmarshal Audit Log Response: %v\n", responseError)
		os.Exit(0)
	}

	// Compare Responses with local file of today's events
	const cacheFileName = "cache.log"

	if _, err := os.Stat(cacheFileName); os.IsNotExist(err) {
		file, _ := os.Create(cacheFileName)

	} else {
		file, _ := os.Open(cacheFileName)
		cachedEvents, _ := readLines(cacheFileName)
	}

	// Create a Slack Client for sending messages
	slackClient := NewSlackClient(config.AuditLogConfig.SlackWebHook)

	// Iterate over the log response and create slack message for each log type that's configured
	fmt.Printf("%v Events returned\n", len(Logs.Events))
	for _, event := range Logs.Events {
		// Check to see if the log event action matches one of the types in the config
		if stringInSlice(event.Action, config.AuditLogConfig.EnabledEventTypes) {
			// Send Slack Message
			slackClient.SendMessage(&SlackMessage{text: BuildSlackText(event)})
			os.Exit(0)
		}
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func BuildSlackText(event Event) string {
	var ResponseText string
	switch event.Action {
	case "user.impersonation.start":
		ResponseText = fmt.Sprintf("User \"%v\" has started impersonating user \"%v\"", event.Actor.Name, event.Target.Name)
	case "user.impersonation.end":
		ResponseText = fmt.Sprintf("User \"%v\" has finished impersonating user \"%v\"", event.Actor.Name, event.Target.Name)
	case "user.create":
		if event.Target.Name != "" {
			ResponseText = fmt.Sprintf("User \"%v\" has created user \"%v\"", event.Actor.Name, event.Target.Name)
		} else {
			ResponseText = fmt.Sprintf("User \"%v\" has created a user", event.Actor.Name)
		}

	case "user.delete":
		ResponseText = fmt.Sprintf("User \"%v\" has deleted user \"%v\"", event.Actor.Name, event.Target.Name)
	case "user.update":
		if event.Actor.Name == event.Target.Name {
			ResponseText = fmt.Sprintf("User \"%v\" has updated their account", event.Actor.Name)
		} else {
			ResponseText = fmt.Sprintf("User \"%v\" has updated user \"%v\"", event.Actor.Name, event.Target.Name)
		}
	}
	return ResponseText
}

type SlackClient struct {
	client     *http.Client
	WebHookURL string
}

type SlackMessage struct {
	text string
}

func (slack *SlackClient) SendMessage(msg *SlackMessage) http.Response {
	bodyMap := map[string]interface{}{
		"text": msg.text,
	}
	body, _ := json.Marshal(bodyMap)
	req, err := http.NewRequest("POST", slack.WebHookURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Printf("Unable to build Slack request: %v\n", err)
	}
	resp, err := slack.client.Do(req)
	if err != nil {
		fmt.Printf("Unable to send Slack request: %v\n", err)
	}
	return *resp

}

func NewSlackClient(WebHookURL string) SlackClient {
	client := &http.Client{}
	slackClient := &SlackClient{
		client,
		WebHookURL,
	}

	return *slackClient
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

type Config struct {
	APIAccessKey   string `yaml:"api_access_key"`
	APISecretKey   string `yaml:"api_secret_key"`
	AuditLogConfig struct {
		SlackWebHook      string   `yaml:"slack_webhook_url"`
		EnabledEventTypes []string `yaml:"enabled_event_types"`
	} `yaml:"audit_logs"`
}

type Event struct {
	ID          string      `json:"id"`
	Action      string      `json:"action"`
	Crud        string      `json:"crud"`
	IsFailure   bool        `json:"is_failure"`
	Received    time.Time   `json:"received"`
	Description interface{} `json:"description"`
	Actor       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"actor"`
	IsAnonymous interface{} `json:"is_anonymous"`
	Target      struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"target"`
	Fields []interface{} `json:"fields"`
}

type AuditLogResponse struct {
	Events     []Event `json:"events"`
	Pagination struct {
		Total int `json:"total"`
		Limit int `json:"limit"`
	} `json:"pagination"`
}

type TenableIOClient struct {
	client    *http.Client
	basePath  string
	accessKey string
	secretKey string
}

func NewTenableIOClient(accessKey string, secretKey string) TenableIOClient {
	client := &http.Client{}
	tio := &TenableIOClient{
		client,
		baseTenablePath,
		accessKey,
		secretKey,
	}
	return *tio
}

func (tio *TenableIOClient) Do(req *http.Request) http.Response {
	req.Header.Set("X-ApiKeys", fmt.Sprintf("accessKey=%v; secretKey=%v;", tio.accessKey, tio.secretKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoTenable")
	fmt.Printf("Requesting \"%v\": %v\n", req.URL, req.Body)
	resp, err := tio.client.Do(req)
	if err != nil {
		fmt.Printf("Unable to run request: %v\n", err)
	}

	return *resp
}

func (tio *TenableIOClient) NewRequest(method string, endpoint string, body []byte) *http.Request {
	fullUrl := fmt.Sprintf("%v/%v", baseTenablePath, endpoint)
	req, err := http.NewRequest(method, fullUrl, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Unable to build request [%v] %v request\n", method, endpoint)
	}
	return req
}
