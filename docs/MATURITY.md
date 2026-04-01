---
version: 1
levels:
  0:
    name: Prototype
    coverage: {}
    files: []
    checks: []
  1:
    name: Development
    coverage:
      backend: 40
      frontend: 30
    files:
      - .github/workflows/ci.yml
      - README.md
    checks:
      - type: command
        name: backend_builds
        command: go build ./...
      - type: command
        name: frontend_builds
        command: pnpm build
  2:
    name: Beta
    coverage:
      backend: 60
      frontend: 50
    files:
      - .github/workflows/ci.yml
      - .github/workflows/deploy-dev.yml
      - README.md
    checks:
      - type: endpoint
        name: health
        path: /api/health
        expect: 200
      - type: endpoint
        name: discovery
        path: /api/discovery
        expect: 200
      - type: e2e
        name: critical_paths
        required: true
  3:
    name: Production
    coverage:
      backend: 80
      frontend: 70
    files:
      - .github/workflows/ci.yml
      - .github/workflows/deploy-dev.yml
      - .github/workflows/deploy-prod.yml
      - README.md
      - docs/runbook.md
    checks:
      - type: endpoint
        name: health
        path: /api/health
        expect: 200
      - type: endpoint
        name: build_info
        path: /api/build
        expect: 200
      - type: security
        name: dependency_scan
        tools:
          - npm audit
          - govulncheck
      - type: monitoring
        name: cloudwatch_alarms
        required: true
  4:
    name: Mature
    coverage:
      backend: 80
      frontend: 70
    files:
      - .github/workflows/ci.yml
      - .github/workflows/deploy-dev.yml
      - .github/workflows/deploy-prod.yml
      - README.md
      - docs/runbook.md
      - docs/architecture.md
      - docs/dr-plan.md
    checks:
      - type: performance
        name: latency_p95
        max_ms: 500
      - type: performance
        name: page_load_3g
        max_ms: 3000
      - type: performance
        name: bundle_size
        max_kb: 300
      - type: accessibility
        name: lighthouse
        min_score: 90
      - type: dr_test
        name: backup_restore
        required: true
---
# Application Maturity Model

> Quality levels for DigiStratum applications.
> Use this to assess readiness and identify gaps.

---

## Quick Reference

| Level | Name | Summary |
|-------|------|---------|
| 0 | Prototype | Works, no quality gates |
| 1 | Development | Unit tests, CI builds |
| 2 | Beta | E2E tests, auto-deploy to dev |
| 3 | Production | Full NFRs, monitoring, security review |
| 4 | Mature | Performance validated, DR tested |

---

## Level 0: Prototype

**Goal:** Prove the concept works.

### Checklist

- [ ] Core functionality implemented
- [ ] App runs locally
- [ ] Basic happy-path works

### Not Required

- Tests
- CI/CD
- Documentation
- Security review

### When to Advance

Prototype validated, ready for development investment.

---

## Level 1: Development

**Goal:** Establish development practices.

### Checklist

- [ ] Project structure follows template
- [ ] Backend builds: `go build ./...`
- [ ] Frontend builds: `pnpm build`
- [ ] Unit tests for business logic
- [ ] CI pipeline runs on push
- [ ] Code compiles without errors
- [ ] Basic README exists

### Coverage Targets

| Component | Minimum |
|-----------|---------|
| Backend | 40% |
| Frontend | 30% |

### Deployment

- Manual deployment to dev environment
- No production access

### When to Advance

Core features work, basic testing in place, ready for integration testing.

---

## Level 2: Beta

**Goal:** Integration tested, deployable.

### Checklist

- [ ] All FR-* requirements implemented (see REQUIREMENTS.md)
- [ ] FR-* traceability: each requirement has test reference
- [ ] E2E tests for critical user paths
- [ ] Health endpoint: `GET /api/health` returns 200
- [ ] HAL discovery: `GET /api/discovery` returns links
- [ ] Automated deploy to dev on merge
- [ ] Environment configuration documented
- [ ] Error handling: no stack traces to users

### Coverage Targets

| Component | Minimum |
|-----------|---------|
| Backend | 60% |
| Frontend | 50% |
| E2E | Critical paths |

### Deployment

- Automated deploy to dev (CI/CD)
- Manual promotion to staging

### Security

- [ ] Authentication integrated (DSAccount SSO)
- [ ] Tenant isolation in queries
- [ ] Input validation on all endpoints
- [ ] No secrets in code

### When to Advance

Feature complete, stable in dev, ready for production hardening.

---

## Level 3: Production

**Goal:** Production-ready with full quality gates.

### Checklist

- [ ] All NFR-* requirements met (see docs/requirements/)
- [ ] NFR-TEST: Coverage targets met
- [ ] NFR-SEC: Security checklist complete
- [ ] NFR-A11Y: Accessibility audit passed (WCAG 2.1 AA)
- [ ] NFR-PERF: Performance targets met
- [ ] NFR-OBS: Monitoring and alerting active
- [ ] Automated deploy to production (with approval gate)
- [ ] `/api/build` endpoint shows version/commit
- [ ] Runbook documented (incident response)
- [ ] Security review completed

### Coverage Targets

| Component | Minimum |
|-----------|---------|
| Backend | 80% |
| Frontend | 70% |
| E2E | Full FR coverage |

### Deployment

