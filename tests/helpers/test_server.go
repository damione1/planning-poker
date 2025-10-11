package helpers

import (
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	// Import migrations to register them (directory path, not package name)
	_ "github.com/damione1/planning-poker/pb_migrations"
)

// TestServer wraps a PocketBase test instance
type TestServer struct {
	App core.App
	t   *testing.T
}

// NewTestServer creates a new test PocketBase instance with in-memory database
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Create temporary directory for test database
	testDir := t.TempDir()

	// Create test app with temporary directory
	app, err := tests.NewTestApp(testDir)
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	// Bootstrap app (runs migrations)
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap test app: %v", err)
	}

	return &TestServer{
		App: app,
		t:   t,
	}
}

// NewTestServerWithData creates a test server and applies migrations from the project
func NewTestServerWithData(t *testing.T) *TestServer {
	t.Helper()

	testDir := t.TempDir()

	// Create PocketBase app with test directory
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:   testDir,
		DataMaxOpenConns: 1,
		DataMaxIdleConns: 1,
	})

	// Bootstrap the app to initialize the database
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap app: %v", err)
	}

	// Run all registered migrations
	// (migrations are registered via init() functions when the package is imported)
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return &TestServer{
		App: app,
		t:   t,
	}
}

// Cleanup closes the test server and removes temporary files
func (ts *TestServer) Cleanup() {
	if app, ok := ts.App.(*pocketbase.PocketBase); ok {
		_ = app.ResetBootstrapState() // Best effort cleanup
	}
}
