# lazyaws

A terminal UI for managing AWS resources, inspired by lazygit, k9s, and lazydocker.

## Overview

lazyaws provides an interactive terminal interface for common AWS operations, making it faster and more intuitive to manage your AWS infrastructure from the command line.

## Features

### EC2 Management
- **Instance Listing & Discovery**
  - List all EC2 instances with color-coded states (running, stopped, terminated)
  - Multi-region support
  - Filter by state (running, stopped, etc.)
  - Filter by tags
  - Search by name or instance ID
  - Auto-refresh with configurable interval

- **Instance Details**
  - View comprehensive instance metadata
  - Display security groups and network interfaces
  - Show attached volumes and block devices
  - Instance type specifications (vCPUs, memory, network performance)
  - View all instance tags

- **Instance Operations**
  - Start, stop, restart, and terminate instances
  - Bulk operations with multi-select support
  - Confirmation prompts for destructive actions
  - Copy instance ID or IP to clipboard

- **Health & Monitoring**
  - System and instance status checks
  - CloudWatch metrics (CPU utilization, network I/O, disk I/O)
  - Scheduled maintenance events
  - Real-time metrics display

- **SSM Integration**
  - Check SSM connectivity status
  - Launch interactive SSM sessions in new terminal
  - Port forwarding to instances
  - Support for port forwarding to remote hosts via bastion

### S3 Management
- **Bucket Operations**
  - List all S3 buckets with region and creation date
  - Create and delete buckets
  - View bucket policies
  - Manage bucket versioning settings
  - Calculate bucket size and object count

- **Object Navigation**
  - Browse bucket contents with folder hierarchy
  - Navigate through nested folders
  - Pagination support for large buckets
  - Display object size, last modified, and storage class
  - Search and filter objects by prefix or pattern

- **Object Operations**
  - Upload files with progress bars
  - Download files and folders with progress tracking
  - Delete objects with confirmation
  - Copy and move objects between buckets
  - Multipart upload for large files (automatic for files >5MB)

- **Object Details**
  - View object metadata and custom metadata
  - Display object tags
  - Show ETag, content type, and storage class
  - Generate presigned URLs with configurable expiration

- **Advanced Features**
  - Sync local directories to S3 (like `aws s3 sync`)
  - Sync S3 prefixes to local directories
  - Object versioning support
  - List and download specific object versions
  - Delete specific object versions

### EKS Management
- **Cluster Discovery**
  - List all EKS clusters in the region
  - Display cluster name, version, status, and node count
  - Color-coded cluster status

- **Cluster Details**
  - View cluster endpoint and certificate authority
  - Display VPC and subnet configuration
  - Show security group assignments
  - View enabled CloudWatch log types
  - Platform version and creation date
  - Cluster tags and metadata

- **Node Group Management**
  - List all node groups for a cluster
  - View node group scaling configuration (min, max, desired size)
  - Display instance types and AMI type
  - Show node group status and version

- **Cluster Operations**
  - Configure kubectl context automatically
  - Update local kubeconfig
  - Switch between cluster contexts
  - View cluster add-ons and their health status
  - Access cluster logs (if enabled)

## Why lazyaws?

The AWS CLI is powerful but can be verbose and difficult to remember. lazyaws provides:
- **Interactive interface** - Navigate resources visually instead of crafting complex CLI commands
- **Keyboard-driven** - Fast navigation without leaving the terminal
- **Context-aware** - See relevant information and actions for each resource type
- **Safe operations** - Confirmation prompts for destructive actions

## Installation

### Building from Source

```bash
# Clone the repository
git clone https://github.com/fuziontech/lazyaws.git
cd lazyaws

# Build the binary
go build -o lazyaws

# Install to your PATH (optional)
sudo mv lazyaws /usr/local/bin/

# Or run directly
./lazyaws
```

### Pre-built Binaries

Pre-built binaries will be available soon for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### Package Managers (Coming Soon)

```bash
# Homebrew (macOS/Linux)
brew install lazyaws

# apt (Debian/Ubuntu)
sudo apt install lazyaws

# yum (RHEL/CentOS)
sudo yum install lazyaws
```

## Usage

```bash
lazyaws
```

### Keyboard Shortcuts

#### General Navigation
```
↑/↓ or j/k    Navigate lists
←/→ or h/l    Switch between panels
Tab           Cycle through sections
Enter         Select/Execute action
q or Esc      Go back/Exit
?             Show help (context-sensitive)
r             Refresh current view
/             Search/Filter
```

#### EC2 Instance Management
```
# Instance Operations
s             Start instance
S             Stop instance (with confirmation)
r             Restart/Reboot instance
t             Terminate instance (with confirmation)
c             Connect via SSM (opens new terminal)
p             Port forward via SSM

# Information
Enter         View instance details
m             View CloudWatch metrics
h             View health checks
i             View instance type information

# Bulk Operations
Space         Mark/Select multiple instances
a             Select all instances
n             Deselect all

# Clipboard
y             Copy instance ID
Y             Copy IP address

# Filtering
f             Filter by state
F             Filter by tag
/             Search by name or ID
```

#### S3 Bucket & Object Management
```
# Navigation
Enter         Open bucket / Download object
Backspace     Go up one level

# File Operations
u             Upload file
d             Download file/folder
D             Delete object (with confirmation)
c             Copy object
m             Move object

# Bucket Operations
n             Create new bucket
D             Delete bucket (with confirmation)
p             View bucket policy
v             View/Manage versioning

# Object Information
Enter         View object details
i             View object metadata
t             View object tags
g             Generate presigned URL

# Advanced
s             Sync to S3
S             Sync from S3
l             List object versions (if versioning enabled)
```

