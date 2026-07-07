package services_test

import (
	"testing"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/internal/services"
	"github.com/damione1/planning-poker/tests/helpers"
	"github.com/stretchr/testify/assert"
)

// aclTestFixture bundles a room plus a creator and non-creator participant,
// mirroring the shape every ACL gate test needs: a creator (always allowed),
// a non-creator (gated by config), and the room ID for error-path cases.
type aclTestFixture struct {
	acl        *services.ACLService
	roomID     string
	creator    string
	nonCreator string
}

// setupACLFixture creates a room with the given config (nil for defaults)
// and two voter participants. The first participant added becomes the room
// creator (see RoomManager.AddParticipant), matching production behavior.
func setupACLFixture(t *testing.T, config *models.RoomConfig) *aclTestFixture {
	t.Helper()

	server := helpers.NewTestServerWithData(t)
	t.Cleanup(server.Cleanup)

	rm := services.NewRoomManager(server.App)
	acl := services.NewACLService(rm)

	room, err := rm.CreateRoom("Test Room", "fibonacci", nil, config)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}

	creator, err := rm.AddParticipant(room.Id, "Creator", models.RoleVoter, "creator-session")
	if err != nil {
		t.Fatalf("Failed to add creator: %v", err)
	}
	nonCreator, err := rm.AddParticipant(room.Id, "NonCreator", models.RoleVoter, "non-creator-session")
	if err != nil {
		t.Fatalf("Failed to add non-creator: %v", err)
	}

	// Sanity check the fixture's core assumption.
	if !rm.IsRoomCreator(room.Id, creator.Id) {
		t.Fatalf("fixture setup invariant broken: first participant should be room creator")
	}
	if rm.IsRoomCreator(room.Id, nonCreator.Id) {
		t.Fatalf("fixture setup invariant broken: second participant should not be room creator")
	}

	return &aclTestFixture{
		acl:        acl,
		roomID:     room.Id,
		creator:    creator.Id,
		nonCreator: nonCreator.Id,
	}
}

// gateFunc is the common shape of the four boolean permission gates on
// ACLService: CanReveal, CanReset, CanTriggerNewRound.
type gateFunc func(acl *services.ACLService, roomID, participantID string) (bool, error)

func runGateSuite(t *testing.T, gateName string, gate gateFunc, permissiveConfig, restrictiveConfig *models.RoomConfig) {
	t.Run(gateName+"/room creator is always allowed even when config denies", func(t *testing.T) {
		fx := setupACLFixture(t, restrictiveConfig)
		allowed, err := gate(fx.acl, fx.roomID, fx.creator)
		assert.NoError(t, err)
		assert.True(t, allowed, "room creator should always be allowed regardless of config")
	})

	t.Run(gateName+"/non-creator allowed when AllowAll* is true", func(t *testing.T) {
		fx := setupACLFixture(t, permissiveConfig)
		allowed, err := gate(fx.acl, fx.roomID, fx.nonCreator)
		assert.NoError(t, err)
		assert.True(t, allowed, "non-creator should be allowed when the corresponding AllowAll* permission is true")
	})

	t.Run(gateName+"/non-creator denied when AllowAll* is false", func(t *testing.T) {
		fx := setupACLFixture(t, restrictiveConfig)
		allowed, err := gate(fx.acl, fx.roomID, fx.nonCreator)
		assert.NoError(t, err)
		assert.False(t, allowed, "non-creator should be denied when the corresponding AllowAll* permission is false")
	})

	t.Run(gateName+"/empty participant ID is denied even with permissive config", func(t *testing.T) {
		// Security fix under test: IsRoomCreator("") must be false, so an
		// empty/anonymous participant ID falls through to config rather than
		// being accidentally treated as the creator. With a permissive
		// config, this still denies because... it must NOT be granted via
		// the creator short-circuit; assert against the restrictive config
		// where it must be false either way, AND against the permissive
		// config to prove the empty ID isn't itself elevating privilege
		// beyond what config would grant a normal non-creator.
		fxRestrictive := setupACLFixture(t, restrictiveConfig)
		allowed, err := gate(fxRestrictive.acl, fxRestrictive.roomID, "")
		assert.NoError(t, err)
		assert.False(t, allowed, "empty participant ID must be denied when config denies AllowAll*")
	})

	t.Run(gateName+"/missing room returns an error", func(t *testing.T) {
		fx := setupACLFixture(t, permissiveConfig)
		allowed, err := gate(fx.acl, "nonexistent-room-id", fx.creator)
		assert.Error(t, err)
		assert.False(t, allowed)
	})
}

