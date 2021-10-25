# WatchDog 🤪

Monitor servers through plug in scripts.

Three components:

[**Client**](cmd/client)s running on multiple servers(that are to be monitored). Requires a configuration file with format mentioned below and runs the tasks mentioned in the conf file. On error, it sends message to **server**.

```
Usage of ./client:
  -c string
        path to cfg file (default "config.json")
  -l string
        path to log file (default "log/log.txt")
  -r string
        server address in the format IP:PORT (default "localhost:40090")
```
The [**Server**](cmd/server) is a gRPC server listening on port 40090. Multiple **client**s can connect to it and share errors/alerts. It then broadcasts the same to **wdc**s. Server also runs a http server for ws connections on port 40080.

```
Usage of ./server:
  -l string
        path to log file (default "log/log.txt")
```

[**wdc**](cmd/wdc)s are clients that run on monitoring spoc's local machines. It connects to the **server** through websockets and listens to incoming alerts.

```
Usage of ./wdc:
  -ep string
        http service address (default "/ws/connect")
  -r string
        http service address (default "localhost:40080")
```

### Configuration file format
type: JSON

```json
{
    "hostname": "h0stnam3",
      // (optional)
      // If not specified, the client will try to get system hostname.
      // Else, `hostname` will be used.
    "tasks": [
        {
            "name": "foo",
                  // (optional)
                  // Note: `name` can be helpful to distinguish tasks while reading log files,
                  // so, it's recommended to give one (ideally separated with dashes).
            "repeatInterval": 60,
                  // (required)
                  // the time interval after which the task should repeat itself
            "cmd": "/path/to/some/script/to/execute.sh",
                  // (required)
                  // The command/script to execute.
                  // Note: If the script failed with exit code != 0, it will trigger an alert.
            "msg": "some message that is to be sent to monitoring spoc when cmd fails. Failure is identified by printing something to stderr or completion with non-zero exit status",
                  // (required)
                  // the message that will sent upon failure of script mentioned in `cmd`
            "actionsToBeTaken": [
                  // (optional)
                  // represents the actions to be taken when task fails.
                  // Note: all actions are executed sequentially one after other.
                {
                    "name": "action-one",
                        // (optional)
                        // Note: `name` can be helpful to distinguish tasks while reading log files,
                        // so, it's recommended to give one (ideally separated with dashes).
                    "cmd": "/path/to/some/script/to/execute.sh",
                        // (required)
                        // The cmd/script to be executed.
                    "continueOnFailure": true
                        // (optional)
                        // To inform the client whether or not to proceed with the next action in the list.
                        // If not mentioned, it wont' proceed to next action if current actions fails.
                },
                {
                    "name": "action-two",
                    "cmd": "/path/to/some/script/to/execute.sh"
                }
            ]
        },
        // example:
        {
            "name": "ex-archival-mount-point-utilization",
            "repeatInterval": 1800,
            "cmd": "/home/dbadmin/wd/scripts/mount-point-utilization.sh",
            "msg": "mount point usage > 90%",
            "actionsToBeTaken": [
                {
                    "name": "delete-arcs-older-than-3-months",
                    "cmd": "/home/dbadmin/wd/scripts/del-old-arcs.sh",
                    "continueOnFailure": true
                },
                {
                    "name": "delete-old-logs",
                    "cmd": "/home/dbadmin/wd/scripts/del-old-logs.sh"
                }
            ]
        }
    ]
}
```

#### Alert Behaviour
| If | Will alert be sent? | Behaviour |
| --- | --- | --- |
| `cmd` completes successfully | No | No alerts. Adds to the log that task was completed successfully. |
| `cmd` fails | Yes | Will log the error and output; and:<br/>If `actionsToBeTaken` is mentioned to the failed task, will proceed with it's execution and then send and alert accordingly: <br/><ul><li>If all actions succeeds, it will send an alert with final status as "OK"</li> <li> If all actions in `actionsToBeTaken` succeeds, it will mention "actions failed to complete" in the alert sent.</li></ul> Else, it will simply send an alert. | 