package main

import (
	"flag"
	"os"
	"testing"
)

// TestMain defaults -parallel to 8, as make coverage does: wider, the pure-Go SQLite engine's process-wide
// allocator lock serializes every test's queries and the async waits starve on many-core hosts.
func TestMain(m *testing.M) {
	flag.Parse()
	set := false
	flag.Visit(func(f *flag.Flag) { set = set || f.Name == "test.parallel" })
	if !set {
		if err := flag.Set("test.parallel", "8"); err != nil {
			panic(err)
		}
	}
	os.Exit(m.Run())
}
