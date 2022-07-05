package dynamicfs_test

import (
	"testing"

	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/fuse/samples"
	"github.com/jacobsa/fuse/samples/dynamicfs"

	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/jacobsa/oglematchers"
	"github.com/jacobsa/ogletest"
)

func TestDynamicFS(t *testing.T) { ogletest.RunTests(t) }

type DynamicFSTest struct {
	samples.SampleTest
}

func init() {
	ogletest.RegisterTestSuite(&DynamicFSTest{})
}

var gCreateTime = time.Date(2017, 5, 4, 14, 53, 10, 0, time.UTC)

func (t *DynamicFSTest) SetUp(ti *ogletest.TestInfo) {
	var err error
	t.Clock.SetTime(gCreateTime)
	t.Server, err = dynamicfs.NewDynamicFS(&t.Clock)
	ogletest.AssertEq(nil, err)
	t.SampleTest.SetUp(ti)
}

func (t *DynamicFSTest) ReadDir_Root() {
	entries, err := fusetesting.ReadDirPicky(t.Dir)
	ogletest.AssertEq(nil, err)
	ogletest.AssertEq(2, len(entries))

	var fi os.FileInfo
	fi = entries[0]
	ogletest.ExpectEq("age", fi.Name())
	ogletest.ExpectEq(0, fi.Size())
	ogletest.ExpectEq(0444, fi.Mode())
	ogletest.ExpectFalse(fi.IsDir())

	fi = entries[1]
	ogletest.ExpectEq("weekday", fi.Name())
	ogletest.ExpectEq(0, fi.Size())
	ogletest.ExpectEq(0444, fi.Mode())
	ogletest.ExpectFalse(fi.IsDir())
}

func (t *DynamicFSTest) ReadDir_NonExistent() {
	_, err := fusetesting.ReadDirPicky(path.Join(t.Dir, "nosuchfile"))

	ogletest.AssertNe(nil, err)
	ogletest.ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file")))
}

func (t *DynamicFSTest) Stat_Age() {
	fi, err := os.Stat(path.Join(t.Dir, "age"))
	ogletest.AssertEq(nil, err)

	ogletest.ExpectEq("age", fi.Name())
	ogletest.ExpectEq(0, fi.Size())
	ogletest.ExpectEq(0444, fi.Mode())
	ogletest.ExpectFalse(fi.IsDir())
	ogletest.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *DynamicFSTest) Stat_Weekday() {
	fi, err := os.Stat(path.Join(t.Dir, "weekday"))
	ogletest.AssertEq(nil, err)

	ogletest.ExpectEq("weekday", fi.Name())
	ogletest.ExpectEq(0, fi.Size())
	ogletest.ExpectEq(0444, fi.Mode())
	ogletest.ExpectFalse(fi.IsDir())
	ogletest.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *DynamicFSTest) Stat_NonExistent() {
	_, err := os.Stat(path.Join(t.Dir, "nosuchfile"))

	ogletest.AssertNe(nil, err)
	ogletest.ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file")))
}

func (t *DynamicFSTest) ReadFile_AgeZero() {
	t.Clock.SetTime(gCreateTime)
	slice, err := ioutil.ReadFile(path.Join(t.Dir, "age"))

	ogletest.AssertEq(nil, err)
	ogletest.ExpectEq("This filesystem is 0 seconds old.", string(slice))
}

func (t *DynamicFSTest) ReadFile_Age1000() {
	t.Clock.SetTime(gCreateTime.Add(1000 * time.Second))
	slice, err := ioutil.ReadFile(path.Join(t.Dir, "age"))

	ogletest.AssertEq(nil, err)
	ogletest.ExpectEq("This filesystem is 1000 seconds old.", string(slice))
}

func (t *DynamicFSTest) ReadFile_WeekdayNow() {
	now := t.Clock.Now()
	// Does simulated clock advance itself by default?
	// Manually set time to ensure it's frozen.
	t.Clock.SetTime(now)
	slice, err := ioutil.ReadFile(path.Join(t.Dir, "weekday"))

	ogletest.AssertEq(nil, err)
	ogletest.ExpectEq(fmt.Sprintf("Today is %s.", now.Weekday().String()), string(slice))
}

func (t *DynamicFSTest) ReadFile_WeekdayCreateTime() {
	t.Clock.SetTime(gCreateTime)
	slice, err := ioutil.ReadFile(path.Join(t.Dir, "weekday"))

	ogletest.AssertEq(nil, err)
	ogletest.ExpectEq(fmt.Sprintf("Today is %s.", gCreateTime.Weekday().String()), string(slice))
}

func (t *DynamicFSTest) ReadFile_AgeUnchangedForHandle() {
	t.Clock.SetTime(gCreateTime.Add(100 * time.Second))
	var err error
	var file *os.File
	file, err = os.Open(path.Join(t.Dir, "age"))
	ogletest.AssertEq(nil, err)

	// Ensure that all reads from the same handle return the contents created at
	// file open time.
	func(file *os.File) {
		defer file.Close()

		var expectedContents string
		var buffer bytes.Buffer
		var bytesRead int64

		expectedContents = "This filesystem is 100 seconds old."
		bytesRead, err = buffer.ReadFrom(file)
		ogletest.AssertEq(nil, err)
		ogletest.ExpectEq(len(expectedContents), bytesRead)
		ogletest.ExpectEq(expectedContents, buffer.String())

		t.Clock.SetTime(gCreateTime.Add(1000 * time.Second))
		// Seek back to the beginning of the file. The contents should be unchanged
		// for the life of the file handle.
		_, err = file.Seek(0, 0)
		ogletest.AssertEq(nil, err)

		buffer.Reset()
		bytesRead, err = buffer.ReadFrom(file)
		ogletest.AssertEq(nil, err)
		ogletest.ExpectEq(len(expectedContents), bytesRead)
		ogletest.ExpectEq(expectedContents, buffer.String())
	}(file)

	// The clock was advanced while the old handle was open. The content change
	// should be reflected by the new handle.
	file, err = os.Open(path.Join(t.Dir, "age"))
	ogletest.AssertEq(nil, err)
	func(file *os.File) {
		defer file.Close()

		expectedContents := "This filesystem is 1000 seconds old."
		buffer := bytes.Buffer{}
		bytesRead, err := buffer.ReadFrom(file)
		ogletest.AssertEq(nil, err)
		ogletest.ExpectEq(len(expectedContents), bytesRead)
		ogletest.ExpectEq(expectedContents, buffer.String())
	}(file)
}
