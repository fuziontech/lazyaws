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
- [ ] Instance filtering and search
  - [x] Filter by state (running, stopped, etc.)
  - [x] Filter by tag
  - [ ] Search by name/ID
- [ ] Instance details view
  - [ ] Show full instance metadata
  - [ ] Display security groups
  - [ ] Show attached volumes
  - [ ] Display network interfaces

### EC2 Actions
- [ ] Instance state management
  - [ ] Start instances
  - [ ] Stop instances (with confirmation)
  - [ ] Restart instances
  - [ ] Terminate instances (with double confirmation)
- [ ] Health checks
  - [ ] Show system status checks
  - [ ] Show instance status checks
  - [ ] Display CloudWatch metrics (CPU, network, etc.)
- [ ] SSM integration
  - [ ] Check SSM connectivity status
  - [ ] Launch SSM session in new terminal
  - [ ] Support for SSM port forwarding

### EC2 Enhancement
- [ ] Bulk operations (multi-select)
- [ ] Instance type information and pricing
- [ ] Auto-refresh with configurable interval
- [ ] Copy instance ID/IP to clipboard

## Phase 2: S3 Management

### Core S3 Features
- [ ] List S3 buckets
  - [ ] Display bucket name, region, creation date
  - [ ] Show bucket size (if available)
- [ ] Browse bucket contents
  - [ ] Navigate folder structure
  - [ ] Display object size, last modified, storage class
  - [ ] Support for pagination (large buckets)
- [ ] Object details view
  - [ ] Show metadata
  - [ ] Display tags
  - [ ] Show permissions

### S3 Actions
- [ ] File operations
  - [ ] Upload files/folders
  - [ ] Download files/folders
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
