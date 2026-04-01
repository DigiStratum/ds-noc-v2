# App-Specific Infrastructure

This directory is **APP-OWNED** — it will not be overwritten by template updates.

Place your app-specific CDK constructs here:

```typescript
// resources.ts
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import { Construct } from 'constructs';

export class AppResources extends Construct {
  constructor(scope: Construct, id: string) {
    super(scope, id);

    // Example: App-specific DynamoDB table
    new dynamodb.Table(this, 'BookmarksTable', {
      partitionKey: { name: 'pk', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'sk', type: dynamodb.AttributeType.STRING },
    });
  }
}
```

Then import in the main stack:

```typescript
// infra/lib/app-stack.ts
import { AppResources } from './app/resources';

// In stack constructor:
new AppResources(this, 'AppResources');
```
