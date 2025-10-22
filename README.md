# lazyaws

A terminal UI for managing AWS resources, inspired by lazygit, k9s, and lazydocker.

## Overview

lazyaws provides an interactive terminal interface for common AWS operations, making it faster and more intuitive to manage your AWS infrastructure from the command line.

## Features

### EC2 Management
- List EC2 instances with filtering and search
- View instance health checks and status
- Connect via AWS Systems Manager (SSM)
- Restart, stop, and terminate instances
- View instance details and metadata

### S3 Management
- Browse buckets and objects
- Upload and download files
- Manage bucket policies and configurations
- View object metadata and properties

### EKS Management
- List EKS clusters
- Configure kubectl context automatically
- View cluster details and status
- Manage node groups

## Why lazyaws?

The AWS CLI is powerful but can be verbose and difficult to remember. lazyaws provides:
- **Interactive interface** - Navigate resources visually instead of crafting complex CLI commands
- **Keyboard-driven** - Fast navigation without leaving the terminal
- **Context-aware** - See relevant information and actions for each resource type
- **Safe operations** - Confirmation prompts for destructive actions

## Installation

```bash
# Installation instructions coming soon
```

## Usage

```bash
lazyaws
```

### Keyboard Shortcuts

```
# Navigation
↑/↓ or j/k    Navigate lists
←/→ or h/l    Switch between panels
Tab           Cycle through sections

# Actions
Enter         Select/Execute
Space         Mark/Select multiple
d             Delete/Terminate (with confirmation)
r             Refresh current view
s             Search/Filter
?             Show help

# Context-specific
EC2:
  c           Connect via SSM
  R           Restart instance
  S           Stop instance

S3:
  u           Upload file
  D           Download file

EKS:
  k           Configure kubectl context
```

## Configuration

lazyaws uses your existing AWS CLI configuration (`~/.aws/config` and `~/.aws/credentials`).

Set your AWS profile:
```bash
export AWS_PROFILE=your-profile
lazyaws
```

## Development Status

This project is in early development. See [TODO.md](TODO.md) for the current roadmap.

## Contributing

Contributions welcome! Please open an issue to discuss major changes.

## License

TBD
