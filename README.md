# GoShare - P2P Secure File & Folder Sharing Tool

GoShare is a dual-mode peer-to-peer (P2P) file sharing application built from scratch in Go. It allows users to transfer files, folders, videos, or images directly from one device to another without uploading them to cloud intermediaries like Google Drive or Dropbox.

The project features both a **Command-Line Interface (CLI)** and a **Web Application (WebSocket & Web Crypto)**.

---

## How It Works

```
        SENDER CLIENT                          RECEIVER CLIENT
   (CLI or Web Browser)                     (CLI or Web Browser)
            │                                        │
            │ 1. SHARE <metadata>                    │ 3. DOWNLOAD <code>
            ▼                                        ▼
    ┌───────────────┐                        ┌───────────────┐
    │               │◄───────────────────────┤               │
    │  Relay Server │  4. Relay Metadata     │  Relay Server │
    │ (TCP/Websocket)│                       │ (TCP/Websocket)│
    │               ├───────────────────────►│               │
    └───────────────┘  5. Stream Raw Chunks  └───────────────┘
            ▲                                        │
            │ 2. CODE 123456                         ▼
            └───────────────────────── (Offline Code Share)
```

1. **The Handshake**: The Sender registers the file's metadata (name, size, SHA-256 hash) with the central signaling server. The server responds with a temporary 6-digit code.
2. **The Connection**: The Receiver connects and enters the 6-digit code. The server pairs both connections together.
3. **The Stream**: The Sender slices the file into binary chunks and streams them to the server, which immediately forwards (relays) them to the Receiver.
4. **The Verification**: The Receiver hashes the incoming stream and verifies the resulting SHA-256 fingerprint matches the original. If there is a mismatch, the file is automatically deleted.

---

## Features

- **Dual Mode**: Stream via raw TCP sockets in your terminal or over WebSockets in a gorgeous dark-mode web browser UI.
- **On-the-Fly Folder Sharing**: Directories are zipped recursively before sharing and unzipped automatically on delivery.
- **Flat Memory Footprint**: Reads and writes files in 4KB/64KB chunks, keeping memory consumption tiny even for multi-gigabyte transfers.
- **SHA-256 Integrity Checks**: Automatically detects and deletes corrupted downloads.
- **Deployment-Ready**: Dockerized for instant hosting on cloud platforms like Fly.io or VPS.

---

## File Structure

```
FileShare/
├── Dockerfile           # Multi-stage Docker packaging configuration
├── main.go              # CLI router and argument parser
├── client/              # Client packages
│   ├── client.go        # CLI network actions (Dial, SHARE, DOWNLOAD handlers)
│   └── chunker.go       # Binary chunking, hashing, zipping, and unzipping helpers
├── server/              # Server packages
│   ├── server.go        # Raw TCP server logic (Multi-thread mutex & maps)
│   └── webserver.go     # HTTP Web server and WebSocket relay logic
└── web/                 # Web Application Frontend
    ├── index.html       # Glassmorphic dashboard markup
    ├── style.css        # Background animations, glows, and layout
    └── app.js           # Browser file slicer, Web Crypto hashing, and WS handlers
```

---

## Getting Started

### Prerequisites
- [Go 1.23.1+](https://go.dev/dl/) installed.

### 1. Running Locally (CLI Mode)

1. **Start the TCP Server** (Terminal 1):
   ```bash
   go run main.go server -port 8080
   ```
2. **Share a File/Folder** (Terminal 2):
   ```bash
   go run main.go share -file path/to/my_file.zip
   ```
   *(Note down the 6-digit code returned).*
3. **Download the File/Folder** (Terminal 3):
   ```bash
   go run main.go download -code <6-digit-code>
   ```

---

### 2. Running Locally (Web Mode)

1. **Start the Web Server**:
   ```bash
   go run main.go server -port 8080 -web
   ```
2. Open your web browser to `http://localhost:8080`.
3. Open a second browser window (or access it from your phone on the same WiFi by typing your computer's local IP address like `http://192.168.1.15:8080`).
4. Select a file on the Sender tab, generate the code, and enter it in the Receiver tab.

---

## What We Learned

- **Binary Chunking**: Breaking files into byte slices (`[]byte` in Go, `ArrayBuffer` in JS) to stream data with minimal RAM overhead.
- **Go Interfaces**: Leveraging `io.Reader` and `io.Writer` to treat local files and network connections interchangeably.
- **Concurrency & Concurrency Safety**: Using goroutines (`go`), thread-safe channels (`chan`), and Mutual Exclusion locks (`sync.Mutex`) to pair clients in real-time.
- **Web APIs**: Slicing file objects in the browser (`file.slice()`) and generating SHA-256 hashes natively using `crypto.subtle.digest`.
