# WatchDog 🤪

Monitor servers through plug in scripts.

Three components:

[**Client**](cmd/client)s running on multiple servers(that are to be monitored). Requires a configuration file with format mentioned at [confs/client-conf-format.json](./confs/client-conf-format.json) and runs the tasks mentioned in the conf file. On error, it sends message to **server**.

```
Usage of ./clientx:
  -c string
        path to cfg file (default "congif.json")
  -r string
        server address in the format IP:PORT (default "localhost:40090")
```
The [**Server**](cmd/server) is a gRPC server listening on port 40090. Multiple **client**s can connect to it and share errors/alerts. It then broadcasts the same to **wdc**s.

[**wdc**](cmd/wdc)s are clients that run on monitoring spoc's local machines. It connects to the **server** through websockets and listens to incoming alerts.

```
Usage of ./wdcx:
  -r string
        http service address (default "localhost:40080")
```