- Automated deploy to dev (on merge to develop)
- Automated deploy to staging (on merge to release/*)
- Automated deploy to prod (on merge to main, with approval)

### Monitoring

- [ ] CloudWatch alarms configured
- [ ] Error rate alerting (Lambda, API Gateway)
- [ ] Latency monitoring (p50, p95, p99)
- [ ] DynamoDB throttle alerts
- [ ] Log aggregation working

### Security

- [ ] OWASP Top 10 addressed
- [ ] Dependency scanning (npm audit, govulncheck)
- [ ] Secrets in AWS Secrets Manager
- [ ] CORS configured properly
- [ ] Security headers set

### When to Advance

Running in production, meeting SLOs, ready for optimization.

---

## Level 4: Mature

**Goal:** Optimized, resilient, fully documented.

### Checklist

- [ ] Performance benchmarks documented and met
- [ ] Load testing completed
- [ ] Accessibility audit by external tool (axe, WAVE)
- [ ] Documentation complete (README, architecture, runbook)
- [ ] Runbook tested with team
- [ ] Disaster recovery plan documented
- [ ] DR tested (backup restore, failover)
- [ ] On-call rotation established
- [ ] SLO/SLA defined and tracked
- [ ] Cost optimization reviewed

### Performance

- [ ] p95 latency < 500ms (measured)
- [ ] Page load < 3s on 3G
- [ ] Bundle size < 300KB gzipped
- [ ] Lighthouse score > 90

### Resilience

- [ ] Graceful degradation tested
- [ ] Rate limiting configured
- [ ] Circuit breakers (if applicable)
- [ ] Backup/restore tested
- [ ] Multi-region (if required)

### Documentation

- [ ] Architecture decision records (ADRs)
- [ ] API documentation complete
- [ ] Onboarding guide for new developers
- [ ] Incident response runbook tested

---

## Maturity Assessment Template

Add this section to your app's README.md or PROJECT_CONTEXT.md:

```markdown
## Application Maturity

**Current Level:** 2 (Beta)
**Target Level:** 3 (Production)
**Assessment Date:** 2026-03-22

### Progress to Next Level

| Requirement | Status | Notes |
|-------------|--------|-------|
| NFR-TEST-001 (80% backend coverage) | 🟡 72% | Need auth tests |
| NFR-SEC-001 (OWASP compliance) | ✅ | Reviewed 2026-03-15 |
| NFR-A11Y-001 (WCAG 2.1 AA) | 🔴 | Not started |
| Monitoring active | 🟡 | Alarms configured, need tuning |
| Security review | ⏳ | Scheduled 2026-04-01 |
```

### Status Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Complete |
| 🟡 | In progress |
| 🔴 | Not started |
| ⏳ | Scheduled |

---

## Level Transitions

### 0 → 1: Start Development

1. Set up project from template
2. Implement core features
3. Add unit tests for business logic
4. Verify CI builds pass

### 1 → 2: Prepare for Beta

1. Complete all FR-* requirements
2. Add E2E tests for critical paths
3. Configure automated deploy to dev
4. Verify health and discovery endpoints

### 2 → 3: Production Hardening

1. Review all NFR-* requirements
2. Achieve coverage targets
3. Complete security review
4. Set up monitoring and alerting
5. Configure production deployment pipeline

### 3 → 4: Optimization

1. Run performance benchmarks
2. Complete accessibility audit
3. Document and test runbook
4. Test disaster recovery
5. Establish SLOs

---

## Enforcement

| Level | Gate | Enforced By |
|-------|------|-------------|
| 1+ | CI builds pass | GitHub Actions |
| 2+ | Coverage thresholds | CI gate |
| 2+ | E2E tests pass | CI gate |
| 3+ | Security scan clean | CI gate |
| 3+ | Accessibility check | CI gate (planned) |

---

## Automated Assessment

Use the `assess-maturity` tool to programmatically check maturity level compliance.

### Quick Start

```bash
# Assess current repo against all levels
go run ./tools/cmd/assess-maturity

# Check specific level
go run ./tools/cmd/assess-maturity --level 2

# JSON output for CI integration
go run ./tools/cmd/assess-maturity --output json
```

### Configuration

Maturity checks are defined in `maturity.yaml` at the repo root. This file uses a machine-readable schema that the assess-maturity tool parses.

```yaml
schema_version: 1
current_level: 1
target_level: 2

levels:
  1:
    name: Development
    checks:
      - type: coverage
        target: backend
        min: 40
      - type: file_exists
        path: .github/workflows/ci.yml
```

See [maturity-schema.md](maturity-schema.md) for complete schema documentation.

### CI Integration

Add maturity checks to your CI workflow:

```yaml
# .github/workflows/ci.yml
- name: Maturity Check
  run: |
    go run ./tools/cmd/assess-maturity --level ${{ vars.TARGET_MATURITY_LEVEL }}
```

### Sample Output

```
Assessing maturity for: myapp
Current level: 1 (Development)
Target level: 2 (Beta)

Level 1 checks:
  ✅ coverage (backend): 45% >= 40%
  ✅ coverage (frontend): 32% >= 30%
  ✅ file_exists: .github/workflows/ci.yml
  ✅ command: Backend builds

Level 2 checks:
  ❌ coverage (backend): 45% < 60%
  ✅ coverage (frontend): 32% < 50%
  ✅ endpoint: Health check (200)
  ❌ command: E2E tests pass

Result: Level 1 PASSED, Level 2 FAILED
Gaps to Level 2:
  - Increase backend coverage to 60%
  - Add E2E tests for critical paths
```

---

## See Also

- [maturity-schema.md](maturity-schema.md) — Machine-readable schema documentation
- [docs/requirements/](requirements/) — NFR specifications
- [docs/reference/](reference/) — Architecture, conventions, API standards
- [docs/reference/agent-contexts.md](reference/agent-contexts.md) — Agent-specific context
