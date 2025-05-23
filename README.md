# AWS-Terror

AWS-Terror is a CLI tool that helps detect configuration drift between AWS EC2 instances and their corresponding Terraform state or configuration files. This ensures your infrastructure remains in sync with your Infrastructure as Code (IaC) definitions.

## Features

- Detect configuration drift between AWS EC2 instances and Terraform state/config
- Support for both Terraform state files and HCL configuration
- Concurrent instance checking with configurable concurrency
- Multiple output formats (text, JSON, YAML)
- Customizable attribute checking
- Detailed drift reporting
- In-memory caching with TTL support
- Prometheus-based metrics collection

## Installation

```bash
# Clone the repository
git clone https://github.com/katungi/aws-terror

# Build the binary
make build
```

## Usage

```bash
# Check drift using Terraform state file
aws-terror drift -i i-1234567890abcdef0 -s terraform.tfstate

# Check drift using Terraform configuration directory
aws-terror drift -i i-1234567890abcdef0 -c ./terraform/

# Check multiple instances
aws-terror drift -i i-1234567890abcdef0,i-0987654321fedcba0 -s terraform.tfstate

# Customize attributes to check
aws-terror drift -i i-1234567890abcdef0 -s terraform.tfstate -a instance_type,ami,tags

# Output in JSON format
aws-terror drift -i i-1234567890abcdef0 -s terraform.tfstate --output json
```

## Configuration

### AWS Credentials

AWS-Terror uses the AWS SDK's default credential provider chain. You can configure credentials through:

- Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
- AWS credentials file (~/.aws/credentials)
- IAM roles for EC2 instances

### AWS Region

The AWS region can be specified through:

1. Command line flag: `--region`
2. Environment variable: `AWS_REGION`
3. AWS configuration file

## Technical Approach

### Architecture

The application is structured into several key packages:

1. `cmd/` - Command line interface using Cobra
2. `aws/` - AWS SDK integration and EC2 instance data retrieval
3. `pkg/drift/` - Core drift detection logic
4. `pkg/terraform/` - Terraform state and HCL configuration parsing
5. `pkg/output/` - Result formatting in various output formats
6. `pkg/cache/` - Thread-safe in-memory caching with TTL
7. `pkg/metrics/` - Prometheus-based metrics collection

### Key Components

#### Caching Mechanism

The application implements a thread-safe in-memory caching system with the following features:

- TTL (Time-To-Live) support for automatic entry expiration
- Thread-safe operations using mutex locks
- Generic interface supporting any data type
- Automatic cleanup of expired entries
- Concurrent access support

Key operations:

- `Set`: Add or update cache entries with automatic TTL
- `Get`: Retrieve cached values with expiration checking
- `Delete`: Remove specific entries
- `Clear`: Remove all cached entries
- `Cleanup`: Remove expired entries

#### Metrics Collection

The metrics collector provides monitoring capabilities using Prometheus:

- AWS API call metrics
- Drift detection operation metrics
- Cache performance metrics
- Error tracking

Metrics are exposed via a Prometheus endpoint for monitoring and alerting.

### Key Design Decisions

1. **Concurrent Processing**: Implemented a worker pool pattern for checking multiple instances concurrently while controlling resource usage.

2. **Flexible Configuration Sources**: Support for both Terraform state files (JSON) and HCL configuration files, allowing users to check drift against their preferred source of truth.

3. **Extensible Attribute Checking**: Modular approach to adding new attributes for drift detection.

4. **Structured Output**: Multiple output formats (text, JSON, YAML) for better integration with other tools and workflows.

5. **Type-Safe Comparisons**: Robust value comparison logic handling different data types and nested structures.

### Technical Challenges

1. **Type Normalization**:
   - Challenge: AWS and Terraform represent values differently (e.g., numbers as strings vs. integers)
   - Solution: Implemented value normalization to ensure accurate comparisons

2. **Nested Attribute Comparison**:
   - Challenge: Handling complex nested structures like block devices and tags
   - Solution: Developed recursive comparison logic with support for maps and slices

3. **HCL Parsing**:
   - Challenge: Extracting specific instance configurations from HCL files
   - Solution: Used HCL parser with careful attribute extraction and validation

4. **Concurrent Resource Access**:
   - Challenge: Managing concurrent AWS API calls and resource usage
   - Solution: Implemented rate limiting and connection pooling

## Sample Files

Example configuration files are provided in the `samples/` directory:

- `samples/instance.tf` - Example Terraform configuration for an EC2 instance
- `samples/terraform.tfstate` - Sample Terraform state file showing the expected structure

These samples demonstrate the structure of data that AWS-Terror compares when detecting drift, including instance attributes like type, AMI, security groups, tags, and block device mappings.

## Dependencies

- [AWS SDK for Go](https://github.com/aws/aws-sdk-go-v2) - AWS API integration
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [HCL](https://github.com/hashicorp/hcl/v2) - Terraform configuration parsing
- [Prometheus Client](https://github.com/prometheus/client_golang) - Metrics collection and exposition
- [testify](https://github.com/stretchr/testify) - Testing framework
- [go-cmp](https://github.com/google/go-cmp) - Deep equality comparison
