package core_test

import (
	"testing"

	"github.com/polagonow/pola/core"
)

func TestFSFileInfoZeroValue(t *testing.T) {
	var fi core.FSFileInfo
	if fi.Name != "" || fi.IsDir || fi.Size != 0 {
		t.Fatal("FSFileInfo zero value should be empty")
	}
}

func TestCacheOptionsZeroValue(t *testing.T) {
	var opts core.CacheOptions
	if opts.TTL != 0 {
		t.Fatal("CacheOptions zero TTL should be 0")
	}
}
