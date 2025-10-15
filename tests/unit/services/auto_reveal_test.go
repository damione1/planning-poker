package services_test

import (
	"testing"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/internal/services"
	"github.com/damione1/planning-poker/tests/helpers"
	"github.com/stretchr/testify/assert"
)

func TestHaveAllVotersVoted_NoVoters(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "custom", []string{"1", "2", "3"}, nil)

	// Add only spectators, no voters
	_, _ = rm.AddParticipant(room.Id, "Spectator1", models.RoleSpectator, "s1")
	_, _ = rm.AddParticipant(room.Id, "Spectator2", models.RoleSpectator, "s2")

	allVoted, err := rm.HaveAllVotersVoted(room.Id)
	assert.NoError(t, err)
	assert.False(t, allVoted, "Should return false when no voters exist")
}

func TestHaveAllVotersVoted_NoVotesCast(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "custom", []string{"1", "2", "3"}, nil)

	// Add voters but no votes cast
	_, _ = rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
	_, _ = rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")

	allVoted, err := rm.HaveAllVotersVoted(room.Id)
	assert.NoError(t, err)
	assert.False(t, allVoted, "Should return false when no votes are cast")
}

func TestHaveAllVotersVoted_PartialVotes(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "custom", []string{"1", "2", "3"}, nil)

	alice, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
	bob, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")
	_, _ = rm.AddParticipant(room.Id, "Charlie", models.RoleVoter, "s3")

	// Only 2 out of 3 voters cast votes
	_ = rm.CastVote(room.Id, alice.Id, "3")
	_ = rm.CastVote(room.Id, bob.Id, "5")

	allVoted, err := rm.HaveAllVotersVoted(room.Id)
	assert.NoError(t, err)
	assert.False(t, allVoted, "Should return false when not all voters have voted")
}

func TestHaveAllVotersVoted_AllVotesCast(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "custom", []string{"1", "2", "3"}, nil)

	alice, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
	bob, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")

	// Both voters cast votes
	_ = rm.CastVote(room.Id, alice.Id, "3")
	_ = rm.CastVote(room.Id, bob.Id, "5")

	allVoted, err := rm.HaveAllVotersVoted(room.Id)
	assert.NoError(t, err)
	assert.True(t, allVoted, "Should return true when all voters have voted")
}

func TestHaveAllVotersVoted_WithSpectators(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "custom", []string{"1", "2", "3"}, nil)

	alice, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
	bob, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")
	_, _ = rm.AddParticipant(room.Id, "Observer", models.RoleSpectator, "s3")

	// Both voters cast votes (spectator doesn't need to)
	_ = rm.CastVote(room.Id, alice.Id, "3")
	_ = rm.CastVote(room.Id, bob.Id, "5")

	allVoted, err := rm.HaveAllVotersVoted(room.Id)
	assert.NoError(t, err)
	assert.True(t, allVoted, "Should return true when all voters have voted, ignoring spectators")
}

func TestHaveAllVotersVoted_VoteUpdate(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "custom", []string{"1", "2", "3"}, nil)

	alice, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
	bob, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")

	// Cast initial votes
	_ = rm.CastVote(room.Id, alice.Id, "3")
	_ = rm.CastVote(room.Id, bob.Id, "5")

	allVoted, err := rm.HaveAllVotersVoted(room.Id)
	assert.NoError(t, err)
	assert.True(t, allVoted)

	// Update Alice's vote - should still count as all voted
	_ = rm.CastVote(room.Id, alice.Id, "8")

	allVoted, err = rm.HaveAllVotersVoted(room.Id)
	assert.NoError(t, err)
	assert.True(t, allVoted, "Should still return true after vote update")
}

func TestAutoRevealConfig_DefaultDisabled(t *testing.T) {
	config := models.DefaultRoomConfig()
	assert.False(t, config.Permissions.AutoReveal, "AutoReveal should be false by default")
}

func TestAutoRevealConfig_CanBeEnabled(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	aclService := services.NewACLService(rm)

	// Create room with auto-reveal enabled
	config := models.DefaultRoomConfig()
	config.Permissions.AutoReveal = true

	room, err := rm.CreateRoom("Test", "custom", []string{"1", "2", "3"}, config)
	assert.NoError(t, err)

	// Retrieve config and verify
	retrievedConfig, err := aclService.GetRoomConfig(room.Id)
	assert.NoError(t, err)
	assert.True(t, retrievedConfig.Permissions.AutoReveal, "AutoReveal should be enabled")
}

func TestAutoRevealConfig_UpdateConfig(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	aclService := services.NewACLService(rm)

	// Create room with default config
	room, _ := rm.CreateRoom("Test", "custom", []string{"1", "2", "3"}, nil)
	creator, _ := rm.AddParticipant(room.Id, "Creator", models.RoleVoter, "s1")

	// Set creator
	roomRecord, _ := rm.GetRoom(room.Id)
	roomRecord.Set("creator_participant_id", creator.Id)
	_ = server.App.Save(roomRecord)

	// Update config to enable auto-reveal
	newConfig := models.DefaultRoomConfig()
	newConfig.Permissions.AutoReveal = true

	err := aclService.UpdateRoomConfig(room.Id, creator.Id, newConfig)
	assert.NoError(t, err)

	// Verify config was updated
	retrievedConfig, err := aclService.GetRoomConfig(room.Id)
	assert.NoError(t, err)
	assert.True(t, retrievedConfig.Permissions.AutoReveal, "AutoReveal should be enabled after update")
}