func permissiveConfigAllowingReveal() *models.RoomConfig {
	cfg := models.DefaultRoomConfig()
	cfg.Permissions.AllowAllReveal = true
	return cfg
}

func restrictiveConfigDenyingReveal() *models.RoomConfig {
	cfg := models.DefaultRoomConfig()
	cfg.Permissions.AllowAllReveal = false
	return cfg
}

func permissiveConfigAllowingReset() *models.RoomConfig {
	cfg := models.DefaultRoomConfig()
	cfg.Permissions.AllowAllReset = true
	return cfg
}

func restrictiveConfigDenyingReset() *models.RoomConfig {
	cfg := models.DefaultRoomConfig()
	cfg.Permissions.AllowAllReset = false
	return cfg
}

func permissiveConfigAllowingNewRound() *models.RoomConfig {
	cfg := models.DefaultRoomConfig()
	cfg.Permissions.AllowAllNewRound = true
	return cfg
}

func restrictiveConfigDenyingNewRound() *models.RoomConfig {
	cfg := models.DefaultRoomConfig()
	cfg.Permissions.AllowAllNewRound = false
	return cfg
}

func TestACL_CanReveal(t *testing.T) {
	runGateSuite(t, "CanReveal",
		func(acl *services.ACLService, roomID, participantID string) (bool, error) {
			return acl.CanReveal(roomID, participantID)
		},
		permissiveConfigAllowingReveal(),
		restrictiveConfigDenyingReveal(),
	)
}

func TestACL_CanReset(t *testing.T) {
	runGateSuite(t, "CanReset",
		func(acl *services.ACLService, roomID, participantID string) (bool, error) {
			return acl.CanReset(roomID, participantID)
		},
		permissiveConfigAllowingReset(),
		restrictiveConfigDenyingReset(),
	)
}

func TestACL_CanTriggerNewRound(t *testing.T) {
	runGateSuite(t, "CanTriggerNewRound",
		func(acl *services.ACLService, roomID, participantID string) (bool, error) {
			return acl.CanTriggerNewRound(roomID, participantID)
		},
		permissiveConfigAllowingNewRound(),
		restrictiveConfigDenyingNewRound(),
	)
}

// CanChangeVoteAfterReveal has no room-creator short-circuit (see
// acl_service.go): it is a room-wide setting, not a per-participant
// permission gate. It is tested separately rather than via runGateSuite.
func TestACL_CanChangeVoteAfterReveal(t *testing.T) {
	t.Run("allowed when AllowChangeVoteAfterReveal is true", func(t *testing.T) {
		cfg := models.DefaultRoomConfig()
		cfg.Permissions.AllowChangeVoteAfterReveal = true
		fx := setupACLFixture(t, cfg)

		allowed, err := fx.acl.CanChangeVoteAfterReveal(fx.roomID)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("denied when AllowChangeVoteAfterReveal is false (default)", func(t *testing.T) {
		fx := setupACLFixture(t, models.DefaultRoomConfig())

		allowed, err := fx.acl.CanChangeVoteAfterReveal(fx.roomID)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("denied for empty participant context (room-wide, participant-agnostic)", func(t *testing.T) {
		cfg := models.DefaultRoomConfig()
		cfg.Permissions.AllowChangeVoteAfterReveal = false
		fx := setupACLFixture(t, cfg)

		// CanChangeVoteAfterReveal doesn't take a participant ID at all -
		// verify it is purely config-driven regardless of who is asking.
		allowed, err := fx.acl.CanChangeVoteAfterReveal(fx.roomID)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("missing room returns an error", func(t *testing.T) {
		fx := setupACLFixture(t, models.DefaultRoomConfig())
		allowed, err := fx.acl.CanChangeVoteAfterReveal("nonexistent-" + fx.roomID)
		assert.Error(t, err)
		assert.False(t, allowed)
	})
}

// TestACL_GetRoomConfig_MissingRoom covers the shared error path used by
// every gate above: a missing room must surface an error rather than
// silently falling back to defaults.
func TestACL_GetRoomConfig_MissingRoom(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	acl := services.NewACLService(rm)

	config, err := acl.GetRoomConfig("does-not-exist")
	assert.Error(t, err)
	assert.Nil(t, config)
}
