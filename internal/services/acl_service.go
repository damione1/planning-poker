package services

import (
	"encoding/json"
	"fmt"

	"github.com/damione1/planning-poker-new/internal/models"
)

// ACLService handles permission checks for room actions
type ACLService struct {
	roomManager *RoomManager
}

func NewACLService(rm *RoomManager) *ACLService {
	return &ACLService{
		roomManager: rm,
	}
}

// GetRoomConfig retrieves and parses room configuration
func (acl *ACLService) GetRoomConfig(roomID string) (*models.RoomConfig, error) {
	room, err := acl.roomManager.GetRoom(roomID)
	if err != nil {
		return nil, fmt.Errorf("room not found: %w", err)
	}

	configJSON := room.GetString("config")

	// If no config exists, return default
	if configJSON == "" {
		return models.DefaultRoomConfig(), nil
	}

	var config models.RoomConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		// If parsing fails, return default config
		return models.DefaultRoomConfig(), nil
	}

	return &config, nil
}

// CanTriggerNewRound checks if participant can create a new round
func (acl *ACLService) CanTriggerNewRound(roomID, participantID string) (bool, error) {
	// Always allow room creator
	if acl.roomManager.IsRoomCreator(roomID, participantID) {
		return true, nil
	}

	config, err := acl.GetRoomConfig(roomID)
	if err != nil {
		return false, err
	}

	return config.Permissions.AllowAllNewRound, nil
}

// CanReset checks if participant can reset the round
func (acl *ACLService) CanReset(roomID, participantID string) (bool, error) {
	// Always allow room creator
	if acl.roomManager.IsRoomCreator(roomID, participantID) {
		return true, nil
	}

	config, err := acl.GetRoomConfig(roomID)
	if err != nil {
		return false, err
	}

	return config.Permissions.AllowAllReset, nil
}

// CanReveal checks if participant can reveal votes
func (acl *ACLService) CanReveal(roomID, participantID string) (bool, error) {
	// Always allow room creator
	if acl.roomManager.IsRoomCreator(roomID, participantID) {
		return true, nil
	}

	config, err := acl.GetRoomConfig(roomID)
	if err != nil {
		return false, err
	}

	return config.Permissions.AllowAllReveal, nil
}

// CanChangeVoteAfterReveal checks if participants can change votes after reveal
func (acl *ACLService) CanChangeVoteAfterReveal(roomID string) (bool, error) {
	config, err := acl.GetRoomConfig(roomID)
	if err != nil {
		return false, err
	}

	return config.Permissions.AllowChangeVoteAfterReveal, nil
}

// UpdateRoomConfig updates room configuration (creator only)
func (acl *ACLService) UpdateRoomConfig(roomID, participantID string, config *models.RoomConfig) error {
	// Only room creator can update config
	if !acl.roomManager.IsRoomCreator(roomID, participantID) {
		return fmt.Errorf("unauthorized: only room creator can update config")
	}

	room, err := acl.roomManager.GetRoom(roomID)
	if err != nil {
		return err
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	room.Set("config", string(configJSON))
	if err := acl.roomManager.app.Save(room); err != nil {
		return fmt.Errorf("failed to save room config: %w", err)
	}

	return nil
}
