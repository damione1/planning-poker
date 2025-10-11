package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
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
		DataDir:       testDir,
		DataMaxOpenConns: 1,
		DataMaxIdleConns: 1,
	})

	// Copy migrations from project to test directory
	projectRoot := filepath.Join("..", "..")
	migrationsDir := filepath.Join(projectRoot, "pb_migrations")
	testMigrationsDir := filepath.Join(testDir, "pb_migrations")

	if err := os.MkdirAll(testMigrationsDir, 0755); err != nil {
		t.Fatalf("Failed to create migrations dir: %v", err)
	}

	// Copy migration files
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("Failed to read migrations: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			srcPath := filepath.Join(migrationsDir, entry.Name())
			dstPath := filepath.Join(testMigrationsDir, entry.Name())

			data, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("Failed to read migration %s: %v", entry.Name(), err)
			}

			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				t.Fatalf("Failed to write migration %s: %v", entry.Name(), err)
			}
		}
	}

	// Bootstrap the app (runs migrations)
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap app: %v", err)
	}

	return &TestServer{
		App: app,
		t:   t,
	}
}

// Cleanup closes the test server and removes temporary files
func (ts *TestServer) Cleanup() {
	if app, ok := ts.App.(*pocketbase.PocketBase); ok {
		app.ResetBootstrapState()
	}
}
