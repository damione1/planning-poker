# Planning Poker Test Suite

Comprehensive test suite for the Planning Poker WebSocket-based real-time application.

## Overview

This test suite follows a pyramid approach focusing on:
- **Unit Tests (75%)**: Fast, isolated tests for models, services, and security components
- **Integration Tests (20%)**: Database-backed service tests (planned)
- **E2E Tests (5%)**: Full system tests with browser automation (planned)

## Current Implementation Status

### ✅ Phase 1: Foundation (Completed)

**Test Infrastructure**:
- `tests/helpers/test_server.go` - PocketBase test app initialization
- `tests/helpers/mock_ws.go` - Mock WebSocket connection for testing
- `tests/helpers/fixtures.go` - Test data creation helpers

**Unit Tests**:
- ✅ **Models**: Room, Participant (20 tests)
- ✅ **Services**: VoteValidator, Hub (60+ tests)
- ✅ **Security**: Validators, XSS prevention, injection protection (45+ tests)

**Total**: 125+ passing tests with comprehensive coverage of core business logic

### 🔄 Phase 2: Integration Tests (Planned)

- Room lifecycle (create, update, expire)
- Participant management
- Voting flows
- Round transitions
- ACL service integration

### 🔄 Phase 3: WebSocket Tests (Planned)

- Connection handling
- Message routing
- Broadcast patterns
- Reconnection scenarios

### 🔄 Phase 4: Multi-User Tests (Planned)

- Concurrent voting
- State synchronization
- Race condition detection

### 🔄 Phase 5: E2E & Security (Planned)

- Browser-based user journeys
- Performance testing
- Security vulnerability scanning

## Running Tests

### All Tests
```bash
go test ./tests/...
```

### Unit Tests Only
```bash
go test ./tests/unit/...
```

### Specific Package
```bash
go test ./tests/unit/models -v
go test ./tests/unit/services -v
go test ./tests/unit/security -v
```

### With Coverage
```bash
go test ./tests/... -cover
```

### Generate Coverage Report
```bash
go test ./tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### With Race Detection
```bash
go test ./tests/... -race
```

## Test Organization

```
tests/
├── unit/
│   ├── models/         # Model struct tests
│   ├── services/       # Business logic tests
│   └── security/       # Security validation tests
├── integration/        # Database-backed tests (planned)
├── helpers/            # Test utilities and fixtures
└── README.md          # This file
```

## Test Coverage

### Models
- ✅ Room creation and state management
- ✅ Participant roles and lifecycle
- ✅ State transitions and fallback behavior

### Services
- ✅ VoteValidator: Custom value parsing, validation, templates
- ✅ Hub: Initialization and basic operations
- ⏳ RoomManager: (Integration tests planned)
- ⏳ ACLService: (Integration tests planned)

### Security
- ✅ Input validation (names, IDs, vote values)
- ✅ XSS prevention (script tags, event handlers)
- ✅ Injection prevention (SQL, null bytes, control chars)
- ✅ Length constraints and character restrictions
- ✅ Error message sanitization

## Key Testing Patterns

### Model Tests
```go
func TestNewParticipant(t *testing.T) {
    p := models.NewParticipant("p-1", "Alice", models.RoleVoter)

    assert.Equal(t, "p-1", p.ID)
    assert.Equal(t, "Alice", p.Name)
    assert.Equal(t, models.RoleVoter, p.Role)
}
```

### Service Tests
```go
func TestVoteValidator_ParseCustomValues(t *testing.T) {
    v := services.NewVoteValidator()

    values, err := v.ParseCustomValues("XS, S, M, L, XL")

    assert.NoError(t, err)
    assert.Equal(t, []string{"XS", "S", "M", "L", "XL"}, values)
}
```

### Security Tests
```go
func TestValidateParticipantName(t *testing.T) {
    _, err := security.ValidateParticipantName("<script>alert('xss')</script>")

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid characters")
}
```

## Dependencies

- **Testing Framework**: Go standard `testing` package
- **Assertions**: `github.com/stretchr/testify/assert`
- **Database**: PocketBase `tests.NewTestApp()` for integration tests
- **WebSocket**: Mock implementation in `helpers/mock_ws.go`

## Best Practices

1. **Test Isolation**: Each test runs independently with fresh state
2. **Table-Driven Tests**: Used for testing multiple input scenarios
3. **Descriptive Names**: Tests clearly describe what they verify
4. **Fast Execution**: Unit tests complete in <1 second
5. **Helper Functions**: Common setup extracted to `helpers/fixtures.go`

## CI/CD Integration (Planned)

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: "1.24"
      - run: go test ./tests/... -race -cover
```

## Contributing

When adding new tests:

1. **Follow Existing Patterns**: Match the style of existing tests
2. **Use Table-Driven Tests**: For multiple input scenarios
3. **Test Edge Cases**: Include boundary conditions and error cases
4. **Keep Tests Fast**: Unit tests should execute in milliseconds
5. **Use Helpers**: Leverage `tests/helpers/` for common operations
6. **Document Complex Tests**: Add comments for non-obvious test logic

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://pkg.go.dev/github.com/stretchr/testify)
- [PocketBase Testing](https://pocketbase.io/docs/testing/)
- [Test Suite Design](../claudedocs/TEST_SUITE_DESIGN.md)
