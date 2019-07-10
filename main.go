package main

import (
	"context"
	"fmt"
	"math"
	"sync"
	"syscall"
	"time"

	"path/filepath"

	"github.com/hanwen/go-fuse/fs"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

// indexedFile is a manageable discrete unit of info for a particular entry
// used to keep things organized.
type indexedFile struct {
	Filename    string
	IsDirectory bool
	Timestamp   time.Time
	Inode       uint64
	Size        uint64

	Data []byte
}

func (file *indexedFile) setAttributes(out *fuse.Attr) {
	out.SetTimes(&file.Timestamp, &file.Timestamp, &file.Timestamp)

	out.Mode = file.mode()
	out.Ino = file.Inode

	out.Size = file.size()
	out.Blksize = 1024
	out.Blocks = uint64(math.Ceil(float64(out.Size / uint64(out.Blksize))))

	out.Uid = 1000
	out.Gid = 1000
}

func (file *indexedFile) size() uint64 {
	if file.IsDirectory == true {
		// Just a small consistent constant size to show for all directories.
		return 10
	} else {
		return uint64(len(file.Data))
	}
}

func (file *indexedFile) mode() uint32 {
	if file.IsDirectory == true {
		return 0755 | uint32(syscall.S_IFDIR)
	} else {
		return 0644 | uint32(syscall.S_IFREG)
	}
}

var (
	// fileIndex maps file-paths to `indexedFile` structs. It's a flat
	// structure. Only the root entry is populated statically. The other
	// entries are driven by `tree`.
	fileIndex = map[string]*indexedFile{
		"/": &indexedFile{
			Filename:    "",
			IsDirectory: true,
			Timestamp:   time.Now(),
			Inode:       1001,
		},
	}

	// tree expresses the hierarchy of the filesystem. The flat `fileIndex`
	// lookup index is established from this.
	tree = map[string][]*indexedFile{
		"/": []*indexedFile{
			&indexedFile{
				Filename:    "subdirectory1",
				IsDirectory: true,
				Timestamp:   time.Now(),
				Inode:       1002,
			},

			&indexedFile{
				Filename:  "file1",
				Timestamp: time.Now(),
				Inode:     11,
				Data:      []byte("test content 1\r\n"),
			},
			&indexedFile{
				Filename:  "file2",
				Timestamp: time.Now(),
				Inode:     22,
				Data:      []byte("test content 2\r\n"),
			},
			&indexedFile{
				Filename:  "file3",
				Timestamp: time.Now(),
				Inode:     33,
				Data:      []byte("test content 3\r\n"),
			},
		},
		"/subdirectory1": []*indexedFile{
			&indexedFile{
				Filename:  "file4",
				Timestamp: time.Now(),
				Inode:     44,
				Data:      []byte("test content 4\r\n"),
			},
			&indexedFile{
				Filename:  "file5",
				Timestamp: time.Now(),
				Inode:     55,
				Data:      []byte("test content 5\r\n"),
			},
			&indexedFile{
				Filename:  "file6",
				Timestamp: time.Now(),
				Inode:     66,
				Data:      []byte("test content 6\r\n"),
			},
		},
	}
)

// HelloNode describes a single entry/inode in the filesystem.
type HelloNode struct {
	fs.Inode
	path string
}

// NewHelloNode returns a new HelloNode struct.
func NewHelloNode(path string) fs.InodeEmbedder {
	return &HelloNode{
		path: path,
	}
}

func (hn *HelloNode) currentPath() string {
	path := hn.Path(nil)

	root := hn.Root().Operations().(*HelloNode)
	return filepath.Join(root.path, path)
}

func (hn *HelloNode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	path := hn.currentPath()
	fmt.Printf("Getattr: [%s]\n", path)

	entry, found := fileIndex[path]
	if found == false {
		fmt.Printf("Not found.\n")
		return syscall.ENOENT
	}

	entry.setAttributes(&out.Attr)

	return fs.OK
}

// Opendir just validates the existence of a directory (in our use-case).
func (hn *HelloNode) Opendir(ctx context.Context) syscall.Errno {
	path := hn.currentPath()
	fmt.Printf("Opendir: [%s]\n", path)

	if _, found := tree[path]; found == false {
		return syscall.ENOENT
	}

	return fs.OK
}

// Readdir returns a list of file entries in the current path.
func (hn *HelloNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	path := hn.currentPath()
	fmt.Printf("Readdir: [%s]\n", path)

	files, found := tree[path]
	if found == false {
		return nil, syscall.ENOENT
	}

	entries := make([]fuse.DirEntry, len(files))
	for i, file := range files {
		entries[i] = fuse.DirEntry{
			Name: file.Filename,
			Mode: file.mode(),
			Ino:  file.Inode,
		}
	}

	ds := fs.NewListDirStream(entries)

	return ds, fs.OK
}

// Lookup returns the attributes for a given file. This is required for the stat
// info in the-listings.
func (hn *HelloNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (childNode *fs.Inode, errno syscall.Errno) {
	childPath := filepath.Join(hn.currentPath(), name)
	fmt.Printf("Lookup: [%s]\n", childPath)

	entry, found := fileIndex[childPath]
	if found == false {
		fmt.Printf("Not found.\n")
		return nil, syscall.ENOENT
	}

	entry.setAttributes(&out.Attr)

	childHelloNode := NewHelloNode(childPath)

	sa := fs.StableAttr{
		Mode: entry.mode(),
		Gen:  1,
		Ino:  entry.Inode,
	}

	childNode = hn.NewInode(ctx, childHelloNode, sa)

	return childNode, fs.OK
}

// Read returns the requested bytes. Even though Open returns a managed object,
// this is still required.
func (hn *HelloNode) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	filepath := hn.currentPath()
	fmt.Printf("Lookup: [%s]\n", filepath)

	entry, found := fileIndex[filepath]
	if found == false {
		fmt.Printf("Not found.\n")
		return nil, syscall.ENOENT
	}

	end := int(off) + len(dest)
	if end > len(entry.Data) {
		end = len(entry.Data)
	}
	return fuse.ReadResultData(entry.Data[off:end]), fs.OK
}

// Flush flushs all buffers. It's a no-op for a read-only filesystem. Even
// though Open returns a managed object, this is still required.
func (hn *HelloNode) Flush(ctx context.Context, fh fs.FileHandle) syscall.Errno {
	return fs.OK
}

// Open returns a file struct with the open file.
func (hn *HelloNode) Open(ctx context.Context, mode uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	filepath := hn.currentPath()
	fmt.Printf("Open (%032b): %s\n", mode, filepath)

	if mode&syscall.S_IFREG == 0 {
		fmt.Printf("File mode not valid: (%d) != (%d)\n", mode, syscall.S_IFREG)
		return nil, 0, syscall.ENOENT
	}

	entry, found := fileIndex[filepath]
	if found == false {
		fmt.Printf("Not found.\n")
		return nil, 0, syscall.ENOENT
	}

	df := nodefs.NewDataFile(entry.Data)
	rf := nodefs.NewReadOnlyFile(df)

	return rf, 0, 0
}

func main() {
	// virtualRootPath is the root of our virtual structure.
	virtualRootPath := "/"

	hn := NewHelloNode(virtualRootPath)

	sec := time.Second

	opts := &fs.Options{
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	}

	fs := fs.NewNodeFS(hn, opts)

	mountPoint := "/mnt/loop"

	server, err := fuse.NewServer(fs, mountPoint, &opts.MountOptions)
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		server.Serve()
		wg.Done()
	}()

	fmt.Printf("Unmount to terminate.\n")
	fmt.Printf("\n")

	wg.Wait()
}

func init() {
	for parentPath, children := range tree {
		for _, child := range children {
			fullPath := filepath.Join(parentPath, child.Filename)
			fileIndex[fullPath] = child
		}
	}
}
