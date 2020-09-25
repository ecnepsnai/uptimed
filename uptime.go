// Command uptime is a simple golang application that can be used to monitor and alert on server reboots.
//
// At a configured freuqney is writes the current time to a specified file. When the application starts up it reads
// that file and will post a discord notification saying that the server has booted and was last running at the date
// of the last heartbeat.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ecnepsnai/discord"
)

func printHelpAndExit() {
	fmt.Printf("Usage: %s [-h|-f|-d <value>]\n", os.Args[0])
	fmt.Printf("-h  --heartbeat-file\n\tSpecify the file path to where the uptime heartbeat should be written.\n\tDefaults to .uptime_heartbeat\n")
	fmt.Printf("-f  --heartbeat-frequency\n\tSpecify the frequency in minutes for how often the heartbeat should be updated.\n\tDefaults to 10 minutes.\n")
	fmt.Printf("-d  --discord-webhook-url\n\tOptionally specify a discord webhook URL to announce when the application starts.\n")
	os.Exit(1)
}

var heartbeatFile = ".uptime_heartbeat"
var heartbeatFrequencyMinutes = uint16(10)
var didNotifyStartup = uint8(0) // 0 = false, 1 = attempted but failed, 2 = true
var lastHeartbeatBeforeReboot int64

func main() {
	if len(os.Args) == 1 {
		printHelpAndExit()
	}

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
			value, err := strconv.Atoi(args[i+1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s requires numerical value\n", arg)
				printHelpAndExit()
			}
			if value < 0 || value > 65535 {
				fmt.Fprintf(os.Stderr, "%s requires numerical value between 1 and 65535\n", arg)
				printHelpAndExit()
			}
			i++
			heartbeatFrequencyMinutes = uint16(value)
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

	if heartbeatFile == "" {
		fmt.Fprintf(os.Stderr, "Must specify a heartbeat file path\n\n")
		printHelpAndExit()
	}
	if heartbeatFrequencyMinutes == 0 {
		fmt.Fprintf(os.Stderr, "Heartbeat freqnecy cannot be 0\n\n")
		printHelpAndExit()
	}

	for true {
		notifyStartup()
		writeHeartbeat()
		time.Sleep(time.Duration(heartbeatFrequencyMinutes) * time.Minute)
	}
}

func writeHeartbeat() {
	f, err := ioutil.TempFile("", "uptime")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening temporary file for writing: %s\n", err.Error())
		return
	}
	if _, err := f.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing heartbeat file: %s\n", err.Error())
		f.Close()
		os.Remove(f.Name())
		return
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

	var lastHeartbeat string
	if didNotifyStartup == 1 {
		lastHeartbeat = time.Unix(0, lastHeartbeatBeforeReboot).Format("2006-01-02 15:04")
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

func readLastHeartbeat() (*string, error) {
	if !fileExists(heartbeatFile) {
		return nil, nil
	}
	f, err := os.OpenFile(heartbeatFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	nano, err := strconv.ParseInt(strings.Trim(string(data), "\n "), 10, 64)
	if err != nil {
		return nil, err
	}

	lastHeartbeatBeforeReboot = nano
	lastHeartbeat := time.Unix(0, nano).Format("2006-01-02 15:04")
	return &lastHeartbeat, nil
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
