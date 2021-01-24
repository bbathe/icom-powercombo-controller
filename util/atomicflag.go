package util

import "sync/atomic"

type AtomFlag struct {
	flag int32
}

func (b *AtomFlag) Set(value bool) {
	var i int32 = 0
	if value {
		i = 1
	}
	atomic.StoreInt32(&(b.flag), int32(i))
}

func (b *AtomFlag) Get() bool {
	return atomic.LoadInt32(&(b.flag)) != 0
}

func (b *AtomFlag) IsTrue() bool {
	return atomic.LoadInt32(&(b.flag)) != 0
}

func (b *AtomFlag) IsFalse() bool {
	return atomic.LoadInt32(&(b.flag)) == 0
}
