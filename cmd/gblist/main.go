package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/weregoat/gblist"
	"os"
	"strings"
	"text/template"
	"time"
)

func main() {
	var databasePath = flag.String("db", "/tmp/gblist.db", "full path of the database file")
	var print = flag.Bool("print", false, "print the non expired IP addresses from the database")
	var days = flag.Int("days", 0, "number of days of banning time (they all sum up)")
	var hours = flag.Int("hours", 0, "number of hours of banning time (they all sum up)")
	var minutes = flag.Int("minutes", 0, "number of minutes of banning time (they all sum up)")
	var bucket = flag.String("bucket", "default", "name of the bucket for storing IP addresses")
	var dump = flag.Bool("dump", false, "dump the result of the database")
	var purge = flag.Bool("purge", false, "remove the given IPs from the bucket")
	var query = flag.Bool("query", false, "query the database for the given IP")
	var description = flag.String("description", "", "add the given text as description for the record")
	flag.Parse()
	duration := fmt.Sprintf("%dh", 14*24) // 14 days

	if *days != 0 || *hours != 0 || *minutes != 0 {
		duration = fmt.Sprintf("%dm", (*days*24+*hours)*60+*minutes)
	}
	ttl, err := time.ParseDuration(duration)
	if err != nil {
		printError(fmt.Sprintf("could not parse duration for banning time because error: %s", err.Error()), true)
	}
	s, err := gblist.Open(*databasePath, ttl)
	if err != nil {
		printError(err, true)
	}
	defer s.Close()
	if !*print && !*dump {
		// https://golang.org/pkg/flag/#NArg
		// If there are not args left we expect a pipe
		if flag.NArg() == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				ip := strings.TrimSpace(scanner.Text())
				process(s, ip, *description, *bucket, *purge, *query)
				/*
					if *query {
						record, err := s.Fetch(*bucket, ip)
						if err != nil {
							log.Fatal(err)
						}
						if record.IsValid() {
							tmpl.Execute(os.Stdout, record)
						}
					} else if *purge {
						err = s.Purge(*bucket, ip)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						record, err := gblist.New(ip, s.TTL, *description)
						if err != nil {
							log.Print(err)
						}
						err = s.Add(*bucket, record)
						if err != nil {
							log.Fatal(err)
						}
					} */
			}
		} else { // We process all the IP given (more than one allowed)
			for _, ip := range flag.Args() {
				process(s, ip, *description, *bucket, *purge, *query)
			}
		}
	} else {
		if *print {
			list, err := s.List(*bucket)
			if err != nil {
				printError(err, false)
			} else {
				for _, record := range list {
					fmt.Println(record.IP)
				}
			}
		}
		if *dump {
			tmpl := template.Must(template.New("dump").Parse("IP: {{.IP}}\nExpiration time: {{.ExpirationTime}}\nDescription: \"{{.Description}}\"\n\n"))
			dump, err := s.Dump(*bucket)
			if err != nil {
				printError(err, true)
			} else {
				for _, record := range dump {
					tmpl.Execute(os.Stdout, record)
				}
			}
		}
	}
}

func process(storage gblist.Storage, ip string, description string, bucket string, purge bool, query bool) {
	if len(ip) == 0 {
		return
	}
	if query {
		record, err := storage.Fetch(bucket, ip)
		if err != nil {
			printError(err, true)
		}
		if record.IsValid() {
			tmpl := template.Must(template.New("dump").Parse("IP: {{.IP}}\nExpiration time: {{.ExpirationTime}}\nDescription: \"{{.Description}}\"\n\n"))
			tmpl.Execute(os.Stdout, record)
		}
	} else if purge {
		err := storage.Purge(bucket, ip)
		if err != nil {
			printError(err, true)
		}
	} else {
		record, err := gblist.New(ip, storage.TTL, description)
		if err != nil {
			printError(err, true)
		}
		err = storage.Add(bucket, record)
		if err != nil {
			printError(err, true)
		}
	}
}

// Error prints an error on StdErr and exits (or not)
func printError(message interface{}, exit bool) {
	fmt.Fprintln(os.Stderr, message)
	if exit {
		os.Exit(1)
	}
}
