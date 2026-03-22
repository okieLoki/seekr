package tests

import (
	"reflect"
	"seekr/index"
	"testing"
)

func TestInvertedIndex_GetEmpty(t *testing.T) {
	idx := index.New()
	if docs := idx.Get("hello"); len(docs) != 0 {
		t.Errorf("expected empty map, got %v", docs)
	}
}

func TestInvertedIndex_AddAndGet(t *testing.T) {
	idx := index.New()
	idx.Add("hello", 1)
	idx.Add("hello", 2)
	idx.Add("world", 1)
	idx.Add("hello", 1)

	expectedHello := map[int]int{1: 2, 2: 1}
	if docs := idx.Get("hello"); !reflect.DeepEqual(docs, expectedHello) {
		t.Errorf("expected %v, got %v", expectedHello, docs)
	}

	expectedWorld := map[int]int{1: 1}
	if docs := idx.Get("world"); !reflect.DeepEqual(docs, expectedWorld) {
		t.Errorf("expected %v, got %v", expectedWorld, docs)
	}
}
