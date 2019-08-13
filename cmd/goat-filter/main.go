package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/weregoat/gblist"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// Config is the definition of the YAML configuration elements
type Config struct {
	Sources   []string `yaml:"sources"`
	Patterns  []string `yaml:"patterns"`
	Database  string   `yaml:"database"`
	Bucket    string   `yaml:"bucket"`
	TTL       string   `yaml:"ttl"`
	WhiteList []string `yaml:"network_whitelist"`
	Template  string   `yaml:"print_template"`
}

// Settings are the settings from the configuration after parsing
type Settings struct {
	Storage   *gblist.Storage
	RegExps   []*regexp.Regexp
	Sources   []string
	Bucket    string
	WhiteList []*net.IPNet
	Template  *template.Template
}

func main() {
	config := flag.String("config", "", "YAML configuration file")
	print := flag.Bool("print", false, "prints the content of the database after parsing")
	flag.Parse()
	if len(*config) == 0 {
		log.Fatalf("Missing path to configuration file argument")
	}

	cfg, err := loadConfig(*config)
	if err != nil {
		log.Fatal(err)
	}

	settings, err := parseConfig(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer settings.Storage.Close()

	// Parses every source file and put the submatched IP into the database
	for _, source := range cfg.Sources {
		file, err := os.Open(source)
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			for _, re := range settings.RegExps {
				matches := re.FindAllStringSubmatch(scanner.Text(), -1)
				for _, match := range matches {
					ip := strings.TrimSpace(match[1])
					var ipAddress net.IP
					if len(ip) > 0 { // No reason to waste time on an empty string
						if strings.Contains(ip, "/") { // Dirty check for getting CIDR
							ipAddress, _, err = net.ParseCIDR(ip)
							if err != nil { // With an error the ipAddress should be null anyway.
								ipAddress = nil // We make sure, in any case.
							}
						} else { // Otherwise we assume is a single IP address
							ipAddress = net.ParseIP(ip) // If it cannot be parsed it will return a nil
						}
						if ipAddress != nil {
							if !isWhitelisted(ipAddress, settings.WhiteList) {
								// Notice that given the way the storage library
								// uses bolt API each add is a transaction, and
								// in case of error whatever was added is not
								// rolled back.
								// Which is fine for my scope.
								// Notice that we are adding the matching string, not the ipAddress
								// as in case of a parsed CIDR is not what we want.
								err = settings.Storage.Add(settings.Bucket, ip)
								if err != nil {
									break
								}
							}
						}
					}
				}

			}
		}
		if err != nil {
			err = scanner.Err()
		}
		file.Close()
		if err != nil {
			log.Fatal(err)
		}

	}

	// If asked, print the blacklisted IPs for processing
	if *print {
		list, err := settings.Storage.List(settings.Bucket)
		if err != nil {
			log.Fatal(err)
		}
		for _, record := range list {
			if settings.Template != nil {
				settings.Template.Execute(os.Stdout, record)
			} else {
				fmt.Println(record.IP)
			}
		}
	}

}

// parseConfig parses the YAML configuration and returns the setting (or an error)
func parseConfig(cfg *Config) (settings Settings, err error) {
	for _, element := range cfg.Sources {
		source := strings.TrimSpace(element)
		if len(source) > 0 {
			settings.Sources = append(settings.Sources, strings.TrimSpace(source))
		}
	}
	if len(settings.Sources) == 0 {
		err = errors.New("no valid source defined")
		return
	}
	TTLString := fmt.Sprintf("%dh", 21*24)
	if len(cfg.TTL) > 0 {
		TTLString = cfg.TTL
	}
	ttl, err := parseTTL(TTLString)
	if err != nil {
		return
	}
	storage, err := gblist.Open(cfg.Database, ttl)
	if err != nil {
		return
	}
	settings.Storage = &storage
	bucket := strings.TrimSpace(cfg.Bucket)
	if len(bucket) > 0 {
		settings.Bucket = bucket
	} else {
		err = errors.New("invalid bucket")
		return
	}
	regExps := make([]*regexp.Regexp, 0)
	for _, pattern := range cfg.Patterns {
		re, compileErr := regexp.Compile(pattern)
		if compileErr != nil {
			err = errors.New(fmt.Sprintf("failed to compile regexp %s: %s", pattern, compileErr.Error()))
			return
		}
		regExps = append(regExps, re)
	}
	settings.RegExps = regExps
	for _, address := range cfg.WhiteList {
		_, ipNet, parseErr := net.ParseCIDR(address)
		if parseErr != nil {
			err = errors.New(fmt.Sprintf("failed to parse whitelisted CIDR %s: %s", address, parseErr.Error()))
			return
		} else {
			settings.WhiteList = append(settings.WhiteList, ipNet)
		}
	}
	if len(cfg.Template) > 0 {
		tmpl, err := template.New("print").Parse(cfg.Template)
		if err != nil {
			return settings, err
		}
		settings.Template = tmpl
	}
	return
}

// Parses the TTL into a duration. I added weeks and days to the time parser.
// Not too happy about it, but it works well enough.
func parseTTL(interval string) (time.Duration, error) {
	var ttl time.Duration
	weeks := 0
	days := 0
	var err error
	// Search for the string "w"
	weeksIndex := strings.Index(interval, "w")
	// If present get a slice of the before that, which should be the number of weeks
	if weeksIndex > 0 {
		weeks, err = strconv.Atoi(interval[:weeksIndex])
		if err != nil {
			return ttl, err
		}
	}
	// Same with days "d"
	daysIndex := strings.Index(interval, "d")
	if daysIndex > 0 {
		days, err = strconv.Atoi(interval[weeksIndex+1 : daysIndex])
		if err != nil {
			return ttl, err
		}
	} else {
		// In case there are no days, but there are weeks, we shift the index
		daysIndex = weeksIndex
	}
	// Converts weeks and days into hours and then into a duration
	weeksAndDays, err := time.ParseDuration(fmt.Sprintf("%dh", weeks*7*24+days*24))
	if err != nil {
		return ttl, err
	}
	// Whatever is left should be parsed as normal time duration
	rest := interval[daysIndex+1:]
	if len(rest) == 0 {
		rest = "0h"
	}
	hoursAndMinutes, err := time.ParseDuration(rest)
	if err != nil {
		return ttl, err
	}

	// Adds everything (converted to Nanoseconds because is not a float); but it doesn't really matter
	ttl, err = time.ParseDuration(fmt.Sprintf("%dns", weeksAndDays.Nanoseconds()+hoursAndMinutes.Nanoseconds()))
	if err != nil {
		return ttl, err
	}
	return ttl, err
}

// isWhitelisted returns if a given IP belongs to a whitelisted network
func isWhitelisted(ip net.IP, whitelist []*net.IPNet) bool {
	for _, network := range whitelist {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// loadConfig loads the configuration file and parses the YAML
func loadConfig(path string) (Config, error) {
	cfg := Config{}
	filename, err := filepath.Abs(path)
	if err != nil {
		return cfg, err
	}
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return cfg, err
	}

	err = yaml.Unmarshal(content, &cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, err
}
