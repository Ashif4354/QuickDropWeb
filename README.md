# QuickDrop Web

QuickDrop Web is a lightweight, locally hosted web application designed for fast and secure file sharing between devices on the same network. Simply drag and drop a file on your computer, scan the generated QR code with your phone (or another device), and download it instantly.

The file link is **one-time use only**—it self-destructs immediately after the download triggers, ensuring your files don't linger.

## Features

*   **Drag & Drop Interface**: Simple, intuitive UI for uploading files.
*   **One-Time Downloads**: Links are valid for a single discharge only. Once downloaded, the file is deleted from the server.
*   **QR Code Sharing**: Instantly generates a QR code pointing to the local network address of the file.
*   **Automatic Cleanup**: Files that are not downloaded are automatically removed after 1 hour.
*   **Neumorphic Design**: A sleek, modern dark-mode interface.
*   **Single Binary**: Can be built into a single executable with embedded static assets.

## Prerequisites

*   [Go](https://golang.org/dl/) (version 1.18 or higher)

## How to Run

1.  **Clone the repository** (or download the source):
    ```bash
    git clone https://github.com/Ashif/QuickDropWeb.git
    cd QuickDropWeb
    ```

2.  **Install dependencies**:
    ```bash
    go mod tidy
    ```

3.  **Run the application**:
    ```bash
    go run main.go
    ```

    The application will start the server on port `8989` and automatically try to open your default web browser to the correct URL (e.g., `http://192.168.1.5:8989`).

## Building for Distribution

To build a standalone executable:

```bash
go build -o quickdrop-web
```

Then you can run it directly:

```bash
./quickdrop-web
```

## Usage

1.  Run the application on your computer.
2.  Drag and drop a file onto the "Drop your file here" zone.
3.  Wait for the upload to complete.
4.  A QR code will appear.
5.  Scan the QR code with your mobile device or open the link on another computer.
6.  The download will start, and the file will be deleted from the host immediately.
