package main

import (
	"log"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/spf13/pflag"
	"golang.org/x/exp/slices"
)

var (
	// regular expression matching a host name with a dot
	hostnamewithdotre = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)+([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)
	// regular expression matching a ipv4 address
	ipv4re = regexp.MustCompile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)

	// regular expression matching a hostname or ip prefixed with a username and possibly a password
	usernamepasswordre = regexp.MustCompile(`^(?:\S+(?::\S+)?@)(\S+)$`)
)

func main() {
	// parse command line options
	ourargs := os.Args[1:]
	commandargs := []string{}

	// Try to split between our arguments and the command by looking for "--"
	dashdashpos := slices.Index(ourargs, "--")
	if dashdashpos != -1 {
		ourargs = ourargs[:dashdashpos]
		commandargs = ourargs[dashdashpos+1:]
	}

	flags := pflag.NewFlagSet("whenup", pflag.ExitOnError)

	host := flags.StringP("host", "h", "", "host to check")
	interval := pflag.IntP("interval", "i", 200, "interval between checks, in milliseconds")
	delay := pflag.IntP("delay", "d", 2000, "delay before executing command after host becoming up, in milliseconds")
	modetext := pflag.StringP("mode", "m", "icmp", "host live check method (icmp only for now)")
	tolerance := pflag.IntP("tolerance", "t", 1000, "tolerance for being down, in milliseconds")

	notify := flags.BoolP("notify", "n", true, "notify using toasts when host changes status")
	continuous := flags.BoolP("continuous", "c", false, "continious monitoring instead of one-off")
	kill := flags.BoolP("kill", "k", false, "kill launched command when host goes down")

	flags.Parse(ourargs)

	if len(flags.Args()) > 0 {
		if len(commandargs) > 0 {
			panic("Cannot have both command line args before and after dashdash (--)")
		}
		commandargs = flags.Args()
	}

	autodetecthost := true
	if *host != "" {
		autodetecthost = false
	}
	if autodetecthost {
		// if host is blank, look through command args for something that looks like a host name
		if *host == "" {
			for _, arg := range commandargs {
				if hostnamewithdotre.MatchString(arg) {
					*host = arg
					break
				}
			}
		}
		// if host is blank, look through command args for something that looks like an ip address
		if *host == "" {
			for _, arg := range commandargs {
				if ipv4re.MatchString(arg) {
					*host = arg
					break
				}
			}
		}
		// if host is blank, try to do URL parsing and find the host that way
		if *host == "" {
			for _, arg := range commandargs {
				// parse the URL
				u, err := url.Parse(arg)
				if err != nil {
					continue
				}
				// Fairly certain it's a host if there's a scheme or a user name
				if u.Scheme != "" || u.User != nil {
					*host = u.Hostname()
				}
				break
			}
		}
		// if host is blank, try to do username and password hostname matching
		if *host == "" {
			for _, arg := range commandargs {
				match := usernamepasswordre.FindStringSubmatch(arg)
				if match != nil {
					*host = match[1]
					break
				}
			}
		}
		// If the host is still not found, panic
		if *host == "" {
			panic("Can not autodetect host to monitor")
		}
	}
	if autodetecthost {
		log.Println("Detected host as", *host)
	}

	mode := ICMP
	switch strings.ToLower(*modetext) {
	case "icmp":
	default:
		panic("Unsupported live method " + *modetext)
	}

	mon, err := monitor(*host, mode, time.Duration(*interval)*time.Millisecond, time.Duration(*tolerance)*time.Millisecond)
	if err != nil {
		panic(err)
	}

	var cmd *exec.Cmd
mainloop:
	for status := range mon {
		if *notify {
			beeep.Notify("whenup", "Host "+*host+" is "+status.String(), "")
		}
		log.Println("Host " + *host + " is " + status.String())

		if *kill && status == Down && cmd != nil && cmd.Process != nil {
			log.Println("Terminating running process")
			cmd.Process.Kill()
			cmd = nil
		}

		// if command line args were given, run the command
		if status == Up && len(commandargs) > 0 && cmd == nil {
			log.Println("Starting process")
			if *delay > 0 {
				time.Sleep(time.Duration(*delay) * time.Millisecond)
			}
			cmd = exec.Command(commandargs[0], commandargs[1:]...)

			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			// Launch it
			cmd.Start()
		}

		// we saw that it was up, so we're done
		if !*continuous && status == Up {
			break mainloop
		}
	}
	// If we're here, and we launched something, we wait for it to terminate
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Wait()
	}
}
