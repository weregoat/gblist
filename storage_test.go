package gblist

import (
	"net"
	"os"
	"testing"
	"time"
)

const DB = "test.db"
const BUCKET = "test"

func TestNew(t *testing.T) {
	ttl, err := time.ParseDuration("10m")
	s, err := Open(DB, ttl)
	if err != nil {
		t.Error(err)
	}
	s.Close()
	err = os.Remove(DB)
	if err != nil {
		t.Error(err)
	}
}

func TestStorage_Add(t *testing.T) {
	ip1 := net.ParseIP("127.0.0.1")
	ip2 := net.ParseIP("192.168.0.0")
	ttl, err := time.ParseDuration("10m")
	if err != nil {
		t.Error(err)
	}
	s, err := Open(DB, ttl)
	if err != nil {
		t.Error(err)
	}
	err = s.Add(BUCKET, ip1)
	if err != nil {
		t.Error(err)
	}

	err = s.Add(BUCKET, ip1)
	if err != nil {
		t.Error(err)
	}

	err = s.Add(BUCKET, ip2)
	if err != nil {
		t.Error(err)
	}
	list, err := s.List(BUCKET)
	if err != nil {
		t.Error(err)
	}
	if len(list) != 2 {
		t.Errorf("wrong number of elements %d", len(list))
	}
	for _, v := range list {
		if !v.IPAddress.Equal(ip1) && !v.IPAddress.Equal(ip2) {
			t.Errorf("Invalid element %s in database", v.IPAddress.String())
		}
	}
	s.Close()
	err = os.Remove(DB)
	if err != nil {
		t.Error(err)
	}
}

func TestStorage_List(t *testing.T) {
	ip1 := net.ParseIP("127.0.0.1")
	ip2 := net.ParseIP("192.168.0.0")
	ttl, err := time.ParseDuration("1ns")
	if err != nil {
		t.Error(err)
	}
	s, err := Open(DB, ttl)
	if err != nil {
		t.Error(err)
	}
	err = s.Add(BUCKET, ip1)
	if err != nil {
		t.Error(err)
	}

	err = s.Add(BUCKET, ip1)
	if err != nil {
		t.Error(err)
	}

	err = s.Add(BUCKET, ip2)
	if err != nil {
		t.Error(err)
	}
	list, err := s.List(BUCKET)
	if err != nil {
		t.Error(err)
	}
	if len(list) != 0 {
		t.Errorf("wrong number of elements %d", len(list))
	}
	// Check the elements have been deleted too
	dump, err := s.Dump(BUCKET)
	if err != nil {
		t.Error(err)
	}
	if len(dump) != 0 {
		t.Errorf("wrong number of elements %d (some were not deleted)", len(dump))
	}
	s.Close()
	err = os.Remove(DB)
	if err != nil {
		t.Error(err)
	}
}

func TestStorage_Purge(t *testing.T) {
	ip1 := net.ParseIP("127.0.0.1")
	ip2 := net.ParseIP("192.168.0.0")
	ttl, err := time.ParseDuration("90m")
	if err != nil {
		t.Error(err)
	}
	s, err := Open(DB, ttl)
	if err != nil {
		t.Error(err)
	}
	err = s.Add(BUCKET, ip1)
	if err != nil {
		t.Error(err)
	}

	err = s.Add(BUCKET, ip1)
	if err != nil {
		t.Error(err)
	}

	err = s.Purge(BUCKET, ip2.String())
	if err != nil {
		t.Error(err)
	}
	list, err := s.List(BUCKET)
	if err != nil {
		t.Error(err)
	}
	if len(list) != 1 {
		t.Errorf("wrong number of elements %d", len(list))
	}
	// Check the elements have been deleted too
	dump, err := s.Dump(BUCKET)
	if err != nil {
		t.Error(err)
	}
	if len(dump) != 1 {
		t.Errorf("wrong number of elements %d (some were not deleted)", len(dump))
	}
	s.Close()
	err = os.Remove(DB)
	if err != nil {
		t.Error(err)
	}
}
