# NFR-PERF: Performance Requirements

> Performance standards for DigiStratum applications.
> Measurable targets with testing approaches and implementation guidance.

---

## Audit-Ready Summary

### NFR-PERF-001: Page Load Time < 3 Seconds

**Acceptance Criteria:**
1. First Contentful Paint (FCP) < 1.8s on 4G mobile
2. Largest Contentful Paint (LCP) < 2.5s
3. Total Blocking Time (TBT) < 200ms
4. Cumulative Layout Shift (CLS) < 0.1
5. Lighthouse CI runs on every PR

**Verification:**
| Method | Location |
|--------|----------|
| Lighthouse CI | `.github/workflows/lighthouse.yml` |
| Config | `lighthouserc.json` |

**Evidence:** Lighthouse CI reports in PR checks, Lighthouse score ≥ 90

---

### NFR-PERF-002: API Response Time p95 < 500ms

**Acceptance Criteria:**
1. API Gateway latency p95 ≤ 500ms (CloudWatch metric)
2. API Gateway latency p99 ≤ 1000ms
3. DynamoDB read latency p95 ≤ 25ms
4. DynamoDB write latency p95 ≤ 50ms
5. CloudWatch alarm triggers if p95 > 500ms for 5 minutes

**Verification:**
| Method | Location |
|--------|----------|
| CloudWatch metrics | `AWS/ApiGateway/Latency` |
| CloudWatch alarm | `infra/lib/monitoring.ts` |
| Load test | `k6` scripts (ad-hoc) |

**Evidence:** CloudWatch dashboard, no alarm triggers in past 30 days

---

### NFR-PERF-003: Time to Interactive < 2 Seconds

**Acceptance Criteria:**
1. Lighthouse TTI metric ≤ 2000ms
2. No JavaScript tasks blocking main thread > 50ms
3. Critical path resources preloaded
4. Non-critical scripts deferred

**Verification:**
| Method | Location |
|--------|----------|
| Lighthouse CI | `.github/workflows/lighthouse.yml` |
| Manual check | Chrome DevTools Performance tab |

**Evidence:** Lighthouse CI reports showing TTI metric

---

### NFR-PERF-004: Bundle Size < 250KB Gzipped

**Acceptance Criteria:**
1. Total JavaScript bundle ≤ 250KB gzipped
2. No single chunk > 100KB gzipped
3. Bundle size reported in CI
4. CI warns (or fails) if bundle exceeds threshold

**Verification:**
| Method | Location |
|--------|----------|
| Build output | `npm run build` |
| CI check | `.github/workflows/ci.yml` (bundle size step) |
| Analyzer | `npx vite-bundle-visualizer` |

**Evidence:** CI logs showing bundle size, bundle analyzer report

---

### NFR-PERF-005: Lighthouse Performance Score ≥ 90

**Acceptance Criteria:**
1. Lighthouse Performance score ≥ 90 on mobile emulation
2. Score maintained across all primary pages (/, /dashboard, /settings)
3. Lighthouse CI runs on every PR
4. Score regression blocks merge

**Verification:**
| Method | Location |
|--------|----------|
| Lighthouse CI | `.github/workflows/lighthouse.yml` |
| PR check | GitHub status check |

**Evidence:** Lighthouse CI badge, historical score trend

---

## Quick Reference

| Requirement | Target | Measurement |
|-------------|--------|-------------|
| NFR-PERF-001 | Page load time < 3 seconds | Lighthouse, WebPageTest |
| NFR-PERF-002 | API response time p95 < 500ms | CloudWatch Metrics |
| NFR-PERF-003 | Time to Interactive < 2 seconds | Lighthouse |
| NFR-PERF-004 | Bundle size < 250KB gzipped | Build metrics |
| NFR-PERF-005 | Lighthouse Performance ≥ 90 | Lighthouse CI |

---

## NFR-PERF-001: Page Load Time

**Target:** Initial page load completes in under 3 seconds on 4G mobile connection.

### Measurement Criteria

| Metric | Target | Acceptable |
|--------|--------|------------|
| First Contentful Paint (FCP) | < 1.8s | < 2.5s |
| Largest Contentful Paint (LCP) | < 2.5s | < 3.0s |
| Total Blocking Time (TBT) | < 200ms | < 300ms |
| Cumulative Layout Shift (CLS) | < 0.1 | < 0.25 |

### Implementation Guidance

1. **Critical CSS Inlining**
   ```html
   <style>
     /* Inline critical above-the-fold CSS */
   </style>
   <link rel="preload" href="/styles.css" as="style" onload="this.onload=null;this.rel='stylesheet'">
   ```

2. **Resource Hints**
   ```html
   <link rel="preconnect" href="https://api.digistratum.com">
   <link rel="dns-prefetch" href="https://account.digistratum.com">
   ```

