package trie

import (
	"strings"
)

// Trie represents a trie data structure.
type Trie[T comparable] struct {
	Key      string
	Children map[string]*Trie[T]
	Values   []T
}

// Put the value in the trie at the given key.
func (t *Trie[T]) Put(key string, value T) {
	if key == t.Key {
		t.Values = append(t.Values, value)
	}

	var newChildren map[string]*Trie[T]
	for childKey, child := range t.Children {
		if len(childKey) > len(key) {
			if strings.HasPrefix(childKey, key) {
				if newChildren == nil {
					newChildren = make(map[string]*Trie[T])
				}
				newChildren[childKey] = child
				delete(t.Children, childKey)
			}
		} else if len(key) <= len(childKey) {
			if strings.HasPrefix(key, childKey) {
				child.Put(key, value)
				return
			}
		}
	}

	if newChildren != nil {
		t.Children[key] = &Trie[T]{
			Key: key, Children: newChildren, Values: []T{value},
		}
	} else {
		t.Children[key] = &Trie[T]{
			Key: key, Children: make(map[string]*Trie[T]), Values: []T{value},
		}
	}
}

func (t *Trie[T]) Search(prefix string, visitFunc func(key string, value T)) {
	for _, val := range t.Values {
		visitFunc(t.Key, val)
	}

	for childKey, child := range t.Children {
		if strings.HasPrefix(childKey, prefix) {
			child.Search(prefix, visitFunc)
		}
	}
}
