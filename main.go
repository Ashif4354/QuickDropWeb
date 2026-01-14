package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

//go:embed static/*
var staticFiles embed.FS

// FileMetadata stores path and creation time
type FileMetadata struct {
	Path      string
	CreatedAt time.Time
}

// Store for managing active files
type FileStore struct {
	sync.RWMutex
	Files map[string]FileMetadata
}

var store = FileStore{
	Files: make(map[string]FileMetadata),
}

const Port = "8989"

func main() {
	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)
	
	r := gin.Default()

	// Serve embedded static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	r.StaticFS("/static", http.FS(staticFS))
	
	// Root redirects to index.html (served by StaticFS above if we mount it right, 
	// but StaticFS usually mounts a directory. Let's make a specific route for root)
	r.GET("/", func(c *gin.Context) {
		f, err := staticFS.Open("index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Internal Server Error")
			return
		}
		defer f.Close()
		content, err := io.ReadAll(f)
		if err != nil {
			c.String(http.StatusInternalServerError, "Internal Server Error")
			return
		}
		c.Data(http.StatusOK, "text/html", content)
	})

	// Routes
	r.POST("/upload", handleUpload)
	r.GET("/download/:token", handleDownload)
	r.GET("/status/:token", handleStatus)
	r.GET("/qr/:token", handleQR)

	// Determine Local IP
	ip := getLocalIP()
	url := fmt.Sprintf("http://%s:%s", ip, Port)

	fmt.Println("=======================================")
	fmt.Printf("QuickDrop Web is running at: %s\n", url)
	fmt.Println("OPEN THIS URL IN YOUR BROWSER")
	fmt.Println("=======================================")

	// Open Browser
	openBrowser(url)

	// Periodically clean up old files (optional, but good practice if they are never downloaded)
	// For this prototype, we rely on the download trigger, but a GC loop is good.
	go runGC()

	r.Run(":" + Port)
}

func handleUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Create temp file
	// Create uploads dir
	tempDir := "uploads"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload dir"})
		return
	}
	ext := filepath.Ext(file.Filename)
	filename := uuid.New().String() + ext
	savePath := filepath.Join(tempDir, filename)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		fmt.Printf("Error saving file to %s: %v\n", savePath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save file: %v", err)})
		return
	}

	// Generate Token
	token := uuid.New().String()
	
	store.Lock()
	store.Files[token] = FileMetadata{
		Path:      savePath,
		CreatedAt: time.Now(),
	}
	store.Unlock()

	ip := getLocalIP()
	downloadURL := fmt.Sprintf("http://%s:%s/download/%s", ip, Port, token)
	qrURL := fmt.Sprintf("/qr/%s", token)

	c.JSON(http.StatusOK, gin.H{
		"token":  token,
		"url":    downloadURL,
		"qr_url": qrURL,
	})
}

func handleDownload(c *gin.Context) {
	token := c.Param("token")

	store.RLock()
	meta, exists := store.Files[token]
	store.RUnlock()

	if !exists {
		c.String(http.StatusNotFound, "File not found or already destroyed.")
		return
	}

	// Stream file to user
	// We need to hook into the completion to delete it. 
	// Gin doesn't have a simple "OnComplete" callback for c.File, 
	// but we can manually serve content.

	f, err := os.Open(meta.Path)
	if err != nil {
		c.String(http.StatusNotFound, "File missing on disk.")
		return
	}
	defer f.Close()

	// Set headers
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(meta.Path))
	c.Header("Content-Type", "application/octet-stream")

	// Copy to response writer
	_, err = io.Copy(c.Writer, f)
	if err != nil {
		// Log error but we probably can't change status code now
		fmt.Println("Error streaming file:", err)
		return
	}

	// SELF DESTRUCT
	go func() {
		// Small delay to ensure connection closes cleanly? Not strictly needed but safe.
		// Actually, let's just delete it.
		store.Lock()
		delete(store.Files, token)
		store.Unlock()

		os.Remove(meta.Path)
		fmt.Printf("Destroyed file for token: %s\n", token)
	}()
}

func handleStatus(c *gin.Context) {
	token := c.Param("token")

	store.RLock()
	_, exists := store.Files[token]
	store.RUnlock()

	if exists {
		c.Status(http.StatusOK)
	} else {
		c.Status(http.StatusNotFound)
	}
}

func handleQR(c *gin.Context) {
	token := c.Param("token")
	
	// Verify it exists first
	store.RLock()
	_, exists := store.Files[token]
	store.RUnlock()

	if !exists {
		c.Status(http.StatusNotFound)
		return
	}

	ip := getLocalIP()
	url := fmt.Sprintf("http://%s:%s/download/%s", ip, Port, token)

	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Data(http.StatusOK, "image/png", png)
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}
	for _, address := range addrs {
		// Check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "localhost"
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Println("Error opening browser:", err)
	}
}

func runGC() {
	for {
		time.Sleep(1 * time.Minute)
		now := time.Now()
		store.Lock()
		for token, meta := range store.Files {
			if now.Sub(meta.CreatedAt) > 1 * time.Hour {
				// Expire after 1 hour if not downloaded
				os.Remove(meta.Path)
				delete(store.Files, token)
				fmt.Printf("GC: Removed expired token %s\n", token)
			}
		}
		store.Unlock()
	}
}