3. **Image Optimization**
   - Use WebP/AVIF with fallbacks
   - Implement lazy loading for below-fold images
   - Provide responsive srcset

4. **Code Splitting**
   ```typescript
   // Route-based code splitting
   const Dashboard = lazy(() => import('./pages/Dashboard'));
   ```

### Testing Approach

```bash
# Run Lighthouse locally
npx lighthouse http://localhost:5173 --output=json --output-path=./lighthouse-report.json

# CI Integration
npx lhci autorun --config=lighthouserc.json
```

**Lighthouse CI Configuration:**
```json
{
  "ci": {
    "collect": {
      "url": ["http://localhost:5173/"],
      "numberOfRuns": 3
    },
    "assert": {
      "assertions": {
        "first-contentful-paint": ["error", {"maxNumericValue": 1800}],
        "largest-contentful-paint": ["error", {"maxNumericValue": 2500}],
        "total-blocking-time": ["error", {"maxNumericValue": 200}],
        "cumulative-layout-shift": ["error", {"maxNumericValue": 0.1}]
      }
    }
  }
}
```

---

## NFR-PERF-002: API Response Time

**Target:** API response time p95 < 500ms, p99 < 1000ms.

### Latency Targets by Operation

| Operation Type | p50 | p95 | p99 |
|----------------|-----|-----|-----|
| Read (GET) | < 50ms | < 200ms | < 500ms |
| Write (POST/PUT) | < 100ms | < 300ms | < 700ms |
| List/Search | < 150ms | < 500ms | < 1000ms |
| Aggregate | < 200ms | < 700ms | < 1500ms |

### Implementation Guidance

1. **Database Query Optimization**
   ```go
   // Use DynamoDB single-table design for single-request queries
   // Avoid multiple roundtrips
   input := &dynamodb.QueryInput{
       TableName:              aws.String(tableName),
       KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
       // Use projection to fetch only needed attributes
       ProjectionExpression:   aws.String("id, title, status"),
   }
   ```

2. **Connection Reuse**
   ```go
   // Reuse HTTP clients and DynamoDB sessions
   var dynamoClient = dynamodb.NewFromConfig(cfg)
   ```

3. **Response Compression**
   ```go
   // Enable gzip compression for responses > 1KB
   func GzipMiddleware(next http.Handler) http.Handler
   ```

4. **Caching Headers**
   ```go
   w.Header().Set("Cache-Control", "private, max-age=60")
   w.Header().Set("ETag", computeETag(data))
   ```

### CloudWatch Metrics Configuration

```typescript
// cdk/lib/constructs/monitoring.ts
export const PerformanceBaselines = {
  apiLatencyP95Ms: 500,
  apiLatencyP99Ms: 1000,
  maxErrorRatePercent: 1,
  dynamoReadLatencyMs: 25,
  dynamoWriteLatencyMs: 50,
};
```

### Testing Approach

```bash
# Load testing with k6
k6 run --vus 50 --duration 30s load-test.js
```

**k6 Script:**
```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
  },
};

export default function() {
  const res = http.get('https://api.example.com/items');
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time OK': (r) => r.timings.duration < 500,
  });
  sleep(1);
}
```

---

## NFR-PERF-003: Time to Interactive

**Target:** Time to Interactive (TTI) < 2 seconds.

### Definition

TTI measures when the page becomes fully interactive:
- First Contentful Paint has occurred
- Event handlers are registered for most visible elements
- Page responds to user input within 50ms

### Implementation Guidance

1. **Defer Non-Critical JavaScript**
   ```html
   <script src="/analytics.js" defer></script>
   ```

2. **Minimize Main Thread Work**
   - Break up long tasks (> 50ms) into smaller chunks
   - Use `requestIdleCallback` for non-urgent work
   - Move heavy computation to Web Workers

3. **Optimize Third-Party Scripts**
   ```html
   <!-- Load third-party scripts after page load -->
   <script>
     window.addEventListener('load', () => {
       const script = document.createElement('script');
       script.src = 'https://analytics.example.com/script.js';
       document.body.appendChild(script);
     });
   </script>
   ```

4. **Reduce JavaScript Execution Time**
   - Tree-shake unused code
   - Avoid synchronous layout/style recalculations
   - Use CSS for animations instead of JavaScript

### Testing Approach

```bash
# Lighthouse measures TTI
npx lighthouse http://localhost:5173 --only-categories=performance
```

---

## NFR-PERF-004: Bundle Size

**Target:** JavaScript bundle size < 250KB gzipped.

### Budget Breakdown

| Category | Budget (gzipped) |
|----------|------------------|
| Framework (React) | 45KB |
| Router | 15KB |
| State Management | 10KB |
| UI Components | 50KB |
| Application Code | 80KB |
| Utilities/Vendor | 50KB |
| **Total** | **250KB** |

### Implementation Guidance

