# goback

Automated backup system written in Go.

## Building

### Local build

```bash
go build -o goback .
``` 

## Usage

### Basic usage

```bash
./goback [config.yaml]
```

If the config path is not specified, `config.yaml` in the current directory is used.

### Command-line flags

- `-config`, `-c` - Path to configuration file (default: `config.yaml`)
- `-backup`, `-b` - Name of backup to run (can be specified multiple times)
- `--skip-global-pre-hooks`, `--skip-pre-hooks` - Skip global pre-hooks execution
- `--skip-global-post-hooks`, `--skip-post-hooks` - Skip global post-hooks execution

### Examples

```bash
# Run all backups from config.yaml
./goback

# Run specific backup
./goback -b backup-name

# Run multiple specific backups
./goback -b backup1 -b backup2 -b backup3

# Use custom config file
./goback -config /path/to/config.yaml

# Run specific backup with custom config
./goback -c custom.yaml -b backup-name

# Skip global hooks
./goback --skip-global-pre-hooks --skip-global-post-hooks

# Positional arguments (for backward compatibility)
./goback config.yaml backup-name
./goback backup-name backup-name2
```

## Configuration

The tool uses a YAML configuration file to set up backups. See `config-example.yaml` for a detailed example with all available options and their descriptions.

### Minimal configuration example

```yaml
global:
  backup_dir: "/var/www/backups"
  filename_mask: "%name%-%Y%m%d%H%M%S"
  default_compression: "gzip"
  retention:
    daily: 2
    weekly: 2
    monthly: 2
    yearly: 2

backups:
  - name: "my-backup"
    subdirectory: "project"
    source_dir: "/var/www/project"
```

For complete configuration documentation with all available options, see `config-example.yaml`.

## Features

- Directory backups with exclusion patterns
- Command-based backups (e.g., database dumps)
- Multiple compression types: gzip, zip, tar, tar.gz, none
- Retention policy based on anchor points (daily, weekly, monthly, yearly)
- Pre/post hooks for executing commands before and after backups
- Automatic loading of backup configs from include_dir
- Selective backup execution by name
- Global hooks control

## Project structure

```
goback/
├── main.go                 # Entry point
├── config/                 # Configuration parsing
├── backup/                 # Backup execution logic
├── compression/            # Compression support
├── retention/              # Retention policy
├── hooks/                   # Hook execution
└── utils/                   # Utilities
```
