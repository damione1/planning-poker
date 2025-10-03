package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/damiengoehrig/planning-poker/internal/models"
)

type RoomManager struct {
	rooms map[string]*models.Room
	mu    sync.RWMutex
}

func NewRoomManager() *RoomManager {
	rm := &RoomManager{
		rooms: make(map[string]*models.Room),
	}

	// Start cleanup goroutine
	go rm.cleanupLoop()

	return rm
}

func (rm *RoomManager) CreateRoom(id, name, pointingMethod string, customValues []string) *models.Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room := models.NewRoom(id, name, pointingMethod, customValues)
	rm.rooms[id] = room
	return room
}

func (rm *RoomManager) GetRoom(id string) (*models.Room, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, exists := rm.rooms[id]
	if !exists {
		return nil, fmt.Errorf("room not found")
	}
	return room, nil
}

func (rm *RoomManager) DeleteRoom(id string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rooms, id)
}

func (rm *RoomManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		rm.cleanupInactiveRooms()
	}
}

func (rm *RoomManager) cleanupInactiveRooms() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)

	for id, room := range rm.rooms {
		lastActivity := room.GetLastActivity()

		if lastActivity.Before(cutoff) {
			delete(rm.rooms, id)
		}
	}
}
