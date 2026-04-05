# Implementation Reminders

> **Purpose:** Track features and design decisions that need to be implemented or revisited later, with specific context and requirements.

**Last Updated:** April 3, 2026

---

## Q69 — Notification & Alert System (UI/UX Design Pending)

**Status:** ⏳ Deferred — Awaiting Figma UI Design

**Decision Made:** Hybrid approach (Toast + Panel + Modal)

**What's needed:**
- Toast notifications: Quick confirmations (auto-disappear after 3s)
- Notification panel: Persistent list (bell icon in top bar)
- Modal dialogs: Requires user action/decision
- Different notification types for different urgency levels

**Context:**
- Processing complete: "Resource added to AI"
- Classification needed: "Please categorize: [Option 1] [Option 2] [Skip]"
- Archive alerts: "X resources archived today"
- Error messages: "Failed to extract content from PDF"

**Implementation Notes:**
- Build notification system as a **standalone service/hook** (not tightly coupled to any specific component)
- Design should follow **React context + hook pattern** for easy integration across app
- Use Zustand store for persistent notification state
- Notification types: `success`, `error`, `warning`, `info`, `action_required`

**To be confirmed from Figma:**
- Visual style (colors, animations, positioning)
- Toast duration for different types
- Maximum notifications in panel
- Sound/vibration options

**Follow-up Tasks:**
- [ ] Get Figma design from user
- [ ] Implement notification service in Go backend
- [ ] Create React notification provider component
- [ ] Test with various notification types

---

## Modularity & Loose Coupling Principles

**User Requirement (April 3, 2026):**
> "All these development things should be modular and loosely coupled because when I decided to remove a feature then it could be removed very easily without making much changes or disturbance to other section of the code/program"

**Implementation Strategy:**

### Backend (Go)

**Architecture Pattern: Repository + Service + Domain Interfaces**

```go
// domain/interfaces.go — Feature-agnostic interfaces
type ResourceRepository interface {
    Create(ctx context.Context, r *domain.Resource) error
    FindByID(ctx context.Context, id string) (*domain.Resource, error)
    // ... other methods
}

type NotificationService interface {
    Send(ctx context.Context, n *domain.Notification) error
    GetByUser(ctx context.Context, userID string) ([]*domain.Notification, error)
}

// If we remove notifications later:
// - Just remove the service implementation
// - Leave interfaces in domain/
// - No changes to business logic
```

**Feature Flags:**
```go
// config/features.go
type Features struct {
    EnableNotifications      bool
    EnableBehavioralModel    bool
    EnableProactiveDiscovery bool
    EnableReminders          bool
}

// In main route setup:
if config.Features.EnableNotifications {
    setupNotificationRoutes(router)
}
```

### Frontend (React + Zustand)

**Modular Store Pattern:**

```typescript
// stores/notifications.ts
export const useNotificationStore = create((set) => ({
    notifications: [],
    addNotification: (notif) => { /* ... */ },
    removeNotification: (id) => { /* ... */ },
}));

// If removing notifications:
// - Delete stores/notifications.ts
// - Remove useNotificationStore from components
// - No changes to core app structure
```

**Feature-Specific Hooks:**

```typescript
// hooks/useNotifications.ts
export function useNotifications() {
    // Notification logic isolated here
    // Easy to remove entirely
}

// Usage in components:
function ResourceCard() {
    const { showNotification } = useNotifications(); // Optional hook
    // Component still works if hook is removed
}
```

### Why This Matters

| Tight Coupling | Loose Coupling |
|---|---|
| Remove notifications → 50+ files break | Remove notifications → Delete 3-4 files |
| Feature buried in business logic | Feature in dedicated module |
| Hard to test | Easy to test in isolation |
| Difficult refactoring | Safe refactoring |

**Key Principle:** Each feature should be removable by deleting its files + removing feature flag

---

*This file will be updated as more deferred features are added during development.*
