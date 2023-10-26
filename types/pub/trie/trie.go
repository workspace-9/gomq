package trie

import (
	"strings"
)

// Trie represents a trie data structure.
type Trie[T comparable] struct {
	Key      string
	Children map[string]*Trie[T]
	Values   map[T]struct{}
}

// Put the value in the trie at the given key.
func (t *Trie[T]) Put(key string, value T) {
	if key == t.Key {
		if t.Values == nil {
			t.Values = make(map[T]struct{})
		}
		t.Values[value] = struct{}{}
		return
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
		} else if len(key) >= len(childKey) {
			if strings.HasPrefix(key, childKey) {
				child.Put(key, value)
				return
			}
		}
	}

	if t.Children == nil {
		t.Children = make(map[string]*Trie[T])
	}

	if newChildren != nil {
		t.Children[key] = &Trie[T]{
			Key: key, Children: newChildren, Values: map[T]struct{}{value: {}},
		}
	} else {
		t.Children[key] = &Trie[T]{
			Key: key, Values: map[T]struct{}{value: {}},
		}
	}
}

func (t *Trie[T]) Remove(key string, value T) bool {
	return t.removeParent(key, value, nil)
}

func (t *Trie[T]) removeParent(key string, value T, parent *Trie[T]) bool {
	if key == t.Key {
		_, ok := t.Values[value]
		if ok {
			delete(t.Values, value)
		}

		if len(t.Values) == 0 {
			delete(parent.Children, t.Key)
			for _, child := range t.Children {
				parent.Children[child.Key] = child
			}
		}

		return ok
	}

	for childKey, child := range t.Children {
		if strings.HasPrefix(key, childKey) {
			return child.removeParent(key, value, t)
		}
	}

	return false
}

func (t *Trie[T]) Query(query string, visitFunc func(key string, value T)) {
	if !strings.HasPrefix(query, t.Key) {
		return
	}

	for value := range t.Values {
		visitFunc(t.Key, value)
	}

	for _, child := range t.Children {
		child.Query(query, visitFunc)
	}
}
