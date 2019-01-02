package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/thathaneydude/go-tenable"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

const cacheFileName = "cache.log"
const logFileName = "TenableIOForSlack.log"

func main() {
	// Command line arguments for YAML config
	var configPath string
	flag.StringVar(&configPath, "config", "config.yml", "Full Path to the YAML configuration file")
	flag.Parse()

	// Check to see if configuration file in the command line exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Error(fmt.Sprintf("Configuration file does not exists: %v\n", err))
		os.Exit(0)
	}

	// Read in YAML config
	yamlData, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Error(fmt.Sprintf("Unable to read YAML configuration file: %v\n", err))
		os.Exit(0)
	}
	var config Config

	yamlErr := yaml.Unmarshal(yamlData, &config)
	if yamlErr != nil {
		log.Error(fmt.Sprintf("Unable to unmarshal YAML config: %v\n", yamlErr))
	}

	ConfigureLogging(config.LogPath, logFileName)

	// Create Tenable.io Client with configured keys
	TenableClient := go_tenable.NewTenableIOClient(config.APIAccessKey, config.APISecretKey)

	// Build the audit log date filter
	layout := "2006-01-02"
	//dateFilter, err := time.Parse(layout, dateTime.String())
	dateFilter := time.Now().Format(layout)

	// Build GET request
	req := TenableClient.NewRequest("GET", "audit-log/v1/events", nil)
	GetParams := req.URL.Query()
	GetParams.Add("f", fmt.Sprintf("date.gt:%v", dateFilter))
	req.URL.RawQuery = GetParams.Encode()

	// Request Logs
	LogResponse := TenableClient.Do(req)
	ResponseBytes, _ := ioutil.ReadAll(LogResponse.Body)
	var Logs go_tenable.AuditLogResponse

	// Unmarshal API response to AuditLogResponse struct
	responseError := json.Unmarshal(ResponseBytes, &Logs)
	if responseError != nil {
		log.Error(fmt.Sprintf("Unable to unmarshal Audit Log Response: %v\n", responseError))
		os.Exit(0)
	}

	// Compare Responses with local file of today's events
	var cachedEvents []string
	if fileStat, err := os.Stat(cacheFileName); os.IsNotExist(err) || time.Now().Sub(fileStat.ModTime()) > 24*time.Hour {
		log.Debug("Cache is being rebuilt for today\n")
		f, err := os.Create(filepath.Join(config.LogPath, cacheFileName))
		if err != nil {
			log.Error(fmt.Sprintf("Unable to create cache file: %v\n", err))
		}
		err = f.Close()
	} else {
		cachedEvents, err = readLines(cacheFileName)
		if err != nil {
			log.Error("Unable to read cached audit messages: %v\n", err)
			os.Exit(0)
		}
	}

	// Create a Slack Client for sending messages
	slackClient := NewSlackClient(config.AuditLogConfig.SlackWebHook)

	// Iterate over the log response and create slack message for each log type that's configured
	var linesToWrite []string

	log.Info(fmt.Sprintf("%v Events returned\n", len(Logs.Events)))
	for _, event := range Logs.Events {
		// Check to see if the log event action matches one of the types in the config
		if !stringInSlice(event.Action, config.AuditLogConfig.EnabledEventTypes) {
			log.Debug(fmt.Sprintf("Event \"%v\" is not configured\n", event.Action))
			continue
		}

		// Check to see if the event has already been sent
		if stringInSlice(event.ID, cachedEvents) {
			log.Debug(fmt.Sprintf("Event \"%v\" has already been sent to Slack\n", event.ID))
			continue
		}

		// Send Slack Message
		log.Debug(fmt.Sprintf("Event to Send: %v\n", event))
		slackMessage := SlackMessage{text: BuildSlackText(event)}
		log.Debug(fmt.Sprintf("Corresponding Slack Message: %v\n", slackMessage))
		slackClient.SendMessage(&slackMessage)
		linesToWrite = append(linesToWrite, event.ID)
	}

	if len(linesToWrite) > 0 {
		writeError := writeLines(linesToWrite, cacheFileName)
		if writeError != nil {
			log.Error(fmt.Sprintf("Unable to write %v events to today's cache: %v\n", len(linesToWrite), writeError))
		}
	} else {
		log.Info("No events to cache\n")
	}
}

func ConfigureLogging(LogPath string, FileName string) {
	f, err := os.OpenFile(filepath.Join(LogPath, FileName), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(f)
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

func writeLines(lines []string, path string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
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
	LogPath        string `yaml:"log_path"`
	AuditLogConfig struct {
		SlackWebHook      string   `yaml:"slack_webhook_url"`
		EnabledEventTypes []string `yaml:"enabled_event_types"`
	} `yaml:"audit_logs"`
}
