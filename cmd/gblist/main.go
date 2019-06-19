package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/weregoat/gblist"
	"net"
	"os"
	"strings"
)



func main() {
	var databasePath = flag.String("db", "/tmp/gblist.db", "full path of the database file")
	var print = flag.Bool("print", false, "print the non expired IP addresses from the database")
	var days = flag.Int("days", 14, "banning time in days")
	var bucket = flag.String("bucket", "default", "name of the bucket for storing IP addresses")
	flag.Parse()
	s := gblist.New(*databasePath, *days)
	defer s.Close()
	if *print {
		for _, ip := range s.List(*bucket) {
			fmt.Println(ip.String())
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			ip := net.ParseIP(strings.TrimSpace(line))
			if ip != nil {
				s.Add(*bucket, ip)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
	}

}

