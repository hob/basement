# MKV Sync Tool

This Go program helps you synchronize your Matroska video files (`.mkv`) from a network drive to a local directory. It scans a specified remote location for `.mkv` files and documents any that are not already present in your local collection.

## Features

-   Indexes local `.mkv` files for efficient comparison.
-   Scans a specified network drive recursively for `.mkv` files.
-   Handles varying directory structures between local and remote locations by comparing filenames.
-   Logs discovered files to missing_files.csv

## How to Use

### 1. Build the Executable

Navigate to the project's root directory in your terminal (where `main.go` is located) and run the following command to compile the program:

```bash
go build -o basement .
```
This will create an executable file named `basement` (or `basement.exe` on Windows) in the current directory.

### 2. Run the Program

Execute the program with the `--local` and `--remote` flags, specifying your local video directory and the network drive path, respectively.

**Example (Windows):**

```bash
.\basement.exe --local "C:\Users\YourUser\Videos\Movies" --remote \\NetworkShare\Media\Movies
```

**Example (Linux/macOS):**

```bash
./basement --local "/home/youruser/Videos/Movies" --remote "/mnt/network/Media/Movies"
```

#### Command-Line Flags:

*   `--local <path>`:
    *   **Required**. The absolute path to your local directory where `.mkv` files are stored.
    *   Example: `"C:\MyVideos"`, `"/home/user/media"`

*   `--remote <path>`:
    *   **Required**. The absolute path to the network drive or remote directory to scan for `.mkv` files.
    *   Example: `"\\Server\Shared\Movies"`, `"/mnt/nas/downloads"`

## Important Notes

*   The program identifies missing files based on their **filename** only. It does not consider the full path.
*   Downloaded files will be placed directly into the specified `--local` directory.
*   The program will output its progress to the console, indicating which files are being indexed, found, and copied.
*   Errors during file access or copying will be logged but will not stop the entire synchronization process.
