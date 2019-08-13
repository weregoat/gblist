package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/weregoat/gblist"
	"log"
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
	flag.Parse()
	duration := fmt.Sprintf("%dh", 14*24) // 14 days
	if *days != 0 || *hours != 0 || *minutes != 0 {
		duration = fmt.Sprintf("%dm", (*days*24+*hours)*60+*minutes)
	}
	ttl, err := time.ParseDuration(duration)
	if err != nil {
		log.Fatalf("could not parse duration for banning time because error: %s", err.Error())
	}
	s, err := gblist.Open(*databasePath, ttl)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	if !*print && !*dump {
		// https://golang.org/pkg/flag/#NArg
		// If there are not args left we expect a pipe
		if flag.NArg() == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				ip := strings.TrimSpace(scanner.Text())
				valid, err := s.IsValid(ip)
				if valid {
					if *purge {
						err = s.Purge(*bucket, ip)
					} else {
						err = s.Add(*bucket, ip)
					}
				}
				if err != nil {
					log.Print(err)
				}
			}
		} else { // We process all the IP given (more than one allowed)
			for _, ip := range flag.Args() {
				process(s, ip, *bucket, *purge)
			}
		}
	} else {
		if *print {
			list, err := s.List(*bucket)
			if err != nil {
				log.Print(err)
			} else {
				for _, record := range list {
					fmt.Println(record.IP)
				}
			}
		}
		if *dump {
			tmpl := template.Must(template.New("dump").Parse("{{.ExpirationTime}} {{.IP}}\n"))
			dump, err := s.Dump(*bucket)
			if err != nil {
				log.Fatal(err)
			} else {
				for _, record := range dump {
					tmpl.Execute(os.Stdout, record)
				}
			}
		}
	}
}

func process(storage gblist.Storage, ip string, bucket string, purge bool) {
	valid, err := storage.IsValid(ip)
	if valid {
		if purge {
			err = storage.Purge(bucket, ip)
		} else {
			err = storage.Add(bucket, ip)
		}
	}
	if err != nil {
		log.Print(err)
	}
}
