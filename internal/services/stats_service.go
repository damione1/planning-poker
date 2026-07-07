package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// StatsService accumulates lightweight usage counters in memory and
// periodically flushes them into a single upserted stats_daily row per
// calendar day. It intentionally has no relation to rooms/participants/votes
// (see the stats_daily migration), so cleanupExpiredRooms can never touch it.
type StatsService struct {
	app core.App

	mu        sync.Mutex
	pending   map[string]int64
	peakConns int64
	peakRooms int64
}

// NewStatsService creates a StatsService bound to the given app.
func NewStatsService(app core.App) *StatsService {
	return &StatsService{
		app:     app,
		pending: make(map[string]int64),
	}
}

// Inc accumulates a delta for the given counter field, to be applied on the
// next Flush.
func (s *StatsService) Inc(field string, n int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[field] += n
}

// ObserveGauges records the current connection/room counts, updating the
// running daily peak watermarks if they exceed the previously observed peaks.
func (s *StatsService) ObserveGauges(conns int64, rooms int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if conns > s.peakConns {
		s.peakConns = conns
	}
	if int64(rooms) > s.peakRooms {
		s.peakRooms = int64(rooms)
	}
}

// Flush upserts today's stats_daily row: pending counter deltas are added to
// the stored values, and peak gauges are raised to the max of the stored and
// observed values. It performs at most one DB write per call regardless of
// how much traffic accumulated in `pending`.
func (s *StatsService) Flush() error {
	s.mu.Lock()
	pending := s.pending
	s.pending = make(map[string]int64)
	peakConns := s.peakConns
	peakRooms := s.peakRooms
	// Reset the in-memory peaks after capturing them. The stored row already
	// retains the running daily maximum (via the max() on save below), so the
	// in-memory watermark only needs to represent activity since the last
	// flush. Without this reset the watermark would be a process-lifetime max
	// and bleed a busy day's peak into every subsequent, quieter day's row.
	s.peakConns = 0
	s.peakRooms = 0
	s.mu.Unlock()

	dayKey := time.Now().Format("2006-01-02")

	record, err := s.app.FindFirstRecordByFilter("stats_daily", "day = {:d}", map[string]any{"d": dayKey})
	if err != nil {
		collection, err := s.app.FindCollectionByNameOrId("stats_daily")
		if err != nil {
			return fmt.Errorf("failed to find stats_daily collection: %w", err)
		}
		record = core.NewRecord(collection)
		record.Set("day", dayKey)
	}

	for field, delta := range pending {
		record.Set(field, record.GetInt(field)+int(delta))
	}

	if peakConns > int64(record.GetInt("peak_connections")) {
		record.Set("peak_connections", peakConns)
	}
	if peakRooms > int64(record.GetInt("peak_rooms")) {
		record.Set("peak_rooms", peakRooms)
	}

	if err := s.app.Save(record); err != nil {
		return fmt.Errorf("failed to save stats_daily record: %w", err)
	}

	return nil
}
