package main

import (
	"errors"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

//go:generate go-enum --marshal --noprefix

/*
ENUM(

	Up
	Down

)
*/
type Status int

/*
ENUM(

	ICMP

)
*/
type LiveMethod int

func monitor(host string, method LiveMethod, interval, tolerance time.Duration) (chan Status, error) {
	results := make(chan Status, 16)
	switch method {
	case ICMP:
		monitor, err := probing.NewPinger(host)
		if err != nil {
			return nil, err
		}

		monitor.SetPrivileged(true)

		laststate := Down
		lastup := time.Now()

		monitor.OnRecv = func(_ *probing.Packet) {
			lastup = time.Now()
			if laststate != Up {
				laststate = Up
				results <- Up
			}
		}
		/*
			monitor.OnRecvError = func(err error) {
				log.Println(err)
			}
		*/
		monitor.OnSendError = func(*probing.Packet, error) {
			panic(err)
		}
		go func() {
			for range time.NewTicker(interval / 10).C {
				if time.Since(lastup) > tolerance {
					if laststate != Down {
						laststate = Down
						results <- Down
					}
				}
			}
		}()

		monitor.Interval = interval
		go func() {
			monitor.Run()
		}()

		return results, nil
	}
	return nil, errors.New("unsupported live method")
}
