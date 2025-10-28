// Command pushover is a simple cli tool to send pushover notifications.
//
// Run with -printconfig to see an example config file.
// Use -configpath to override the default /etc/pushover.conf.
//
// Example:
//
//	pushover -priority high -title 'Bad stuff' 'This is the message. There has been an unfortunate incident.'
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mjl-/sconf"
)

var config struct {
	AppToken string `sconf-doc:"Token identifying the sending application."`
	DestKey  string `sconf-doc:"Key selecting the destination user or group."`
	Title    string `sconf:"optional" sconf-doc:"Title to show with message, instead of application name."`
}

func xcheckf(err error, format string, args ...any) {
	if err != nil {
		log.Fatalf("%s: %s", fmt.Sprintf(format, args...), err)
	}
}

func main() {
	var configPath = "/etc/pushover.conf"
	var priority string
	var title string
	var retry = 300
	var expire = 3600
	var timeout = 30 * time.Second
	var printConfig bool

	log.SetFlags(0)
	flag.BoolVar(&printConfig, "printconfig", false, "print empty config file and exit")
	flag.StringVar(&configPath, "configpath", configPath, "path to config file")
	flag.StringVar(&priority, "priority", priority, "priority to send with: low, lowest, normal (default), high, highest")
	flag.StringVar(&title, "title", "", "title to show with message, instead of possible value from config file, or the default: the application name")
	flag.IntVar(&retry, "retry", retry, "interval between resends of highest priority notifications until they are acknowledged; at most 50 retries are attempted by pushover")
	flag.IntVar(&retry, "expire", expire, "interval after which highest priority notifications aren't retried anymore")
	flag.DurationVar(&timeout, "timeout", timeout, "timeout for call to pushover api")
	flag.Usage = func() {
		log.Println("usage: pushover [flags] message...")
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()

	if printConfig {
		sconf.Describe(os.Stdout, config)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
	}
	msg := strings.Join(args, " ")

	err := sconf.ParseFile(configPath, &config)
	xcheckf(err, "parsing config file")

	// https://pushover.net/api
	data := url.Values{}
	data.Set("token", config.AppToken)
	data.Set("user", config.DestKey)
	data.Set("message", msg)
	var p string
	switch priority {
	case "lowest", "-2":
		p = "-2"
	case "low", "-1":
		p = "-1"
	case "", "normal", "0":
		p = "0"
	case "high", "1":
		p = "1"
	case "highest", "2":
		p = "2"
	default:
		log.Printf("invalid priority value %q", priority)
		flag.Usage()
	}
	if p != "0" {
		data.Set("priority", p)
	}
	if p == "2" {
		data.Set("retry", fmt.Sprintf("%d", retry))
		data.Set("expire", fmt.Sprintf("%d", expire))
	}

	if title == "" {
		title = config.Title
	}
	if title != "" {
		data.Set("title", title)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.pushover.net/1/messages.json", strings.NewReader(data.Encode()))
	xcheckf(err, "making request")

	resp, err := http.DefaultClient.Do(req)
	xcheckf(err, "api request")

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if err != nil {
			log.Printf("warning: reading error response body: %v", err)
		}
		log.Fatalf("got status %q, expected 200 ok, body %q", resp.Status, respBody)
	}
}
