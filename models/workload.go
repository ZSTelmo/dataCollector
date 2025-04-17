package models

import (
	"encoding/json"
	"io/ioutil"
)

// Workload represents the configuration loaded from workload.json
type Workload struct {
	Workers       int      `json:"workers"`
	Targets       []string `json:"targets"`
	Output        string   `json:"output"`
	FilterPattern string   `json:"filter_pattern"`
	Query         string   `json:"query"`   // SQL query to execute
	OutputDir     string   `json:"outdir"`  // Optional output directory
	OutputFile    string   `json:"outfile"` // Optional output file name
}

// LoadWorkloadConfig reads and parses the workload configuration file
func LoadWorkloadConfig(filePath string) (*Workload, error) {
	// Read the workload.json file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse the JSON into the Workload struct
	var workload Workload
	if err := json.Unmarshal(data, &workload); err != nil {
		return nil, err
	}

	return &workload, nil
}
