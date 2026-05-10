# Session Invalidation

SystemForge provides a session invalidation package for tracking user sessions across devices and enabling features like "logout all devices" and session management.

## Features

- Track sessions with device info, IP address, and metadata
- Invalidate single sessions or all sessions for a user
- "Logout all other devices" functionality
- Automatic expired session cleanup
- Maximum sessions per user enforcement
- Memory and Redis storage backends

## Quick Start

### Basic Usage

```go
import (
    "github.com/grokify/systemforge/session/invalidation"
)

// Create a memory store
store := invalidation.NewMemoryStore()

// Create the session manager
manager := invalidation.NewManager(store,
    invalidation.WithSessionTTL(24*time.Hour),
    invalidation.WithMaxSessionsPerUser(5),
)
defer manager.Close()

// Create a session for a user
session, err := manager.CreateSession(ctx, userID,
    invalidation.WithDeviceID("device-123"),
    invalidation.WithDeviceInfo("Chrome on macOS"),
    invalidation.WithIPAddress("192.168.1.100"),
)

// Validate a session (also updates LastActiveAt)
session, err := manager.ValidateSession(ctx, sessionID)
if err != nil {
    if errors.Is(err, invalidation.ErrSessionExpired) {
        // Session has expired
    }
    if errors.Is(err, invalidation.ErrSessionInvalid) {
        // Session was invalidated
    }
}

// List all active sessions for a user
sessions, err := manager.ListSessions(ctx, userID)

// Invalidate a specific session (logout one device)
err := manager.InvalidateSession(ctx, sessionID)

// Invalidate all sessions (logout all devices)
count, err := manager.InvalidateAllSessions(ctx, userID)

// Invalidate all except current (logout other devices)
count, err := manager.InvalidateOtherSessions(ctx, userID, currentSessionID)
```

## Session Structure

```go
type Session struct {
    ID           string            // Unique session identifier
    UserID       string            // User who owns this session
    DeviceID     string            // Optional device identifier
    DeviceInfo   string            // Device information (user agent, etc.)
    IPAddress    string            // Client IP address
    CreatedAt    time.Time         // When session was created
    LastActiveAt time.Time         // When session was last used
    ExpiresAt    time.Time         // When session expires
    Metadata     map[string]string // Additional session data
}
```

## Configuration

### Manager Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithSessionTTL(duration)` | Default session lifetime | 24 hours |
| `WithMaxSessionsPerUser(n)` | Max sessions per user (0 = unlimited) | 0 |
| `WithConfig(cfg)` | Full configuration struct | See below |

### Config Struct

```go
cfg := invalidation.Config{
    SessionTTL:         24 * time.Hour,  // Session lifetime
    CleanupInterval:    time.Hour,       // How often to clean expired sessions
    MaxSessionsPerUser: 5,               // 0 = unlimited
}

manager := invalidation.NewManager(store, invalidation.WithConfig(cfg))
```

## Storage Backends

### Memory Store

For development and single-instance deployments:

```go
store := invalidation.NewMemoryStore()
```

### Redis Store

For production and multi-instance deployments:

```go
import "github.com/redis/go-redis/v9"

client := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

store := invalidation.NewRedisStore(client,
    invalidation.WithKeyPrefix("myapp:sessions:"),
)
```

#### Redis Store Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithKeyPrefix(prefix)` | Redis key prefix | `"sessions:"` |

## Session Options

When creating sessions, you can set additional metadata:

```go
session, err := manager.CreateSession(ctx, userID,
    invalidation.WithDeviceID("device-uuid"),
    invalidation.WithDeviceInfo("Mozilla/5.0 (Macintosh; Intel Mac OS X...)"),
    invalidation.WithIPAddress("192.168.1.100"),
    invalidation.WithTTL(48*time.Hour),  // Custom TTL for this session
    invalidation.WithMetadata("provider", "google"),
    invalidation.WithMetadata("app_version", "2.1.0"),
)
```

## Common Patterns

### Integration with JWT

Pair session invalidation with JWT tokens for stateless authentication with revocation:

```go
// On login: create session and issue JWT with session ID
session, _ := manager.CreateSession(ctx, userID)
claims := jwt.Claims{
    SessionID: session.ID,
    // ... other claims
}
token := jwtService.GenerateToken(claims)

// On each request: validate JWT, then validate session
claims, err := jwtService.ValidateToken(token)
if err != nil {
    return unauthorized
}
session, err := manager.ValidateSession(ctx, claims.SessionID)
if err != nil {
    return unauthorized  // Session was invalidated or expired
}

// On logout: invalidate session (JWT becomes unusable)
manager.InvalidateSession(ctx, sessionID)
```

### Session Management UI

Display active sessions to users:

```go
sessions, _ := manager.ListSessions(ctx, userID)

for _, s := range sessions {
    fmt.Printf("Device: %s\n", s.DeviceInfo)
    fmt.Printf("IP: %s\n", s.IPAddress)
    fmt.Printf("Last active: %s\n", s.LastActiveAt.Format(time.RFC3339))
    fmt.Printf("Session ID: %s\n", s.ID)
}
```

### Password Change: Logout All Devices

After a password change, invalidate all other sessions:

```go
// After password change, keep current session but logout others
count, err := manager.InvalidateOtherSessions(ctx, userID, currentSessionID)
log.Printf("Logged out %d other sessions", count)
```

## Errors

| Error | Description |
|-------|-------------|
| `ErrSessionNotFound` | Session does not exist |
| `ErrSessionExpired` | Session has expired |
| `ErrSessionInvalid` | Session was invalidated |
| `ErrStorageFailure` | Storage backend error |
| `ErrInvalidSessionID` | Invalid session ID format |
