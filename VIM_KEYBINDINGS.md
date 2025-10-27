# VIM Keybindings Guide for lazyaws

lazyaws implements comprehensive VIM-style keybindings for efficient navigation and control, similar to tools like k9s. This guide covers all VIM mechanics available in the application.

## Table of Contents

- [Modes](#modes)
- [Navigation](#navigation)
- [Search](#search)
- [Commands](#commands)
- [Screen-Specific Keys](#screen-specific-keys)

## Modes

lazyaws has three modes, similar to VIM:

### Normal Mode (Default)
- The default mode where you can navigate and execute actions
- All navigation keys and single-key commands work in this mode

### Search Mode (/)
- Activated by pressing `/`
- Allows you to search through the current list
- Type your search query and press `Enter` to apply
- Press `ESC` to cancel

### Command Mode (:)
- Activated by pressing `:`
- Execute VIM-style commands
- Type your command and press `Enter` to execute
- Press `ESC` to cancel

## Navigation

All views (EC2, S3, EKS) support consistent VIM-style navigation:

### Basic Movement

| Key | Action | Description |
|-----|--------|-------------|
| `j` / `↓` | Move down | Move selection down one item |
| `k` / `↑` | Move up | Move selection up one item |
| `g` | Go to top | Jump to first item in list |
| `G` / `Ctrl+G` | Go to bottom | Jump to last item in list |

### Page Movement

| Key | Action | Description |
|-----|--------|-------------|
| `Ctrl+D` | Half page down | Move down half a page (10 items) |
| `Ctrl+U` | Half page up | Move up half a page (10 items) |
| `Ctrl+F` / `PgDn` | Page down | Move down one full page (20 items) |
| `Ctrl+B` / `PgUp` | Page up | Move up one full page (20 items) |

### Navigation Tips

- Page size defaults to 20 items but adjusts to terminal height
- All navigation respects list boundaries (won't go past first/last item)
- Navigation is smooth and responsive

## Search

Search functionality works across all list views:

### Entering Search Mode

1. Press `/` to enter search mode
2. Type your search query (case-insensitive)
3. Press `Enter` to apply search
4. Press `ESC` to cancel

### Search Behavior

| Key | Action | Description |
|-----|--------|-------------|
| `/` | Start search | Enter search mode |
| `Enter` | Apply search | Execute search and jump to first match |
| `n` | Next match | Jump to next search result |
| `N` | Previous match | Jump to previous search result |
| `ESC` | Cancel search | Exit search mode without applying |

### What Gets Searched

- **EC2 Screen**: Instance ID, name, state, type, public IP, private IP
- **S3 Buckets**: Bucket name, region
- **S3 Objects**: Object key/path

### Search Features

- Search results are highlighted in status bar
- Shows count of matches found
- Wraps around (pressing `n` at last match goes to first)
- Search persists until cleared with `:cf` command

## Commands

Press `:` to enter command mode and execute VIM-style commands:

### Available Commands

| Command | Shortcut | Description |
|---------|----------|-------------|
| `:q` | `:quit` | Quit current view or application |
| `:r` | `:refresh` | Refresh current view |
| `:sa` | - | Select all items (EC2 only) |
| `:da` | - | Deselect all items (EC2 only) |
| `:cf` | `:clearfilter` | Clear active search/filter |
| `:help` / `:h` / `:?` | - | Show help message with available commands |

### Command Examples

```
:q          → Quit current view (or app if on main view)
:r          → Reload EC2 instances / S3 buckets / etc.
:sa         → Select all EC2 instances for bulk operations
:da         → Clear all EC2 instance selections
:cf         → Clear active search filter
:help       → Display command help in status bar
```

### Command Features

- Commands are executed immediately on `Enter`
- Press `ESC` to cancel command entry
- Commands are context-aware (some only work in specific views)
- Invalid commands show error message in status bar

## Screen-Specific Keys

In addition to VIM navigation, each screen has its own action keys:

### EC2 Instance List

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` / `i` | View details | Open instance details view |
| `Space` | Toggle select | Select/deselect instance for bulk operations |
| `s` | Start | Start instance (or bulk start if multiple selected) |
| `S` | Stop | Stop instance (with confirmation) |
| `R` | Reboot | Reboot instance (with confirmation) |
| `t` | Terminate | Terminate instance (with confirmation) |
| `a` | Auto-refresh | Toggle 30-second auto-refresh |
| `x` | Clear selections | Deselect all instances |
| `y` | Copy to clipboard | Copy IP or instance ID |
| `f` | Filter | Legacy filter mode (prefer `/` search) |
| `c` | Change region | Cycle through configured AWS regions |
| `1/2/3` | Switch service | Jump to EC2/S3/EKS |
| `Tab` | Next service | Cycle through services |
| `q` / `:q` | Quit | Exit application |

### EC2 Instance Details

| Key | Action | Description |
|-----|--------|-------------|
| `s` | Start | Start this instance |
| `S` | Stop | Stop this instance |
| `R` | Reboot | Reboot this instance |
| `t` | Terminate | Terminate this instance |
| `C` | SSM Connect | Launch SSM session (if connected) |
| `ESC` / `q` / `:q` | Back | Return to instance list |

### S3 Bucket List

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` / `i` | Browse | Open bucket and view contents |
| `D` | Delete bucket | Delete bucket (must be empty) |
| `p` | Policy | View bucket policy |
| `v` | Versioning | View bucket versioning configuration |
| `r` | Refresh | Reload bucket list |
| `c` | Change region | Cycle through regions |
| `q` / `:q` | Quit | Exit application |

### S3 Object Browser

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` / `i` | Open/View | Open folder or view file details |
| `d` | Download | Download selected file |
| `D` | Delete | Delete selected object (with confirmation) |
| `p` | Presigned URL | Generate 1-hour presigned URL |
| `h` / `Backspace` | Up directory | Go to parent folder |
| `n` | Next page | Load next page (if paginated, or next search result) |
| `ESC` / `q` / `:q` | Back | Return to bucket list |

### S3 Object Details

| Key | Action | Description |
|-----|--------|-------------|
| `d` | Download | Download this file |
| `p` | Presigned URL | Generate 1-hour presigned URL |
| `ESC` / `q` / `:q` | Back | Return to object browser |

## Tips and Tricks

### Efficient Workflow

1. **Quick Jump**: Use `g` and `G` to quickly jump to top/bottom of long lists
2. **Rapid Navigation**: Use `Ctrl+D` and `Ctrl+U` for fast scrolling
3. **Search and Act**: Press `/`, search for what you need, press `n`/`N` to navigate matches
4. **Bulk Operations**: In EC2, press `Space` to select multiple instances, then use `s`/`S`/`R`/`t` for bulk actions
5. **Command Mode**: Use `:sa` to select all instances quickly for bulk operations

### Muscle Memory from VIM

If you're familiar with VIM, these keybindings will feel natural:

- `j`/`k` for up/down navigation
- `g`/`G` for top/bottom
- `Ctrl+D`/`Ctrl+U` for page movement
- `/` for search, `n`/`N` for next/previous
- `:` for command mode
- `:q` to quit

### Muscle Memory from k9s

If you use k9s, lazyaws follows similar patterns:

- `/` for filtering/search
- `:` for commands
- `Ctrl+D`/`Ctrl+U` for page navigation
- `:q` to quit views
- Context-sensitive action keys

## Visual Feedback

### Mode Indicators

When in search or command mode, you'll see an indicator at the bottom of the screen:

- **Search Mode**: `/your-search-query` (yellow/orange background)
- **Command Mode**: `:your-command` (cyan/blue background)

### Status Messages

- Search results show match count: "Found 5 matches"
- Commands show execution status: "Successfully refreshed"
- Errors display in red at the bottom

### Help Bar

The bottom of the screen shows contextual help:
- First line: Quick reference for current view's keys
- Second line: VIM commands available (`:q`, `:r`, `:sa`, etc.)

## Configuration

Currently, VIM keybindings are enabled by default and cannot be disabled. Future versions may include:

- Customizable keybindings
- Option to disable VIM mode
- Additional VIM features (marks, macros, etc.)

## Troubleshooting

### Search Not Working?
- Make sure you're in a list view (not a details view)
- Press `ESC` if you're stuck in a mode
- Use `:cf` to clear any existing filters

### Navigation Keys Not Responding?
- Ensure you're not in search (`/`) or command (`:`) mode
- Press `ESC` to return to normal mode
- Check that your terminal supports the key combinations

### Confirmation Dialogs
- Some actions (stop, terminate, delete) require confirmation
- Press `y` to confirm, `n` or `ESC` to cancel
- VIM commands work after confirmation dialogs are dismissed

## Future Enhancements

Planned VIM features:

- [ ] Marks (`m{a-z}` to set mark, `'{a-z}` to jump to mark)
- [ ] Visual mode for multi-select
- [ ] Macros for recording command sequences
- [ ] Search highlighting in the list view
- [ ] Regex support in search
- [ ] Filter chaining (multiple search criteria)
- [ ] Command history (`:` followed by `↑`/`↓`)
- [ ] Customizable keybindings via config file

## Comparison with Other Tools

### vs k9s
lazyaws VIM bindings are heavily inspired by k9s:
- Same command mode structure (`:q`, `:r`, etc.)
- Similar navigation patterns
- Context-sensitive help

### vs kubectl with VIM
- Familiar `j`/`k` navigation
- `/` for search
- `g`/`G` for jump to top/bottom

### vs AWS CLI
VIM bindings provide:
- Faster navigation than CLI paging
- Visual feedback
- Bulk operations without scripting
- Interactive search and filter

## Feedback

Found a bug or have a suggestion for VIM keybindings? Please open an issue on GitHub!
