package gblist

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type Storage struct {
	Database *bolt.DB
	TTL      time.Duration
}

// Opens a Bolt DB database at the given path
func Open(path string, ttl time.Duration) (Storage, error) {
	db, err := bolt.Open(path, 0600, nil)
	s := Storage{
		Database: db,
		TTL:      ttl,
	}
	return s, err
}

// Add insert or replace an IP address in the given bucket.
func (s *Storage) Add(bucket string, record Record) error {
	valid, err := IsValid(record.IP) // Double checking this, as the property is public.
	if valid {
		err = s.Database.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				log.Fatal(err)
			}
			payload, err := json.Marshal(&record)
			if err == nil {
				err = b.Put([]byte(record.IP), payload)
			}
			return err
		})
	}
	return err
}

// List returns all the IP addresses from the given bucket that have not expired yet
// and purges the expired records from the database.
func (s *Storage) List(bucket string) ([]Record, error) {
	var list []Record
	var purge []string
	now := time.Now()
	records, err := s.Dump(bucket)
	if err == nil {
		for _, record := range records {
			if now.Before(record.ExpirationTime) {
				list = append(list, record)
			} else {
				purge = append(purge, record.IP)
			}
		}
	}
	err = s.Purge(bucket, purge...)
	return list, err
}

// Purge removes records from the database
func (s *Storage) Purge(bucket string, addresses ...string) error {
	err := s.Database.Update(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte(bucket))
		if b != nil {
			for _, ip := range addresses {
				err = b.Delete([]byte(ip))
				if err != nil {
					break
				}
			}
		} else {
			err = errors.New(fmt.Sprintf("no %s bucket found", bucket))
		}
		return err
	})
	return err
}

// Close closes the Bolt database
func (s *Storage) Close() error {
	return s.Database.Close()
}

// Dump returns a slice of the current (valid) IPs in the bucket and purges invalid ones
func (s *Storage) Dump(bucket string) ([]Record, error) {
	var entries []Record
	var purge []string
	err := s.Database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b != nil {
			b.ForEach(func(k, v []byte) error {
				var record Record
				parseErr := json.Unmarshal(v, &record)
				if parseErr != nil {
					// Compatibility check with older format, where the value was just a timestamp
					unixTimestamp, timeError := strconv.ParseInt(string(v), 10, 64)
					if timeError == nil {
						record.IP = string(k)
						record.ExpirationTime = time.Unix(unixTimestamp, 0)
						parseErr = nil
					}
				}
				valid, _ := IsValid(record.IP)
				if valid && parseErr == nil {
					entries = append(entries, record)
				} else {
					purge = append(purge, string(k))
				}
				return nil
			})
		}
		return nil
	})
	err = s.Purge(bucket, purge...)
	return entries, err
}

// Fetch tries to fetch a record from the given IP and bucket.
// The function does not return an error if the record is not present,
// as it's not properly an error (may add a bool in the return, for
// such a case), but the record is not a valid one.
// See: Record.IsValid()
func (s *Storage) Fetch(bucket string, ip string) (Record, error) {
	var record Record
	err := s.Database.View(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte(bucket))
		if b != nil {
			payload := b.Get([]byte(ip))
			if payload != nil {
				err = json.Unmarshal(payload, &record)
			}
		} else {
			err = errors.New(fmt.Sprintf("no %s bucket found", bucket))
		}
		return err
	})
	return record, err
}

// IsValid tries to parse an IP (address or CIDR) and return true if it succeed; false otherwise
func IsValid(ip string) (valid bool, err error) {
	var address net.IP
	// Assume that it's a CIDR if it has "/"
	if strings.Contains(ip, "/") {
		var network *net.IPNet
		address, network, err = net.ParseCIDR(ip)
		if network == nil && address != nil {
			address = nil
		}
	} else {
		// Otherwise we assume is a single IP address
		address = net.ParseIP(ip)
	}
	if address != nil && err == nil {
		valid = true
	} else {
		valid = false
		errorText := fmt.Sprintf("%s is not a valid IP address or CIDR", ip)
		if err != nil {
			errorText = fmt.Sprintf("%s: %s", errorText, err.Error())
		}
		err = errors.New(errorText)
	}
	return valid, err
}
