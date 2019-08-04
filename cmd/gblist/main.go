package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/weregoat/gblist"
	"log"
	"net"
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
		scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Text()
				_,ip, err := net.ParseCIDR(strings.TrimSpace(addMask(line)))
				if err != nil {
					log.Print(err)
				} else {
					if ip != nil {
						var err error
						if *purge {
							err = s.Purge(*bucket, ip.String())
						} else {
							err = s.Add(*bucket, *ip)
						}
						if err != nil {
							log.Print(err)
						}
					}
				}
			}
			if err := scanner.Err(); err != nil {
				fmt.Fprintln(os.Stderr, "reading standard input:", err)
			}
	} else {
		if *print {
			list, err := s.List(*bucket)
			if err != nil {
				log.Print(err)
			} else {
				for _, record := range list {
					fmt.Println(record.CIDR.String())
				}
			}
		}
		if *dump {
			fmap := template.FuncMap{
				"format": format,
			}
			tmpl := template.Must(template.New("dump").Funcs(fmap).Parse("{{.ExpirationTime}} {{.CIDR | format}}\n"))
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

// addMask tries to convert a IP address string into a CIDR string.
func addMask(address string) string {
	// If the address already contains a "/" we assume is already correctly masked.
	if strings.Contains(address, "/") {
		return address
	}
	mask := "32"
	// If the address does include a ":", we assume is IPv6 and
	// use the "/128 mask"
	if strings.Contains(address,":") {
		mask = "128"
	}
	address = fmt.Sprintf("%s/%s", address, mask)
	return address
}

// formatIP returns the IP CIDR as string (to be used in template)
func format(ip net.IPNet) string {
	return ip.String()
}