1. **Bundle Analysis**
   ```bash
   # Vite bundle analyzer
   npx vite-bundle-visualizer
   ```

2. **Dynamic Imports**
   ```typescript
   // Lazy load routes
   const Settings = lazy(() => import('./pages/Settings'));
   
   // Lazy load heavy libraries
   const Chart = lazy(() => import('chart.js').then(m => ({ default: m.Chart })));
   ```

3. **Tree Shaking**
   ```typescript
   // ❌ Import entire library
   import _ from 'lodash';
   
   // ✅ Import specific functions
   import debounce from 'lodash/debounce';
   ```

4. **External CDN for Large Libraries** (if not tree-shakeable)
   ```typescript
   // vite.config.ts
   build: {
     rollupOptions: {
       external: ['chart.js'],
       output: {
         globals: {
           'chart.js': 'Chart'
         }
       }
     }
   }
   ```

### CI Gate

```yaml
# .github/workflows/ci.yml
- name: Check bundle size
  run: |
    npm run build
    BUNDLE_SIZE=$(du -sk dist/assets/*.js | awk '{sum+=$1} END {print sum}')
    if [ $BUNDLE_SIZE -gt 300 ]; then
      echo "Bundle size ${BUNDLE_SIZE}KB exceeds 300KB limit"
      exit 1
    fi
```

---

## NFR-PERF-005: Lighthouse Score

**Target:** Lighthouse Performance score ≥ 90.

### Score Components

| Metric | Weight | Target |
|--------|--------|--------|
| First Contentful Paint | 10% | < 1.8s |
| Speed Index | 10% | < 3.4s |
| Largest Contentful Paint | 25% | < 2.5s |
| Time to Interactive | 10% | < 3.8s |
| Total Blocking Time | 30% | < 200ms |
| Cumulative Layout Shift | 15% | < 0.1 |

### Implementation Checklist

- [ ] Serve static assets from CDN
- [ ] Enable HTTP/2 or HTTP/3
- [ ] Implement service worker for caching
- [ ] Use font-display: swap for custom fonts
- [ ] Preload critical resources
- [ ] Optimize images (WebP, lazy loading, sizing)
- [ ] Minimize render-blocking resources
- [ ] Enable text compression (gzip/brotli)

### CI Integration

```yaml
# .github/workflows/lighthouse.yml
name: Lighthouse CI
on: [push, pull_request]

jobs:
  lighthouse:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
      - run: npm ci && npm run build
      - name: Run Lighthouse
        uses: treosh/lighthouse-ci-action@v12
        with:
          urls: http://localhost:5173/
          uploadArtifacts: true
          configPath: ./lighthouserc.json
```

---

## Cold Start Optimization (Lambda)

**Target:** Lambda cold start < 1 second.

### Optimization Strategies

1. **Use ARM64 Runtime**
   ```typescript
   new lambda.Function(this, 'Handler', {
     architecture: lambda.Architecture.ARM_64,
   });
   ```

2. **Minimize Dependencies**
   - Avoid large frameworks
   - Use AWS SDK v2 individual clients vs. full SDK

3. **Provisioned Concurrency** (for critical endpoints)
   ```typescript
   const alias = handler.addAlias('live', {
     provisionedConcurrentExecutions: 2,
   });
   ```

4. **Init Code Optimization**
   ```go
   // Move initialization outside handler
   var dynamoClient *dynamodb.Client
   
   func init() {
       cfg, _ := config.LoadDefaultConfig(context.Background())
       dynamoClient = dynamodb.NewFromConfig(cfg)
   }
   
   func Handler(ctx context.Context, event Event) error {
       // Use pre-initialized client
   }
   ```

---

## Monitoring & Alerting

### CloudWatch Dashboard Widgets

```typescript
// Include in monitoring construct
new cloudwatch.GraphWidget({
  title: 'API Latency',
  left: [apiLatencyP50, apiLatencyP95, apiLatencyP99],
  leftAnnotations: [
    { value: 500, label: 'p95 Target', color: '#ff7f0e' },
  ],
});
```

### Performance Alarms

| Alarm | Threshold | Evaluation |
|-------|-----------|------------|
| High Latency | p95 > 500ms | 2/3 periods (1m) |
| Slow Cold Starts | Duration > 1s | 5/5 periods (1m) |
| DynamoDB Throttles | > 0 | Any occurrence |

---

## Traceability

| Requirement | Implementation | Test |
|-------------|----------------|------|
| NFR-PERF-001 | CDN, code splitting, image optimization | Lighthouse CI |
| NFR-PERF-002 | Query optimization, caching | k6 load tests |
| NFR-PERF-003 | Deferred scripts, code splitting | Lighthouse CI |
| NFR-PERF-004 | Tree shaking, lazy loading | Bundle size gate |
| NFR-PERF-005 | All above combined | Lighthouse CI |

---

*Last updated: 2026-03-22*
