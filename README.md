## iso9660

[![GoDoc](https://godoc.org/github.com/KarpelesLab/iso9660?status.svg)](https://godoc.org/github.com/KarpelesLab/iso9660)

A Go package for reading and creating ISO9660 images, forked from https://github.com/kdomanski/iso9660.

Requires Go 1.21 or newer.

## Features

### Reading
- Basic ISO9660 support
- Rock Ridge extensions (long filenames, POSIX permissions, symlinks)
- SUSP (System Use Sharing Protocol)

### Writing
- Basic ISO9660 support
- El Torito boot support
- Hard links (same file added multiple times is written once)
- Streaming writes (no temp files needed)

Joliet extensions are not supported.

## Examples

### Extracting an ISO

```go
package main

import (
  "log"
  "os"

  "github.com/KarpelesLab/iso9660/isoutil"
)

func main() {
  f, err := os.Open("/home/user/myImage.iso")
  if err != nil {
    log.Fatalf("failed to open file: %s", err)
  }
  defer f.Close()

  if err = isoutil.ExtractImageToDirectory(f, "/home/user/target_dir"); err != nil {
    log.Fatalf("failed to extract image: %s", err)
  }
}
```

### Reading an ISO with Rock Ridge

```go
package main

import (
  "fmt"
  "log"
  "os"

  "github.com/KarpelesLab/iso9660"
)

func main() {
  f, err := os.Open("/home/user/myImage.iso")
  if err != nil {
    log.Fatalf("failed to open file: %s", err)
  }
  defer f.Close()

  img, err := iso9660.OpenImage(f)
  if err != nil {
    log.Fatalf("failed to open image: %s", err)
  }

  // Get the volume label
  label, _ := img.Label()
  fmt.Printf("Volume: %s\n", label)

  root, err := img.RootDir()
  if err != nil {
    log.Fatalf("failed to get root dir: %s", err)
  }

  children, err := root.GetChildren()
  if err != nil {
    log.Fatalf("failed to get children: %s", err)
  }

  for _, child := range children {
    // With Rock Ridge, Name() returns the full filename
    // Mode() returns POSIX permissions
    fmt.Printf("%s %s (%d bytes)\n", child.Mode(), child.Name(), child.Size())
  }
}
```

### Creating an ISO

```go
package main

import (
  "log"
  "os"

  "github.com/KarpelesLab/iso9660"
)

func main() {
  writer, err := iso9660.NewWriter()
  if err != nil {
    log.Fatalf("failed to create writer: %s", err)
  }

  // Set volume name
  writer.Primary.VolumeIdentifier = "testvol"

  // Add a single file
  err = writer.AddLocalFile("/home/user/myFile.txt", "folder/MYFILE.TXT")
  if err != nil {
    log.Fatalf("failed to add file: %s", err)
  }

  // Or add an entire directory recursively
  err = writer.AddLocalDirectory("/home/user/myFolder", "folder")
  if err != nil {
    log.Fatalf("failed to add directory: %s", err)
  }

  outputFile, err := os.OpenFile("/home/user/output.iso", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
  if err != nil {
    log.Fatalf("failed to create file: %s", err)
  }

  _, err = writer.WriteTo(outputFile)
  if err != nil {
    log.Fatalf("failed to write ISO image: %s", err)
  }

  err = outputFile.Close()
  if err != nil {
    log.Fatalf("failed to close output file: %s", err)
  }
}
```

### Creating a Bootable ISO

```go
package main

import (
  "log"
  "os"

  "github.com/KarpelesLab/iso9660"
)

func main() {
  writer, err := iso9660.NewWriter()
  if err != nil {
    log.Fatalf("failed to create writer: %s", err)
  }

  writer.Primary.VolumeIdentifier = "BOOTABLE"

  // Add El Torito boot entry
  isolinux, err := iso9660.NewItemFile("/usr/share/syslinux/isolinux.bin")
  if err != nil {
    log.Fatalf("failed to open isolinux.bin: %s", err)
  }

  err = writer.AddBootEntry(&iso9660.BootCatalogEntry{BootInfoTable: true}, isolinux, "isolinux/isolinux.bin")
  if err != nil {
    log.Fatalf("failed to add boot entry: %s", err)
  }

  // Add other boot files
  writer.AddLocalFile("/usr/share/syslinux/ldlinux.c32", "isolinux/ldlinux.c32")

  outputFile, err := os.Create("/home/user/bootable.iso")
  if err != nil {
    log.Fatalf("failed to create file: %s", err)
  }

  _, err = writer.WriteTo(outputFile)
  if err != nil {
    log.Fatalf("failed to write ISO image: %s", err)
  }

  outputFile.Close()
}
```

### Streaming an ISO via HTTP

It is possible to stream a dynamically generated ISO via HTTP:

```go
package main

import (
  "net/http"
  "log"

  "github.com/KarpelesLab/iso9660"
)

func handler(rw http.ResponseWriter, req *http.Request) {
  writer, err := iso9660.NewWriter()
  if err != nil {
    http.Error(rw, err.Error(), 500)
    return
  }

  writer.Primary.VolumeIdentifier = "LIVE IMAGE"

  // Add files dynamically
  writer.AddLocalFile("kernel.img", "boot/kernel.img")
  writer.AddLocalFile("initrd.img", "boot/initrd.img")

  rw.Header().Set("Content-Type", "application/x-iso9660-image")
  rw.Header().Set("Content-Disposition", "attachment; filename=image.iso")

  writer.WriteTo(rw)
}

func main() {
  http.HandleFunc("/image.iso", handler)
  log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## API

### Reader

- `OpenImage(ra io.ReaderAt) (*Image, error)` - Open an ISO image for reading
- `(*Image) RootDir() (*File, error)` - Get the root directory
- `(*Image) Label() (string, error)` - Get the volume label
- `(*File) GetChildren() ([]*File, error)` - Get directory children (excludes `.` and `..`)
- `(*File) GetAllChildren() ([]*File, error)` - Get all directory children (includes `.` and `..`)
- `(*File) Reader() io.Reader` - Get a reader for file contents
- `(*File) Name() string` - Get filename (Rock Ridge long name if available)
- `(*File) Mode() os.FileMode` - Get file mode (Rock Ridge permissions if available)
- `(*File) IsDir() bool` - Check if entry is a directory
- `(*File) Size() int64` - Get file size

### Writer

- `NewWriter() (*ImageWriter, error)` - Create a new ISO writer
- `(*ImageWriter) AddFile(data io.Reader, filePath string) error` - Add a file from a reader
- `(*ImageWriter) AddLocalFile(localPath, filePath string) error` - Add a file from the filesystem
- `(*ImageWriter) AddLocalDirectory(origin, target string) error` - Add a directory recursively
- `(*ImageWriter) AddBootEntry(boot *BootCatalogEntry, data Item, filePath string) error` - Add El Torito boot entry
- `(*ImageWriter) WriteTo(w io.Writer) (int64, error)` - Write the ISO image
