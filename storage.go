package gblist

import (
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

type Record struct {
	IP             string
	ExpirationTime time.Time
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
func (s *Storage) Add(bucket string, ip string) error {
	valid, err := s.IsValid(ip)
	if valid {
		err = s.Database.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				log.Fatal(err)
			}
			expirationTimestamp := strconv.FormatInt(time.Now().Add(s.TTL).Unix(), 10)
			err = b.Put([]byte(ip), []byte(expirationTimestamp))
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
	var records []Record
	var purge []string
	err := s.Database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b != nil {
			b.ForEach(func(k, v []byte) error {
				ip := string(k)
				unixTimestamp, timeError := strconv.ParseInt(string(v), 10, 64)
				valid, _ := s.IsValid(ip)
				if valid && timeError == nil {
					expirationTime := time.Unix(unixTimestamp, 0)
					record := Record{
						IP:             ip,
						ExpirationTime: expirationTime,
					}
					records = append(records, record)
				} else {
					purge = append(purge, ip)
				}
				return nil
			})
		}
		return nil
	})
	err = s.Purge(bucket, purge...)
	return records, err
}

// isValid tries to parse an IP (address or CIDR) and return true if it succeed; false otherwise
func (s *Storage) IsValid(ip string) (valid bool, err error) {
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