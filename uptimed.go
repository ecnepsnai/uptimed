// Command uptimed is a simple golang application that can be used to monitor and alert on server reboots.
//
// At a configured frequency is writes the current time to a specified file. When the application starts up it reads
// that file and will post a discord notification saying that the server has booted and was last running at the date
// of the last heartbeat.
//
//    Usage: ./uptimed [-h|-f|-d <value>]
//    -h  --heartbeat-file
//        Specify the file path to where the uptime heartbeat should be written.
//        Defaults to .uptimed_heartbeat.
//        Will also take value from environment variable HEARTBEAT_FILE.
//    -f  --heartbeat-frequency
//        Specify the frequency in minutes for how often the heartbeat should be updated.
//        Defaults to 10 minutes.
//        Will also take value from environment variable HEARTBEAT_FREQUENCY.
//    -d  --discord-webhook-url
//        Optionally specify a discord webhook URL to announce when the application starts.
//        Will also take value from environment variable DISCORD_WEBHOOK_URL.
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ecnepsnai/discord"
)

func printHelpAndExit() {
	fmt.Printf("Usage: %s [-h|-f|-d <value>]\n", os.Args[0])
	fmt.Printf("-h  --heartbeat-file\n\tSpecify the file path to where the uptime heartbeat should be written.\n\tDefaults to .uptimed_heartbeat\n\tWill also take value from environment variable HEARTBEAT_FILE.\n")
	fmt.Printf("-f  --heartbeat-frequency\n\tSpecify the frequency in minutes for how often the heartbeat should be updated.\n\tDefaults to 10 minutes.\n\tWill also take value from environment variable HEARTBEAT_FREQUENCY.\n")
	fmt.Printf("-d  --discord-webhook-url\n\tOptionally specify a discord webhook URL to announce when the application starts.\n\tWill also take value from environment variable DISCORD_WEBHOOK_URL.\n")
	os.Exit(1)
}

const heartbeatLayout = time.RFC1123

var heartbeatFile = ".uptimed_heartbeat"
var heartbeatFrequencyMinutes = uint16(10)
var didNotifyStartup = uint8(0) // 0 = false, 1 = attempted but failed, 2 = true
var lastHeartbeatBeforeReboot time.Time
var workingDir = ""

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	workingDir = wd

	if value := os.Getenv("HEARTBEAT_FILE"); value != "" {
		heartbeatFile = value
	}
	if value := os.Getenv("HEARTBEAT_FREQUENCY"); value != "" {
		frequency, _ := parseFrequency(value)
		if frequency > 0 {
			heartbeatFrequencyMinutes = frequency
		}
	}
	if value := os.Getenv("DISCORD_WEBHOOK_URL"); value != "" {
		discord.WebhookURL = value
	}

	if len(os.Args) > 1 {
		args := os.Args[1:]
		i := 0
		for i < len(args) {
			arg := args[i]
			if arg == "-h" || arg == "--heartbeat-file" {
				if len(args) == i+1 {
					fmt.Fprintf(os.Stderr, "%s requires value\n\n", arg)
					printHelpAndExit()
				}
				value := args[i+1]
				i++
				heartbeatFile = value
			} else if arg == "-f" || arg == "--heartbeat-frequency" {
				if len(args) == i+1 {
					fmt.Fprintf(os.Stderr, "%s requires value\n\n", arg)
					printHelpAndExit()
				}
				value, err := parseFrequency(args[i+1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s requires numerical value between 1 and 65535\n", arg)
					printHelpAndExit()
				}
				i++
				heartbeatFrequencyMinutes = value
			} else if arg == "-d" || arg == "--discord-webhook-url" {
				if len(args) == i+1 {
					fmt.Fprintf(os.Stderr, "%s requires value\n\n", arg)
					printHelpAndExit()
				}
				value := args[i+1]
				i++
				discord.WebhookURL = value
			} else {
				fmt.Fprintf(os.Stderr, "Unknown option %s\n\n", arg)
				printHelpAndExit()
			}
			i++
		}
	}

	if heartbeatFile == "" {
		fmt.Fprintf(os.Stderr, "Must specify a heartbeat file path\n\n")
		printHelpAndExit()
	}
	if heartbeatFrequencyMinutes == 0 {
		fmt.Fprintf(os.Stderr, "Heartbeat frequency cannot be 0\n\n")
		printHelpAndExit()
	}

	for {
		notifyStartup()
		writeHeartbeat()
		time.Sleep(time.Duration(heartbeatFrequencyMinutes) * time.Minute)
	}
}

func parseFrequency(strValue string) (uint16, error) {
	value, err := strconv.Atoi(strValue)
	if err != nil {
		return 0, err
	}
	if value < 0 || value > 65535 {
		return 0, fmt.Errorf("value out of range for uint16")
	}
	return uint16(value), nil
}

func writeHeartbeat() {
	f, err := os.CreateTemp(workingDir, "uptimed")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening temporary file for writing: %s\n", err.Error())
		return
	}
	if _, err := f.Write([]byte(time.Now().Format(heartbeatLayout))); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing heartbeat file: %s\n", err.Error())
		f.Close()
		os.Remove(f.Name())
		return
	}
	if err := f.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "Error syncing heartbeat file: %s\n", err.Error())
	}
	f.Close()
	if err := os.Rename(f.Name(), heartbeatFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing heartbeat file: %s\n", err.Error())
		return
	}
}

func notifyStartup() {
	if discord.WebhookURL == "" {
		return
	}

	if didNotifyStartup == 2 {
		return
	}

	var lastHeartbeat time.Time
	if didNotifyStartup == 1 {
		lastHeartbeat = lastHeartbeatBeforeReboot
	} else {
		last, err := readLastHeartbeat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting last heartbeat: %s\n", err.Error())
			return
		}
		if last == nil {
			didNotifyStartup = 2
			return
		}
		lastHeartbeat = *last
	}

	message := fmt.Sprintf("System **%s** has booted. Last heartbeat was at **%s**", getHostname(), lastHeartbeat)
	if err := discord.Say(message); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending discord notification: %s\n", err.Error())
		didNotifyStartup = 1
	} else {
		didNotifyStartup = 2
	}
}

func readLastHeartbeat() (*time.Time, error) {
	if !fileExists(heartbeatFile) {
		return nil, nil
	}
	f, err := os.OpenFile(heartbeatFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	heartbeat, err := time.Parse(heartbeatLayout, strings.Trim(string(data), "\n "))
	if err != nil {
		return nil, err
	}

	lastHeartbeatBeforeReboot = heartbeat
	return &heartbeat, nil
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "UNKNOWN_HOSTNAME"
	}
	return name
}

func fileExists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}
