# OneDrive Client

A command-line interface (CLI) client for Microsoft OneDrive that provides comprehensive file management capabilities, including upload, download, search, and access to shared content.

## Features

- **Authentication**: OAuth2 device code flow authentication
- **File Operations**: Upload, download, list, create folders
- **File Management**: Copy, move, rename, delete files and folders
- **Search**: Search across your OneDrive by query string
- **Recent Items**: List recently accessed files
- **Shared Content**: View items shared with you from other users
- **Special Folders**: Access well-known folders (Documents, Photos, Music, etc.)
- **Drive Management**: List available drives and check quota
- **Resumable Uploads**: Large file uploads with session management

## Installation

### Build from Source

```bash
git clone <repository-url>
cd onedrive-client
go build -o onedrive-client .
```

## Authentication

Before using the client, you need to authenticate with Microsoft:

```bash
# Start authentication flow
./onedrive-client auth login

# Check authentication status
./onedrive-client auth status

# Logout (clear stored credentials)
./onedrive-client auth logout
```

## Usage

### Basic File Operations

```bash
# List files in root directory
./onedrive-client files list

# List files in a specific directory
./onedrive-client files list "Documents/Projects"

# Get detailed information about a file or folder
./onedrive-client files stat "Documents/myfile.txt"

# Create a new folder
./onedrive-client files mkdir "NewFolder"

# Upload a file
./onedrive-client files upload /local/path/file.txt "RemoteFolder/"

# Download a file
./onedrive-client files download "RemoteFolder/file.txt" /local/path/

# Delete a file or folder
./onedrive-client files rm "RemoteFolder/file.txt"
```

### Advanced File Operations

```bash
# Copy a file or folder
./onedrive-client files copy "source.txt" "Documents/" "new-name.txt"

# Move a file or folder
./onedrive-client files mv "source.txt" "Documents/"

# Rename a file or folder
./onedrive-client files rename "old-name.txt" "new-name.txt"

# Check copy operation status
./onedrive-client files copy-status <monitor-url>
```

### Search and Discovery

```bash
# Search for files and folders by name or content
./onedrive-client files search "project report"

# List recently accessed files
./onedrive-client files recent

# Access special folders
./onedrive-client files special documents
./onedrive-client files special photos
./onedrive-client files special music
```

### Shared Content

```bash
# List items shared with you from other users
./onedrive-client shared list
```

### Drive Management

```bash
# List all available drives
./onedrive-client drives list

# Show drive quota information
./onedrive-client drives quota
```

### Upload Sessions (for Large Files)

```bash
# Upload large files with resumable sessions
./onedrive-client files upload /path/to/large-file.zip "Backups/"

# Check upload session status
./onedrive-client files get-upload-status <upload-session-url>

# Cancel an active upload session
./onedrive-client files cancel-upload <upload-session-url>
```

## Special Folders

The client supports accessing OneDrive's special folders:

- `documents` - Documents folder
- `photos` - Photos/Pictures folder  
- `cameraroll` - Camera Roll (Business accounts)
- `approot` - App Root folder (Business accounts)
- `music` - Music folder
- `recordings` - Recordings folder (Business accounts)

## Command Reference

### Authentication Commands
- `auth login` - Start OAuth2 authentication flow
- `auth status` - Check current authentication status
- `auth logout` - Clear stored credentials

### File Commands
- `files list [path]` - List directory contents
- `files stat <path>` - Get file/folder metadata
- `files mkdir <path>` - Create directory
- `files upload <local-file> [remote-path]` - Upload file
- `files download <remote-path> [local-path]` - Download file
- `files rm <path>` - Delete file/folder
- `files copy <source> <destination> [new-name]` - Copy file/folder
- `files mv <source> <destination>` - Move file/folder
- `files rename <path> <new-name>` - Rename file/folder
- `files search <query>` - Search files and folders
- `files recent` - List recently accessed items
- `files special <folder-name>` - Access special folders

### Shared Content Commands
- `shared list` - List items shared with you

### Drive Commands
- `drives list` - List available drives
- `drives quota` - Show drive quota information

### Upload Session Commands
- `files get-upload-status <url>` - Check upload progress
- `files cancel-upload <url>` - Cancel upload session

## Global Flags

- `--debug` - Enable debug logging for troubleshooting

## Examples

### Upload and organize files
```bash
# Create a project folder
./onedrive-client files mkdir "Projects/MyProject"

# Upload project files
./onedrive-client files upload README.md "Projects/MyProject/"
./onedrive-client files upload src.zip "Projects/MyProject/"

# Verify upload
./onedrive-client files list "Projects/MyProject"
```

### Search and access recent work
```bash
# Find all documents containing "budget"
./onedrive-client files search "budget"

# Check what you worked on recently
./onedrive-client files recent

# Access your documents folder directly
./onedrive-client files special documents
```

### Manage shared content
```bash
# See what others have shared with you
./onedrive-client shared list

# Copy shared file to your documents
./onedrive-client files copy "SharedFile.xlsx" "Documents/" "MySheet.xlsx"
```

## Status

This project is currently in alpha development. Core functionality is working, but some features may have limitations:

- **Search**: May return limited results depending on OneDrive indexing
- **Shared Items**: Access may be restricted on personal OneDrive accounts
- **Recent Items**: Results depend on OneDrive activity tracking
- **Error Handling**: Some edge cases may not be fully handled

## Contributing

This is an open-source project. Contributions, bug reports, and feature requests are welcome.