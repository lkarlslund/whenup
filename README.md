# whenup
Allows you to start processes when a host is up, or just monitor its status using ICMP

If you're like me and doing lots of reconfigurations to stuff, that require a restart, it's very useful to be able to connect or run a command when the unit is ready again. The 'whenup' tool solves this:

````
whenup ssh myuser@mythingamabob
````

This tests for ICMP ping responses from mythingamabob before running the ssh command. The tool automagically figures out that mythingamabob is the host to check, using various hardcoded match patterns - this can be overridden on the command line, and there are lots of options to customize the behaviour.

````
whenup [-h|--host hostname|ip] [-i|--interval 200] [-t|--tolerance 1000] [-d|--delay 2000] [-m|--mode icmp] [-n|--notify true] [-c|--continious false] [-k|--kill false] [--] command [arguments]
````

* Host: Either a DNS resolvable name or an IP for the host to monitor
* Interval: Test if the host is reachable/ready every N milliseconds
* Tolerance: The host is down if it's not responsive at all after N milliseconds
* Delay: Wait N milliseconds after detecting the host is up, before running the command
* Mode: Test host reachable/ready using ICMP (only option for now)
* Notify: Use toasts to indicate a status change for the monitored host
* Continious: Switch from oneshot command mode to loop mode
* Kill: Terminate the launched process if the host is detected as down

On Linux you'll probably need to allow the binary to send ICMP packets:

````
setcap cap_net_raw=+ep /path/to/whenup
````

