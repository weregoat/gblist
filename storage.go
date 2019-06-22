package gblist

import (
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"net"
	"strconv"
	"time"
)

type Storage struct {
	Database *bolt.DB
	TTL      time.Duration
}

type Record struct {
	IPAddress net.IP
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
func (s *Storage) Add(bucket string, ip net.IP) error {
	err := s.Database.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			log.Fatal(err)
		}
		expirationTimestamp := strconv.FormatInt(time.Now().Add(s.TTL).Unix(), 10)
		err = b.Put([]byte(ip.String()), []byte(expirationTimestamp))
		return err
	})
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
				purge = append(purge, record.IPAddress.String())
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
				ip := net.ParseIP(string(k))
				unixTimestamp, err := strconv.ParseInt(string(v), 10, 64)
				if err == nil && ip != nil {
					expirationTime := time.Unix(unixTimestamp, 0)
					record := Record{
						IPAddress: ip,
						ExpirationTime: expirationTime,
					}
					records = append(records, record)
				} else {
					purge = append(purge, string(k))
				}
				return nil
			})
		}
		return nil
	})
	err = s.Purge(bucket, purge...)
	return records, err
}
