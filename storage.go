package gblist

import (
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
func New(path string, ttl time.Duration) (storage, error) {
	db, err := bolt.Open(path, 0600, nil)
	s := storage{
		Database: db,
		TTL: ttl,
	}
	return s, err
}

// Add insert or replace an IP address in the given bucket.
func (s *storage) Add(bucket string, ip net.IP) error {
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

// List returns all the IP addresses from the given bucket that have not expired yet.
func (s *storage) List(bucket string) ([]net.IP, error) {
	var list []net.IP
	now := time.Now()
	err := s.Database.Update(func(tx *bolt.Tx) error {
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
		}
		return nil
	})
	return list, err
}

// Close closes the Bolt database
func (s *storage) Close() error {
	return s.Database.Close()
}

// Dump returns a map of the current IPs in the bucket with their expiration time
func (s *storage) Dump(bucket string) (map[string]string, error) {
	var list = make(map[string]string)
	err := s.Database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b != nil {
			b.ForEach(func(k, v []byte) error {
				var ipValue string
				var expirationTime string
				ipValue = string(k)
				ip := net.ParseIP(ipValue)
				if ip == nil {
					ipValue = ipValue + "(invalid)"
				}
				unixTimestamp, err := strconv.ParseInt(string(v), 10, 64)
				if err != nil {
					expirationTime = err.Error()
				}
				expirationTime = time.Unix(unixTimestamp, 0).String()
				list[ipValue] = expirationTime
				return nil
			})
		}
		return nil
	})
	return list, err
}