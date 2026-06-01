package lacrima

import "unsafe"

const (
	defaultHashMB = 16
	minHashMB     = 1
	maxHashMB     = 33554432
)

type TTFlag uint8

const (
	TTExact TTFlag = iota
	TTLowerBound
	TTUpperBound
)

type TTEntry struct {
	Key   uint64
	Flag  TTFlag
	Depth int
	Score int
	Move  Move
	Valid bool
}

type TranspositionTable struct {
	entries []TTEntry
}

func NewTranspositionTable(size uint64) *TranspositionTable {
	if size == 0 {
		size = 1
	}

	return &TranspositionTable{
		entries: make([]TTEntry, size),
	}
}

func (tt *TranspositionTable) Clear() {
	if tt == nil {
		return
	}

	clear(tt.entries)
}

func (tt *TranspositionTable) Probe(key uint64) (TTEntry, bool) {
	if tt == nil || len(tt.entries) == 0 {
		return TTEntry{}, false
	}

	entry := tt.entries[tt.index(key)]
	return entry, entry.Valid && entry.Key == key
}

func (tt *TranspositionTable) Store(key uint64, depth int, score int, flag TTFlag, move Move) {
	if tt == nil || len(tt.entries) == 0 {
		return
	}

	index := tt.index(key)
	entry := tt.entries[index]
	if entry.Key != key || depth >= entry.Depth {
		tt.entries[index] = TTEntry{
			Key:   key,
			Flag:  flag,
			Depth: depth,
			Score: score,
			Move:  move,
			Valid: true,
		}
	}
}

func (tt *TranspositionTable) index(key uint64) uint64 {
	return key % uint64(len(tt.entries))
}

func transpositionTableEntriesForMB(hashMB int) uint64 {
	bytes := uint64(hashMB) * 1024 * 1024
	entrySize := uint64(unsafe.Sizeof(TTEntry{}))
	if bytes < entrySize {
		return 1
	}
	return bytes / entrySize
}
