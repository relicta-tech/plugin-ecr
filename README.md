# plugin-ecr

Relicta plugin for pushing container images to AWS Elastic Container Registry (ECR).

## Features

- Push container images to private ECR registries
- Push container images to public ECR (public.ecr.aws)
- Automatic ECR authentication via AWS credentials
- Multiple image tag support with template variables
- Automatic repository creation
- Dry-run mode for testing

## Installation

```bash
relicta plugin install ecr
```

## Configuration

### Basic Configuration

```yaml
plugins:
  ecr:
    region: us-east-1
    registry_id: "123456789012"  # AWS Account ID
    repository: my-app
    source_image: myapp:latest
    tags:
      - "{{.Version}}"
      - latest
```

### Full Configuration

```yaml
plugins:
  ecr:
    # AWS Configuration
    region: us-west-2
    access_key_id: ${AWS_ACCESS_KEY_ID}
    secret_access_key: ${AWS_SECRET_ACCESS_KEY}
    session_token: ${AWS_SESSION_TOKEN}  # Optional, for temporary credentials
    profile: production                   # Optional, AWS profile name
    role_arn: arn:aws:iam::123456789012:role/DeployRole  # Optional, assume role

    # ECR Configuration
    registry_id: "123456789012"  # AWS Account ID (required for private ECR)
    repository: my-app
    public: false                # Set to true for public ECR

    # Image Configuration
    source_image: myapp:latest   # Local image to push
    tags:
      - "{{.Version}}"
      - "{{.GitSHA}}"
      - latest
      - stable

    # Behavior
    create_repository: true      # Create repository if it doesn't exist
    dry_run: false
```

### Public ECR Configuration

```yaml
plugins:
  ecr:
    region: us-east-1
    repository: myorg/my-app     # Public ECR repository format
    public: true
    source_image: myapp:latest
    tags:
      - "{{.Version}}"
      - latest
```

## Configuration Options

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `region` | string | Yes | - | AWS region |
| `registry_id` | string | No* | - | AWS Account ID (*required for private ECR) |
| `repository` | string | Yes | - | ECR repository name |
| `public` | bool | No | `false` | Use public ECR |
| `source_image` | string | Yes | - | Local Docker image to push |
| `tags` | []string | No | `["{{.Version}}"]` | Image tags to apply |
| `access_key_id` | string | No | - | AWS access key ID |
| `secret_access_key` | string | No | - | AWS secret access key |
| `session_token` | string | No | - | AWS session token |
| `profile` | string | No | - | AWS profile name |
| `role_arn` | string | No | - | ARN of IAM role to assume |
| `create_repository` | bool | No | `false` | Create repository if not exists |
| `dry_run` | bool | No | `false` | Run without making changes |

## Tag Templates

The following template variables are available for tags:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.Version}}` | Release version | `1.2.3` |
| `{{.PreviousVersion}}` | Previous version | `1.2.2` |
| `{{.ProjectName}}` | Project name | `my-app` |
| `{{.GitSHA}}` | Short Git SHA (7 chars) | `abc1234` |
| `{{.GitSHAFull}}` | Full Git SHA | `abc1234567890...` |
| `{{.GitBranch}}` | Git branch (/ replaced with -) | `feature-foo` |

## AWS Authentication

The plugin supports multiple authentication methods:

1. **Environment Variables**: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`
2. **AWS Profile**: Use `profile` option or `AWS_PROFILE` environment variable
3. **IAM Role**: Use `role_arn` to assume a role
4. **EC2 Instance Profile**: Automatic when running on EC2
5. **ECS Task Role**: Automatic when running in ECS

## Required IAM Permissions

### Private ECR

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:InitiateLayerUpload",
        "ecr:UploadLayerPart",
        "ecr:CompleteLayerUpload",
        "ecr:PutImage"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ecr:CreateRepository",
        "ecr:DescribeRepositories"
      ],
      "Resource": "arn:aws:ecr:*:*:repository/*"
    }
  ]
}
```

### Public ECR

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr-public:GetAuthorizationToken",
        "ecr-public:BatchCheckLayerAvailability",
        "ecr-public:InitiateLayerUpload",
        "ecr-public:UploadLayerPart",
        "ecr-public:CompleteLayerUpload",
        "ecr-public:PutImage"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": "sts:GetServiceBearerToken",
      "Resource": "*"
    }
  ]
}
```

## Hooks

This plugin supports the following hooks:

- `post_publish` - Push images after release is published

## Examples

### Push with Multiple Tags

```yaml
plugins:
  ecr:
    region: us-east-1
    registry_id: "123456789012"
    repository: my-app
    source_image: my-app:build
    tags:
      - "{{.Version}}"
      - "v{{.Version}}"
      - "{{.GitSHA}}"
      - latest
```

### CI/CD with Role Assumption

```yaml
plugins:
  ecr:
    region: us-east-1
    role_arn: arn:aws:iam::123456789012:role/DeployRole
    registry_id: "123456789012"
    repository: my-app
    source_image: my-app:${CI_COMMIT_SHA}
    tags:
      - "{{.Version}}"
    create_repository: true
```

### Multi-Region Deployment

```yaml
# Use multiple plugin instances
plugins:
  ecr-us:
    region: us-east-1
    registry_id: "123456789012"
    repository: my-app
    source_image: my-app:latest
    tags:
      - "{{.Version}}"

  ecr-eu:
    region: eu-west-1
    registry_id: "123456789012"
    repository: my-app
    source_image: my-app:latest
    tags:
      - "{{.Version}}"
```

## License

Apache-2.0
