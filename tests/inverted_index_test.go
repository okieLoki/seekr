package tests

import (
	"reflect"
	"testing"
	"seekr/index"
)

func TestInvertedIndex(t *testing.T) {
	idx := index.New()

	if docs := idx.Get("hello"); len(docs) != 0 {
		t.Errorf("expected empty map, got %v", docs)
	}

	idx.Add("hello", 1)
	idx.Add("hello", 2)
	idx.Add("world", 1)
	idx.Add("hello", 1) // duplicate increases TF

	docs := idx.Get("hello")
	expectedHello := map[int]int{1: 2, 2: 1}
	if !reflect.DeepEqual(docs, expectedHello) {
		t.Errorf("expected %v, got %v", expectedHello, docs)
	}

	docsWorld := idx.Get("world")
	expectedWorld := map[int]int{1: 1}
	if !reflect.DeepEqual(docsWorld, expectedWorld) {
		t.Errorf("expected %v, got %v", expectedWorld, docsWorld)
	}
}