#### EKS Cluster Management
```
# Cluster Operations
Enter         View cluster details
k             Configure kubectl context
u             Update kubeconfig

# Information
n             View node groups
a             View cluster add-ons
l             View cluster logs
i             View cluster details

# Navigation
Tab           Switch between clusters/node groups/add-ons
```

## Configuration

lazyaws uses your existing AWS CLI configuration (`~/.aws/config` and `~/.aws/credentials`).

### AWS Credentials

Make sure you have AWS credentials configured. You can set them up using:
```bash
aws configure
```

### Using AWS Profiles

Set your AWS profile before launching:
```bash
export AWS_PROFILE=your-profile
lazyaws
```

Or specify the region:
```bash
export AWS_REGION=us-west-2
lazyaws
```

### Configuration File

lazyaws supports configuration files for customizing behavior (auto-refresh intervals, default regions, etc.). Configuration is stored in `~/.config/lazyaws/config.yaml`.

### SSM Session Requirements

For SSM functionality, ensure you have:
- AWS Systems Manager Session Manager plugin installed
- Appropriate IAM permissions for SSM
- SSM agent running on target EC2 instances

Install the Session Manager plugin:
```bash
# macOS (via Homebrew)
brew install --cask session-manager-plugin

# Linux
curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/linux_64bit/session-manager-plugin.rpm" -o "session-manager-plugin.rpm"
sudo yum install -y session-manager-plugin.rpm
```

## Requirements

- **Go 1.21+** (for building from source)
- **AWS CLI v2** (for SSM sessions and kubectl integration)
- **Session Manager Plugin** (for SSM connectivity)
- **kubectl** (optional, for EKS cluster management)

## Development Status

lazyaws is under active development. The core features for EC2, S3, and EKS management are implemented and functional.

### Completed Features
- ✅ Phase 0: Foundation (AWS SDK integration, TUI framework)
- ✅ Phase 1: EC2 Management (full instance lifecycle, SSM, CloudWatch)
- ✅ Phase 2: S3 Management (buckets, objects, versioning, sync)
- ✅ Phase 3: EKS Management (clusters, node groups, kubectl integration)

### Coming Soon
- Comprehensive help system and documentation
- Additional AWS services (Lambda, RDS, DynamoDB, CloudWatch Logs)
- Enhanced filtering and search capabilities
- Theme customization
- Export functionality (CSV, JSON)
- Cost tracking and estimation

See [TODO.md](TODO.md) for the complete roadmap and upcoming features.

## Common Workflows

### Quick EC2 Instance Connect
1. Launch lazyaws
2. Navigate to EC2 section
3. Find your instance (use `/` to search)
4. Press `c` to open SSM session

### Upload Files to S3
1. Navigate to S3 section
2. Select your bucket
3. Navigate to desired folder
4. Press `u` to upload
5. Progress bar shows upload status

### Sync Local Directory to S3
1. Navigate to S3 bucket
2. Navigate to target folder
3. Press `s` for sync
4. Select local directory
5. Automatic sync with change detection

### Configure kubectl for EKS
1. Navigate to EKS section
2. Select your cluster
3. Press `k` to configure kubectl
4. Context automatically added to `~/.kube/config`

## Troubleshooting

### SSM Connection Issues
- Verify SSM agent is installed and running on target instance
- Check IAM role attached to instance has SSM permissions
- Ensure security groups allow outbound HTTPS (443) to SSM endpoints
- Verify Session Manager plugin is installed locally

### S3 Upload/Download Failures
- Check IAM permissions for S3 operations
- Verify bucket policies don't restrict access
- For large files, ensure stable network connection (multipart uploads will resume)

### EKS kubectl Configuration
- Ensure AWS CLI v2 is installed
- Verify IAM permissions for EKS DescribeCluster
- Check that kubectl is installed for cluster operations

## Contributing

Contributions are welcome! Here's how to contribute:

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes**: Follow the existing code style
4. **Add tests**: Ensure your changes are tested
5. **Commit your changes**: `git commit -m 'Add amazing feature'`
6. **Push to the branch**: `git push origin feature/amazing-feature`
7. **Open a Pull Request**

### Development Guidelines
- Follow Go best practices and idioms
- Add unit tests for new functionality
- Update documentation for user-facing changes
- Keep commits focused and atomic
- See [CLAUDE.md](CLAUDE.md) for the development workflow

### Running Tests
```bash
go test ./...
go test -v ./...           # Verbose output
go test -cover ./...       # With coverage
```

## Acknowledgments

Inspired by excellent TUI tools:
- [lazygit](https://github.com/jesseduffield/lazygit) - Terminal UI for git
- [k9s](https://github.com/derailed/k9s) - Terminal UI for Kubernetes
- [lazydocker](https://github.com/jesseduffield/lazydocker) - Terminal UI for Docker

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2) - AWS API integration

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: Report bugs or request features via [GitHub Issues](https://github.com/fuziontech/lazyaws/issues)
- **Discussions**: Join conversations in [GitHub Discussions](https://github.com/fuziontech/lazyaws/discussions)
- **Documentation**: See [TODO.md](TODO.md) for roadmap and [CLAUDE.md](CLAUDE.md) for development workflow
