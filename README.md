# lazyaws

A terminal UI for managing AWS resources, inspired by lazygit, k9s, and lazydocker.

## Features

### EC2
- List instances with color-coded states
- Start/stop/reboot/terminate instances
- Multi-select for bulk operations
- SSM sessions with proper Ctrl+C handling
- View instance details, metrics, and health checks
- k9s integration for EKS nodes
- Returns to EC2 view after SSM exit

### S3
- Browse buckets and objects
- **Edit files in $EDITOR** - press `e` to edit, auto-uploads on save
- Download/delete objects with typed confirmation
- Generate presigned URLs
- View bucket policies and versioning
- Returns to same location after editing

### EKS
- List clusters with status
- View node groups and add-ons
- Update kubeconfig automatically
- Launch k9s for clusters

### Navigation
- **VIM-style keybindings** - j/k, g/G, Ctrl+d/u for navigation
- **Search** - `/` to search, `n/N` for next/prev match
- **Commands** - `:q` quit, `:r` refresh, `:help` show help, `:ec2/:s3/:eks` switch services
- **Multi-region/account** - `:region` and `:account` to switch contexts

## Installation

```bash
git clone https://github.com/fuziontech/lazyaws.git
cd lazyaws
go build -o lazyaws
sudo mv lazyaws /usr/local/bin/
```

## Usage

```bash
lazyaws
```

Press `:help` or `?` for keyboard shortcuts.

### Quick Start

**Connect to EC2 via SSM:**
1. Press `/` to search for instance
2. Press `c` to connect
3. Ctrl+C works inside session (kills commands, not the session)
4. `exit` to return to lazyaws

**Edit S3 file:**
1. Navigate to object
2. Press `e` to edit in $EDITOR
3. Save and quit
4. File automatically uploads
5. Returns to same S3 location

**Key Shortcuts:**
```
j/k, ↑/↓      Navigate
g/G           Top/bottom
Ctrl+d/u      Page down/up
/             Search
n/N           Next/prev match
Enter         Select/view details
Esc/q         Back/quit
:help         Show help
```

**EC2:**
```
s/S           Start/stop
r/t           Reboot/terminate
c             SSM connect
9             Launch k9s (EKS nodes)
Space         Multi-select
```

**S3:**
```
e             Edit file in $EDITOR
d             Download
D             Delete (typed confirmation)
u             Presigned URL
p/v           Policy/versioning
```

**EKS:**
```
9             Launch k9s
u             Update kubeconfig
```

## Configuration

Uses existing AWS CLI configuration (`~/.aws/config` and `~/.aws/credentials`).

```bash
aws configure                    # Set up credentials
export AWS_PROFILE=your-profile  # Use specific profile
export AWS_REGION=us-west-2      # Override region
```

### SSO Authentication

lazyaws supports AWS SSO with automatic account and region selection.

## Requirements

- **Go 1.21+** (build only)
- **AWS CLI v2** (for SSM)
- **Session Manager Plugin** (for SSM connectivity)
- **kubectl** (optional, for EKS)

Install Session Manager Plugin:
```bash
# macOS
brew install --cask session-manager-plugin

# Linux
curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/linux_64bit/session-manager-plugin.rpm" -o "session-manager-plugin.rpm"
sudo yum install -y session-manager-plugin.rpm
```

## Contributing

See [CLAUDE.md](CLAUDE.md) for development workflow.

```bash
git checkout -b feature/your-feature
# Make changes
go test ./...
git commit -m 'Add feature'
git push origin feature/your-feature
# Open PR
```

## Acknowledgments

Inspired by [lazygit](https://github.com/jesseduffield/lazygit), [k9s](https://github.com/derailed/k9s), and [lazydocker](https://github.com/jesseduffield/lazydocker).

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2).

## License

MIT License - see [LICENSE](LICENSE) file.
