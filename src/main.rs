use chrono::prelude::*;
use reqwest::blocking::Client as HTTPClient;
use std::collections::HashMap;
use std::fs::File;
use std::io::prelude::*;
use std::path::Path;
use std::{process, thread, time};

/// Upon startup, a notification was not posted
const NOTIFY_STARTUP_FALSE: u8 = 0;
/// Upon startup, an unsuccessful attempt to post a notification was made
const NOTIFY_STARTUP_FAILED: u8 = 0;
/// Upon startup, a notification was posted
const NOTIFY_STARTUP_NOTIFIED: u8 = 0;

/// Describes options for application runtime
struct Options<'a> {
    /// The Discord webhook URL
    webhook_url: &'a str,
    /// The file path to the heartbeat file
    heartbeat_file_path: &'a str,
    /// The frequency in minutes to write a heartbeat
    heartbeat_frequency: u16,
    /// The state of if the application did post a notification on startup
    did_notify_startup: u8,
    /// The last heartbeat timestamp before reboot
    last_heartbeat_before_reboot: i64,
}

fn main() {
    let mut options = Options {
        webhook_url: "",
        heartbeat_file_path: ".uptime_heartbeat",
        heartbeat_frequency: 10,
        did_notify_startup: NOTIFY_STARTUP_FALSE,
        last_heartbeat_before_reboot: 0,
    };

    let args: Vec<String> = std::env::args().collect();
    let mut skip_next_arg = false;
    for i in 1..args.len() {
        if skip_next_arg {
            skip_next_arg = false;
            continue;
        }
        let arg = &args[i];

        if arg == "-h" || arg == "--heartbeat-file" {
            if args.len() == i + 1 {
                eprintln!("Argument {} requires a value", arg);
                print_help_and_exit();
            }
            let value = &args[i + 1];
            options.heartbeat_file_path = value;
            skip_next_arg = true;
        } else if arg == "-f" || arg == "--heartbeat-frequency" {
            if args.len() == i + 1 {
                eprintln!("Argument {} requires a value", arg);
                print_help_and_exit();
            }
            let value = args[i + 1]
                .parse::<u16>()
                .expect("invalid value for heartbeat frequency");
            options.heartbeat_frequency = value;
            skip_next_arg = true;
        } else if arg == "-d" || arg == "--discord-webhook-url" {
            if args.len() == i + 1 {
                eprintln!("Argument {} requires a value", arg);
                print_help_and_exit();
            }
            let value = &args[i + 1];
            options.webhook_url = value;
            skip_next_arg = true;
        } else {
            eprintln!("Unknown argument {}", arg);
            print_help_and_exit();
        }
    }

    loop {
        options.did_notify_startup = check_heartbeat(&options);
        write_heartbeat_file(options.heartbeat_file_path);
        thread::sleep(time::Duration::from_secs(
            (options.heartbeat_frequency * 60).into(),
        ));
    }
}

/// Print help text to the terminal and exit
fn print_help_and_exit() {
    print!(
        "Usage: {} [-h|-f|-d <value>]\n",
        std::env::args().next().unwrap()
    );
    print!("-h  --heartbeat-file\n\tSpecify the file path to where the uptime heartbeat should be written.\n\tDefaults to .uptime_heartbeat\n");
    print!("-f  --heartbeat-frequency\n\tSpecify the frequency in minutes for how often the heartbeat should be updated.\n\tDefaults to 10 minutes.\n");
    print!("-d  --discord-webhook-url\n\tOptionally specify a discord webhook URL to announce when the application starts.\n");
    process::exit(1);
}

/// Write a heartbeat file
///
/// # Arguments
/// * `heartbeat_file_path` - The file path to the heartbeat file
/// # Panics
/// Will panic if it could not create or write to the file
fn write_heartbeat_file(heartbeat_file_path: &str) {
    let path = Path::new(heartbeat_file_path);
    let mut file = File::create(&path).expect("create failed");
    let ts: String = Local::now().timestamp().to_string();
    file.write_all(ts.as_bytes()).expect("write failed");
}

/// Read the last heartbeat file and get a formatted date string from it
/// # Arguments
/// * `heartbeat_file_path` - The file path to the heartbeat file
/// # Returns
/// Will return a string containing a formatted date string of the heartbeat from the file, or None.
fn read_last_heartbeat_file(heartbeat_file_path: &str) -> Option<String> {
    let path = Path::new(heartbeat_file_path);
    if !path.exists() {
        return None;
    }

    let mut file = match File::open(&path) {
        Err(_why) => return None,
        Ok(file) => file,
    };
    let mut ts_str = String::new();

    match file.read_to_string(&mut ts_str) {
        Err(_why) => return None,
        Ok(_) => {}
    };

    let timestamp = match ts_str.parse::<i64>() {
        Err(_e) => return None,
        Ok(t) => t,
    };

    return Some(
        Local
            .timestamp(timestamp, 0)
            .format("%Y-%m-%d %H:%M")
            .to_string(),
    );
}

/// Check the heartbeat file and post a notification if needed
/// # Arguments
/// * `options` - Runtime options
/// # Returns
/// Returns a new value for the did_notify_startup
fn check_heartbeat(options: &Options) -> u8 {
    if options.webhook_url == "" {
        return NOTIFY_STARTUP_NOTIFIED;
    }

    if options.did_notify_startup == 2 {
        return NOTIFY_STARTUP_NOTIFIED;
    }

    let last_heartbeat_str: String;
    if options.did_notify_startup == 1 {
        last_heartbeat_str = Local
            .timestamp(options.last_heartbeat_before_reboot, 0)
            .format("%Y-%m-%d %H:%M")
            .to_string();
    } else {
        let result = read_last_heartbeat_file(options.heartbeat_file_path);
        if result.is_none() {
            return NOTIFY_STARTUP_NOTIFIED;
        }
        last_heartbeat_str = result.unwrap();
    }

    let message = format!(
        "System **{}** has booted. Last heartbeat was at **{}**",
        get_hostname(),
        last_heartbeat_str
    );
    println!("{}", message);
    return match discord_say(message, &options.webhook_url) {
        Ok(_v) => NOTIFY_STARTUP_NOTIFIED,
        Err(_e) => NOTIFY_STARTUP_FAILED,
    };
}

/// Get the systems hostname
/// # Returns
/// A string of the system hostname
/// # Panics
/// Will panic if there was a problem getting the hostname
fn get_hostname() -> String {
    return hostname::get().unwrap().into_string().unwrap();
}

/// Post a text message to a discord webhook URL
/// # Arguments
/// * `message` - The message text. May include Discord text formatting.
/// * `webhook_url` - The webhook URL.
fn discord_say(message: String, webhook_url: &str) -> Result<(), String> {
    let mut body = HashMap::new();
    body.insert("content", message);
    let req = HTTPClient::new()
        .post(webhook_url)
        .header("Content-Type", "application/JSON")
        .json(&body);
    return match req.send() {
        Ok(_v) => Ok(()),
        Err(e) => Err(e.to_string()),
    };
}
