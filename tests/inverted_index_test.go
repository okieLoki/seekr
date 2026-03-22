package tests

import (
	"reflect"
	"seekr/index"
	"testing"
)

func TestInvertedIndex(t *testing.T) {
	idx := index.New()

	if docs := idx.Get("hello"); len(docs) != 0 {
		t.Errorf("expected empty slice, got %v", docs)
	}

	idx.Add("hello", 1)
	idx.Add("world", 1)
	idx.Add("hello", 2)

	if docs := idx.Get("hello"); !reflect.DeepEqual(docs, []int{1, 2}) {
		t.Errorf("expected [1, 2], got %v", docs)
	}

	if docs := idx.Get("world"); !reflect.DeepEqual(docs, []int{1}) {
		t.Errorf("expected [1], got %v", docs)
	}

	idx.Add("hello", 1)
	if docs := idx.Get("hello"); !reflect.DeepEqual(docs, []int{1, 2}) {
		t.Errorf("expected [1, 2] after duplicate add, got %v", docs)
	}
}
