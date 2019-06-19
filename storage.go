package gblist

import (
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"net"
	"strconv"
	"time"
)

type storage struct {
	Database *bolt.DB
	TTL time.Duration
}

// New opens a Bolt DB database at the given path
func New(path string, days int) storage {

	var ttl time.Duration
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	if days > 0 {
		ttl, err = time.ParseDuration(fmt.Sprintf("%dh", days*24)) // each day 24h
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatalf("invalid number of days: %d", days)
	}
	s := storage{
		Database: db,
		TTL: ttl,
	}
	return s
}

// Add insert or replace an IP address in the given bucket.
func (s *storage) Add(bucket string, ip net.IP) {
	s.Database.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			log.Fatal(err)
		}
		expirationTimestamp := strconv.FormatInt(time.Now().Add(s.TTL).Unix(), 10)
		err = b.Put([]byte(ip.String()), []byte(expirationTimestamp))
		return err
	})
}

// List returns all the IP addresses from the given bucket that have not expired yet.
func (s *storage) List(bucket string) []net.IP {
	var list []net.IP
	now := time.Now()
	s.Database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b != nil {
			b.ForEach(func(k, v []byte) error {
				ipValue := string(k)
				ip := net.ParseIP(ipValue)
				if ip != nil {
					unixTimestamp, err := strconv.ParseInt(string(v), 10, 64)
					if err == nil {
						expirationTime := time.Unix(unixTimestamp, 0)
						if now.Before(expirationTime) {
							list = append(list, ip)
						} else {
							b.Delete(k)
						}
					} else {
						log.Print(err.Error())
						b.Delete(k)
					}
				} else {
					b.Delete(k)
				}
				return nil
			})
		} else {
			log.Fatalf("no bucket %s", bucket)
		}
		return nil
	})
	return list
}

// Close closes the Bolt database
func (s *storage) Close() {
	s.Database.Close()
}