# uptimed

Uptimed is a simple golang application that can be used to monitor and alert on server reboots.

At a configured frequency is writes the current time to a specified file. When the application starts up it reads that
file and will post a discord notification saying that the server has booted and was last running at the date of the last
heartbeat.

## Usage

```
Usage: ./uptimed [-h|-f|-d <value>]
-h  --heartbeat-file
	Specify the file path to where the uptime heartbeat should be written.
	Defaults to .uptime_heartbeatd.
	Will also take value from environment variable HEARTBEAT_FILE.
-f  --heartbeat-frequency
	Specify the frequency in minutes for how often the heartbeat should be updated.
	Defaults to 10 minutes.
	Will also take value from environment variable HEARTBEAT_FREQUENCY.
-d  --discord-webhook-url
	Optionally specify a discord webhook URL to announce when the application starts.
	Will also take value from environment variable DISCORD_WEBBOOK_URL.
```

## Resource Usage

uptimed has a very small footprint. The vast majority of its time will be idling between heartbeats and consumes only
about a megabyte of memory at any given time.
