# lazyaws Development Roadmap

## Phase 0: Foundation

- [x] Choose tech stack (Go/Bubble Tea, Rust/Ratatui, or Python/Textual)
- [x] Set up project structure
- [x] Initialize basic TUI framework
- [x] Implement AWS SDK integration
- [x] Handle AWS credentials and profile selection
- [x] Create main navigation structure
- [x] Implement basic error handling and logging
- [x] Set up configuration file support

## Phase 1: EC2 Management

### Core EC2 Features
- [x] List EC2 instances
  - [x] Display instance ID, name, state, type, AZ
  - [x] Color coding for instance states (running=green, stopped=yellow, terminated=red)
  - [x] Support for multiple regions
- [x] Instance filtering and search
  - [x] Filter by state (running, stopped, etc.)
  - [x] Filter by tag
  - [x] Search by name/ID
- [x] Instance details view
  - [x] Show full instance metadata
  - [x] Display security groups
  - [x] Show attached volumes
  - [x] Display network interfaces

### EC2 Actions
- [x] Instance state management
  - [x] Start instances
  - [x] Stop instances (with confirmation)
  - [x] Restart instances
  - [x] Terminate instances (with confirmation)
- [x] Health checks
  - [x] Show system status checks
  - [x] Show instance status checks
  - [x] Display CloudWatch metrics (CPU, network, etc.)
- [x] SSM integration
  - [x] Check SSM connectivity status
  - [x] Launch SSM session in new terminal
  - [x] Support for SSM port forwarding (via StartPortForward function)

### EC2 Enhancement
- [x] Bulk operations (multi-select)
- [x] Instance type information and pricing
- [x] Auto-refresh with configurable interval
- [x] Copy instance ID/IP to clipboard
- [ ] Fix SSM in-terminal Ctrl+C signal handling
  - Currently Ctrl+C in SSM session terminates the entire session instead of just the running process
  - Need to investigate proper terminal/signal handling for session-manager-plugin
  - May require using pty (pseudo-terminal) or alternative approach

## Phase 2: S3 Management

### Core S3 Features
- [x] List S3 buckets
  - [x] Display bucket name, region, creation date
  - [x] Show bucket size (if available)
- [x] Browse bucket contents
  - [x] Navigate folder structure
  - [x] Display object size, last modified, storage class
  - [x] Support for pagination (large buckets)
- [x] Object details view
  - [x] Show metadata
  - [x] Display tags
  - [x] Show ETag, content type, storage class

### S3 Actions
- [x] File operations
  - [x] Upload files (API implemented, UI shows placeholder message)
  - [x] Download files/folders
  - [x] Delete objects (with confirmation)
  - [x] Copy/move objects between buckets (API implemented)
- [x] Bucket management
  - [x] Create buckets (API implemented)
  - [x] Delete buckets (with confirmation)
  - [x] View bucket policies
  - [x] View bucket versioning settings
- [x] Generate presigned URLs
- [x] Search/filter objects by prefix or pattern

### S3 Enhancement
- [x] Progress bars for uploads/downloads
- [x] Support for multipart uploads
- [x] Sync functionality (like aws s3 sync)
- [x] Object versioning support

## Phase 3: EKS Management

### Core EKS Features
- [x] List EKS clusters
  - [x] Display cluster name, version, status, region
  - [x] Show node count
- [x] Cluster details view
  - [x] Show endpoint and certificate
  - [x] Display networking configuration
  - [x] Show enabled log types
- [x] Node group management
  - [x] List node groups
  - [x] Show node group details (size, instance types)
  - [x] Display scaling configuration

### EKS Actions
- [x] Configure kubectl context
  - [x] Update kubeconfig automatically
  - [x] Switch between cluster contexts
- [x] View cluster add-ons
- [x] Display cluster logs (if enabled)
- [ ] Open cluster in AWS console (browser)

### EKS Enhancement
- [ ] Integration with kubectl for pod viewing
- [ ] Show cluster cost estimation
- [ ] Fargate profile management

## Phase 4: Polish & Additional Features

### UX Improvements
- [ ] Comprehensive help system
- [ ] Customizable keyboard shortcuts
- [ ] Theme support (light/dark, custom colors)
- [ ] Command history
- [x] Vim keybinding modes (complete with search, navigation, commands)
- [x] VIM service navigation via commands (`:ec2`, `:s3`, `:eks` with tab completion)
- [x] K9s-style interface (header with context info, key hints, breadcrumb navigation, cyan highlights)
- [ ] Mouse support (optional)

### Additional AWS Services (Future)
- [ ] Lambda functions
- [ ] CloudWatch Logs
- [ ] RDS instances
- [ ] DynamoDB tables
- [ ] IAM roles and policies
- [ ] VPC and networking
- [ ] Route53 DNS records
- [ ] CloudFormation stacks
- [ ] ECR repositories

### Developer Experience
- [ ] Unit tests
- [ ] Integration tests with LocalStack
- [ ] CI/CD pipeline
- [ ] Documentation
- [ ] Release process (binaries for major platforms)
- [ ] Homebrew formula
- [ ] Package for major Linux distributions

### Performance & Reliability
- [ ] Caching of AWS API responses
- [ ] Retry logic with exponential backoff
- [ ] Handle AWS rate limiting
- [ ] Async operations for better responsiveness
- [ ] Memory optimization for large datasets

## Ideas for Future Consideration

- [ ] Plugin system for custom resource types
- [ ] Export data to CSV/JSON
- [ ] Resource graph visualization
- [ ] Cost tracking and estimates
- [ ] CloudWatch dashboard integration
- [ ] Terraform/CloudFormation integration
- [x] Multi-account/organization support
- [ ] Custom scripts/macros for common workflows
- [x] AWS SSO integration
- [ ] MFA support
