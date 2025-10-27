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
- [ ] Bulk operations (multi-select)
- [x] Instance type information and pricing
- [ ] Auto-refresh with configurable interval
- [ ] Copy instance ID/IP to clipboard

## Phase 2: S3 Management

### Core S3 Features
- [x] List S3 buckets
  - [x] Display bucket name, region, creation date
  - [ ] Show bucket size (if available)
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
  - [ ] Delete objects (with confirmation)
  - [ ] Copy/move objects between buckets
- [ ] Bucket management
  - [ ] Create buckets
  - [ ] Delete buckets (with confirmation)
  - [ ] View bucket policies
  - [ ] View bucket versioning settings
- [ ] Generate presigned URLs
- [ ] Search/filter objects by prefix or pattern

### S3 Enhancement
- [ ] Progress bars for uploads/downloads
- [ ] Support for multipart uploads
- [ ] Sync functionality (like aws s3 sync)
- [ ] Object versioning support

## Phase 3: EKS Management

### Core EKS Features
- [ ] List EKS clusters
  - [ ] Display cluster name, version, status, region
  - [ ] Show node count
- [ ] Cluster details view
  - [ ] Show endpoint and certificate
  - [ ] Display networking configuration
  - [ ] Show enabled log types
- [ ] Node group management
  - [ ] List node groups
  - [ ] Show node group details (size, instance types)
  - [ ] Display scaling configuration

### EKS Actions
- [ ] Configure kubectl context
  - [ ] Update kubeconfig automatically
  - [ ] Switch between cluster contexts
- [ ] View cluster add-ons
- [ ] Display cluster logs (if enabled)
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
- [ ] Vim/Emacs keybinding modes
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
- [ ] Multi-account/organization support
- [ ] Custom scripts/macros for common workflows
- [ ] AWS SSO integration
- [ ] MFA support
