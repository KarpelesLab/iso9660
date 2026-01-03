package iso9660

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMangleDirectoryName(t *testing.T) {
	for _, testcase := range []struct {
		input  string
		output string
	}{
		{
			input:  "ThisStringIsFarTooLongToBeWritten",
			output: "THISSTRINGISFARTOOLONGTOBEWRITT",
		},
		{
			input:  "ThisStringHasUnicodeCharacterŁ",
			output: "THISSTRINGHASUNICODECHARACTER__",
		},
		{
			input:  "ThisStringHasItByteBeforeThEndŁ",
			output: "THISSTRINGHASITBYTEBEFORETHEND_",
		},
	} {
		t.Run(testcase.input, func(t *testing.T) {
			o := mangleDirectoryName(testcase.input)
			assert.Equal(t, testcase.output, o)
		})
	}
}

func TestMangleFileName(t *testing.T) {
	for _, testcase := range []struct {
		input  string
		output string
	}{
		{
			input:  "ThisStringIsFarTooLongToBeWritten",
			output: "THISSTRINGISFARTOOLONGTOBEWR;1",
		},
		{
			input:  "ThisStringHasUnicodeCharacŁ",
			output: "THISSTRINGHASUNICODECHARAC__;1",
		},
		{
			input:  "ThisStringHasAFileExtensionAndItIsVery.Long",
			output: "THISSTRINGHASAFILEEXTEN.LONG;1",
		},
		{
			input:  "ThisStringHasAFileExtensionThats.FarTooLong",
			output: "THISSTRINGHASAFILEE.FARTOOLO;1",
		},
	} {
		t.Run(testcase.input, func(t *testing.T) {
			o := mangleFileName(testcase.input)
			assert.Equal(t, testcase.output, o)
		})
	}
}

/*
TODO: re-enable test
func TestWriterStaging(t *testing.T) {
	w, err := NewWriter()
	assert.NoError(t, err)

	testFileContents := "hrh2309hr320h"
	testFilePath := "FarTooLongFilePathThatWillBeTrimmed/dirø1/somefile.dat"
	testFileMangledPath := "FARTOOLONGFILEPATHTHATWILLBETRI/DIR__1/SOMEFILE.DAT;1"

	r := strings.NewReader(testFileContents)
	err = w.AddFile(r, testFilePath)
	assert.NoError(t, err)

	readData, err := os.ReadFile(path.Join(w.stagingDir, testFileMangledPath))
	assert.NoError(t, err)

	assert.Equal(t, testFileContents, string(readData))
}
*/

func TestWriter(t *testing.T) {
	w, err := NewWriter()
	assert.NoError(t, err)

	err = w.AddFile(strings.NewReader("hrh2309hr320h"), "someDirectoryPath/dir1/somefile.dat")
	assert.NoError(t, err)

	largeFileData, err := os.ReadFile("testdata/test.iso_source/dir2/large.txt")
	assert.NoError(t, err)

	err = w.AddLocalFile("testdata/test.iso_source/dir2/large.txt", "anotherDir/large.txt")
	assert.NoError(t, err)

	f, err := os.CreateTemp(os.TempDir(), "iso9660_golang_test")
	assert.NoError(t, err)
	defer os.Remove(f.Name())

	imageFileName := f.Name()

	_, err = w.WriteTo(f)
	assert.NoError(t, err)

	f.Close() // nolint: errcheck

	f, err = os.Open(imageFileName)
	assert.NoError(t, err)

	img, err := OpenImage(f)
	assert.NoError(t, err)

	root, err := img.RootDir()
	assert.NoError(t, err)

	children, err := root.GetChildren()
	assert.NoError(t, err)
	assert.Len(t, children, 2)
	assert.Equal(t, "ANOTHERDIR", children[0].Name())

	children, err = children[0].GetChildren()
	assert.NoError(t, err)
	assert.Len(t, children, 1)
	assert.Equal(t, "LARGE.TXT", children[0].Name())

	readData, err := io.ReadAll(children[0].Reader())
	assert.NoError(t, err)

	assert.Equal(t, largeFileData, readData)
}
