package gblist

import (
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
	ttl, err := time.ParseDuration("10m")
	if err != nil {
		t.Error(err)
	}

	r1 := createRecord("193.22.0.0/16", "Something", ttl, t)
	r2 := createRecord("10.55.11.12", "Something else", ttl, t)

	s, err := Open(DB, ttl)
	if err != nil {
		t.Error(err)
	}
	err = s.Add(BUCKET, r1)
	if err != nil {
		t.Error(err)
	}

	err = s.Add(BUCKET, r1) // Add the same element again, to check overwrites
	if err != nil {
		t.Error(err)
	}

	err = s.Add(BUCKET, r2)
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
		if (v.IP != r1.IP && v.IP != r2.IP) || (v.Description != r1.Description && v.Description != r2.Description) {
			t.Errorf("Invalid element %s in database", v.IP)
		}
	}
	s.Close()
	err = os.Remove(DB)
	if err != nil {
		t.Error(err)
	}
}

func TestStorage_List(t *testing.T) {
	ttl, err := time.ParseDuration("1ns")
	if err != nil {
		t.Error(err)
	}

	r1 := createRecord("193.22.0.0/16", "", ttl, t)
	r2 := createRecord("10.55.11.12", "", ttl, t)

	s, err := Open(DB, ttl)
	if err != nil {
		t.Error(err)
	}
	err = s.Add(BUCKET, r1)
	if err != nil {
		t.Error(err)
	}

	// Add the same record again; it should be overwritten
	err = s.Add(BUCKET, r1)
	if err != nil {
		t.Error(err)
	}

	err = s.Add(BUCKET, r2)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Duration(5))
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
	ttl, err := time.ParseDuration("90m")
	if err != nil {
		t.Error(err)
	}

	r1 := createRecord("127.0.0.1", "", ttl, t)
	r2 := createRecord("192.168.1.0/24", "", ttl, t)

	s, err := Open(DB, ttl)
	if err != nil {
		t.Error(err)
	}

	err = s.Add(BUCKET, r1)
	if err != nil {
		t.Error(err)
	}

	err = s.Add(BUCKET, r2)
	if err != nil {
		t.Error(err)
	}

	err = s.Purge(BUCKET, r2.IP)
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
	} else {
		// Check the correct record was deleted
		record := dump[0]
		if record.IP != r1.IP {
			t.Errorf("wrong record deleted")
		}
	}
	s.Close()
	err = os.Remove(DB)
	if err != nil {
		t.Error(err)
	}
}

func TestRecord_New(t *testing.T) {
	_, err := New("8888", time.Duration(1000), "")
	if err == nil {
		t.Errorf("failed to verify IP correctly")
	}
}

func createRecord(ip, description string, ttl time.Duration, t *testing.T) Record {
	record, err := New(ip, ttl, description)
	if err != nil {
		t.Error(err)
	}
	return record
}
