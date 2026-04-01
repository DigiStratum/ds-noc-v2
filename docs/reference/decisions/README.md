# Architecture Decision Records

> Document significant architectural decisions and their rationale.

---

## ADR Format

Each decision follows this format:

```markdown
## ADR-NNN: Title

**Status:** Proposed | Accepted | Deprecated | Superseded
**Date:** YYYY-MM-DD
**Deciders:** Names

### Context
What is the issue that we're seeing that is motivating this decision?

### Decision
What is the change that we're proposing and/or doing?

### Consequences
What becomes easier or more difficult to do because of this change?
```

---

## Decisions

### ADR-001: Two-Layer Architecture

**Status:** Accepted  
**Date:** 2026-03-01  
**Deciders:** skelly, lucca

#### Context
Apps built from templates diverge over time, making updates painful. Changes to shared infrastructure require manual merges across all apps.

#### Decision
Implement manifest-based two-layer architecture:
- Template layer: tracked in `.template-manifest`, replaceable on update
- App layer: outside manifest, preserved across updates

#### Consequences
- ✅ Template updates are automated and non-destructive
- ✅ Clear separation of concerns
- ❌ Requires discipline to not modify template files
- ❌ Some customization requires upstream template changes

---

### ADR-002: HAL/HATEOAS APIs

**Status:** Accepted  
**Date:** 2026-03-01  
**Deciders:** skelly, lucca

#### Context
API consumers (including AI agents) need to discover available operations without hardcoding URLs.

#### Decision
All API responses follow HAL format with `_links` for related resources and actions.

#### Consequences
- ✅ Self-documenting APIs
- ✅ Clients can navigate without URL knowledge
- ❌ Slightly larger response payloads
- ❌ Requires consistent implementation across handlers

---

### ADR-003: SSO-First Authentication

**Status:** Accepted  
**Date:** 2026-03-01  
**Deciders:** skelly

#### Context
Each app managing its own authentication creates security risks and user friction.

#### Decision
All DS ecosystem apps delegate authentication to DSAccount. Apps validate session cookies but never handle credentials.

#### Consequences
- ✅ Single sign-on across ecosystem
- ✅ Centralized security controls
- ❌ DSAccount is a critical dependency
- ❌ Apps cannot work without DSAccount available

---

*Add new ADRs above this line*
