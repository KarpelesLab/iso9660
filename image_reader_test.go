package iso9660

import (
	"io"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const loremIpsum = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.
`

func TestImageReader(t *testing.T) {
	tz := time.FixedZone("", 3600*2)
	recordTime := time.Date(2018, 07, 25, 22, 01, 02, 0, tz)

	f, err := os.Open("testdata/test.iso")
	assert.NoError(t, err)
	defer f.Close() // nolint: errcheck

	image, err := OpenImage(f)
	if assert.NoError(t, err) {
		assert.Equal(t, 2, len(image.volumeDescriptors))
		assert.Equal(t, volumeTypePrimary, image.volumeDescriptors[0].Header.Type)
		assert.Equal(t, volumeTypeTerminator, image.volumeDescriptors[1].Header.Type)
	}

	rootDir, err := image.RootDir()
	assert.NoError(t, err)
	assert.True(t, rootDir.IsDir())
	assert.Equal(t, string([]byte{0}), rootDir.Name())
	assert.Equal(t, int64(sectorSize), rootDir.Size())
	assert.Equal(t, recordTime, rootDir.ModTime())

	children, err := rootDir.GetChildren()
	assert.NoError(t, err)
	assert.Len(t, children, 4)

	cicero := children[0]
	dir1 := children[1]
	dir2 := children[2]
	dir4 := children[3]

	assert.Equal(t, "CICERO.TXT", cicero.Name())
	assert.Equal(t, int64(845), cicero.Size())

	if assert.Equal(t, "DIR1", dir1.Name()) {
		dir1Children, err := dir1.GetChildren()
		assert.NoError(t, err)
		assert.Len(t, dir1Children, 1)

		loremFile := dir1Children[0]

		assert.Equal(t, "LOREM_IP.TXT", loremFile.Name())
		assert.Equal(t, int64(446), loremFile.Size())

		data, err := io.ReadAll(loremFile.Reader())
		assert.NoError(t, err)

		assert.Equal(t, loremIpsum, string(data))
	}

	if assert.Equal(t, "DIR2", dir2.Name()) {
		dir2Children, err := dir2.GetChildren()
		assert.NoError(t, err)
		assert.Len(t, dir2Children, 2)

		assert.Equal(t, "DIR3", dir2Children[0].Name())
		dir3Children, err := dir2Children[0].GetChildren()
		assert.NoError(t, err)
		assert.Len(t, dir3Children, 1)

		assert.Equal(t, "DATA.BIN", dir3Children[0].Name())
		assert.Equal(t, int64(512), dir3Children[0].Size())

		assert.Equal(t, "LARGE.TXT", dir2Children[1].Name())
		assert.Equal(t, int64(2808), dir2Children[1].Size())
		assert.False(t, dir2Children[1].IsDir())
	}

	if assert.Equal(t, "DIR4", dir4.Name()) {
		dir4Children, err := dir4.GetChildren()
		assert.NoError(t, err)
		assert.Len(t, dir4Children, 1000)
		assert.Equal(t, "FILE1012", dir4Children[12].Name())
	}
}

func TestImageReaderLabel(t *testing.T) {
	f, err := os.Open("testdata/test.iso")
	assert.NoError(t, err)
	defer f.Close()

	image, err := OpenImage(f)
	assert.NoError(t, err)

	label, err := image.Label()
	assert.NoError(t, err)
	assert.Equal(t, "my-vol-id", label)
}

func TestImageReaderNoVolumes(t *testing.T) {
	imageWithoutDescriptors := Image{
		ra:                nil,
		volumeDescriptors: []volumeDescriptor{},
	}

	_, err := imageWithoutDescriptors.RootDir()
	assert.Error(t, err)

	_, err = imageWithoutDescriptors.Label()
	assert.Error(t, err)
}

func TestFileMode(t *testing.T) {
	f := File{
		ra: nil,
		de: &DirectoryEntry{
			FileFlags: dirFlagDir,
		},
		children: []*File{},
	}

	assert.Nil(t, f.Reader())
	assert.Equal(t, os.ModeDir, f.Mode())
}

func TestImageReaderRockRidge(t *testing.T) {
	f, err := os.Open("testdata/test_rockridge.iso")
	assert.NoError(t, err)
	defer f.Close()

	image, err := OpenImage(f)
	assert.NoError(t, err)

	// root dir
	rootDir, err := image.RootDir()
	assert.NoError(t, err)

	rootDot, err := rootDir.GetDotEntry()
	assert.NoError(t, err)

	ers, err := rootDot.de.SystemUseEntries.GetExtensionRecords()
	assert.NoError(t, err)

	// has Rock Ridge
	if assert.Len(t, ers, 1) {
		assert.Equal(t, "RRIP_1991A", ers[0].Identifier)
		assert.Equal(t, 1, ers[0].Version)
	}

	children, err := rootDir.GetChildren()
	assert.NoError(t, err)
	assert.Len(t, children, 5)

	// symlink
	symlink := children[4]
	assert.Equal(t, "this-is-a-symlink", symlink.Name())
	assert.Equal(t, os.ModeSymlink, symlink.Mode()&os.ModeSymlink)

	dir1 := children[1]
	assert.Equal(t, "dir1", dir1.Name())

	dir1Children, err := dir1.GetChildren()
	assert.NoError(t, err)
	assert.Len(t, dir1Children, 1)

	// lorem ipsum with Rock Ridge long filename and permissions
	loremFile := dir1Children[0]
	assert.Equal(t, "lorem_ipsum.txt", loremFile.Name())
	assert.Equal(t, int64(446), loremFile.Size())
	assert.Equal(t, fs.FileMode(0640), loremFile.Mode().Perm(), "expected mode %o, got %o", 0640, loremFile.Mode().Perm())
	assert.NotNil(t, loremFile.susp)
	assert.True(t, loremFile.susp.HasRockRidge)

	data, err := io.ReadAll(loremFile.Reader())
	assert.NoError(t, err)
	assert.Equal(t, loremIpsum, string(data))

	// verify SUSP entries are parsed
	assert.Len(t, loremFile.de.SystemUseEntries, 4)
	assert.Equal(t, "RR", loremFile.de.SystemUseEntries[0].Type())
	assert.Equal(t, "PX", loremFile.de.SystemUseEntries[2].Type())
	assert.Equal(t, "TF", loremFile.de.SystemUseEntries[3].Type())
}

func TestGetAllChildren(t *testing.T) {
	f, err := os.Open("testdata/test.iso")
	assert.NoError(t, err)
	defer f.Close()

	image, err := OpenImage(f)
	assert.NoError(t, err)

	rootDir, err := image.RootDir()
	assert.NoError(t, err)

	// GetAllChildren should include . and .. entries
	allChildren, err := rootDir.GetAllChildren()
	assert.NoError(t, err)

	// GetChildren should exclude . and .. entries
	children, err := rootDir.GetChildren()
	assert.NoError(t, err)

	// All children should have 2 more entries (. and ..)
	assert.Equal(t, len(allChildren), len(children)+2)
}

func TestFileNotDirectory(t *testing.T) {
	f, err := os.Open("testdata/test.iso")
	assert.NoError(t, err)
	defer f.Close()

	image, err := OpenImage(f)
	assert.NoError(t, err)

	rootDir, err := image.RootDir()
	assert.NoError(t, err)

	children, err := rootDir.GetChildren()
	assert.NoError(t, err)

	// cicero.txt is a file, not a directory
	cicero := children[0]
	assert.False(t, cicero.IsDir())

	_, err = cicero.GetChildren()
	assert.Error(t, err)
}
