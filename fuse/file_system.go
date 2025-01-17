package fuse

import (
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/superfly/litefs"
)

var _ fs.FS = (*FileSystem)(nil)
var _ fs.FSStatfser = (*FileSystem)(nil)
var _ litefs.Invalidator = (*FileSystem)(nil)

// FileSystem represents a raw interface to the FUSE file system.
type FileSystem struct {
	path string // mount path

	store *litefs.Store

	conn   *fuse.Conn
	server *fs.Server
	root   *RootNode

	// User & Group ID for all files in the filesystem.
	Uid int
	Gid int

	// If set, function is called for each FUSE request & response.
	Debug func(msg any)
}

// NewFileSystem returns a new instance of FileSystem.
func NewFileSystem(path string, store *litefs.Store) *FileSystem {
	fsys := &FileSystem{
		path:  path,
		store: store,

		Uid: os.Getuid(),
		Gid: os.Getgid(),
	}

	fsys.root = newRootNode(fsys)

	return fsys
}

// Path returns the path to the mount point.
func (fsys *FileSystem) Path() string { return fsys.path }

// Store returns the underlying store.
func (fsys *FileSystem) Store() *litefs.Store { return fsys.store }

// Mount mounts the file system to the mount point.
func (fsys *FileSystem) Mount() (err error) {
	fsys.conn, err = fuse.Mount(fsys.path,
		fuse.FSName("litefs"),
		fuse.LockingPOSIX(),
	)
	if err != nil {
		return err
	}

	config := fs.Config{Debug: fsys.Debug}
	fsys.server = fs.New(fsys.conn, &config)

	go func() {
		if err := fsys.server.Serve(fsys); err != nil {
			log.Printf("fuse serve error: %s", err)
		}
	}()

	return nil
}

// Unmount unmounts the file system.
func (fsys *FileSystem) Unmount() (err error) {
	if fsys.conn != nil {
		if e := fuse.Unmount(fsys.path); err == nil {
			err = e
		}
		if e := fsys.conn.Close(); err == nil {
			err = e
		}
		fsys.conn = nil
	}
	return err
}

// Root returns the root directory in the file system.
func (fsys *FileSystem) Root() (fs.Node, error) {
	return fsys.root, nil
}

// InvalidateDB invalidates a database in the kernel page cache.
func (fsys *FileSystem) InvalidateDB(db *litefs.DB, offset, size int64) error {
	node := fsys.root.Node(db.Name())
	if node == nil {
		return nil
	}

	if err := fsys.server.InvalidateNodeDataRange(node, offset, size); err != nil && err != fuse.ErrNotCached {
		return err
	}
	return nil
}

// InvalidatePos invalidates the position file in the kernel page cache.
func (fsys *FileSystem) InvalidatePos(db *litefs.DB) error {
	node := fsys.root.Node(db.Name() + "-pos")
	if node == nil {
		return nil
	}

	if err := fsys.server.InvalidateNodeData(node); err != nil && err != fuse.ErrNotCached {
		return err
	}
	return nil
}
