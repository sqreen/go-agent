// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package actor

import (
	iradix "github.com/hashicorp/go-immutable-radix"
	"github.com/sqreen/go-agent/internal/sqlib/squnsafe"
)

type PathListStore iradix.Tree

func NewPathListStore(paths []string) *PathListStore {
	if len(paths) == 0 {
		return nil
	}

	txn := iradix.New().Txn()
	for _, path := range paths {
		txn.Insert(squnsafe.StringToBytes(path), struct{}{})
	}

	return (*PathListStore)(txn.Commit())
}

func (s *PathListStore) unwrap() *iradix.Tree { return (*iradix.Tree)(s) }

func (s *PathListStore) Find(path string) (exists bool) {
	_, exists = s.unwrap().Get(squnsafe.StringToBytes(path))
	return
}
