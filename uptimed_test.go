package main

import (
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

func TestWriteHeartbeat(t *testing.T) {
	heartbeatFile = path.Join(t.TempDir(), ".uptimed_heartbeat")
	writeHeartbeat()
	f, err := os.OpenFile(heartbeatFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		t.Fatalf("Error opening heartbeat file: %s", err.Error())
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("Error reading heartbeat file: %s", err.Error())
	}

	heartbeat, err := time.Parse(heartbeatLayout, strings.Trim(string(data), "\n "))
	if err != nil {
		t.Fatalf("Error parsing heartbeat file: %s", err.Error())
	}

	if time.Since(heartbeat).Minutes() >= 1.0 {
		t.Errorf("Invalid time in heartbeat file: %s", data)
	}
}

func TestParseFrequency(t *testing.T) {
	var frequency uint16
	var err error

	frequency, err = parseFrequency("5")
	if err != nil {
		t.Errorf("Unexpected error parsing valid frequency amount: %s", err.Error())
	}
	if frequency != 5 {
		t.Errorf("Unexpected frequency value: %d", frequency)
	}

	frequency, err = parseFrequency("-1")
	if err == nil {
		t.Errorf("No error seen when one expected for parsing invalid frequency")
	}
	if frequency != 0 {
		t.Errorf("Unexpected frequency value: %d", frequency)
	}

	frequency, err = parseFrequency("65537")
	if err == nil {
		t.Errorf("No error seen when one expected for parsing invalid frequency")
	}
	if frequency != 0 {
		t.Errorf("Unexpected frequency value: %d", frequency)
	}

	frequency, err = parseFrequency("apples")
	if err == nil {
		t.Errorf("No error seen when one expected for parsing invalid frequency")
	}
	if frequency != 0 {
		t.Errorf("Unexpected frequency value: %d", frequency)
	}
}
