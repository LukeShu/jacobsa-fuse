// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hellofs_test

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/fuse/samples"
	"github.com/jacobsa/fuse/samples/hellofs"
	"github.com/jacobsa/oglematchers"
	"github.com/jacobsa/ogletest"
)

func TestHelloFS(t *testing.T) { ogletest.RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type HelloFSTest struct {
	samples.SampleTest
}

func init() { ogletest.RegisterTestSuite(&HelloFSTest{}) }

func (t *HelloFSTest) SetUp(ti *ogletest.TestInfo) {
	var err error

	t.Server, err = hellofs.NewHelloFS(&t.Clock)
	ogletest.AssertEq(nil, err)

	t.SampleTest.SetUp(ti)
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *HelloFSTest) ReadDir_Root() {
	entries, err := fusetesting.ReadDirPicky(t.Dir)

	ogletest.AssertEq(nil, err)
	ogletest.AssertEq(2, len(entries))
	var fi os.FileInfo

	// dir
	fi = entries[0]
	ogletest.ExpectEq("dir", fi.Name())
	ogletest.ExpectEq(0, fi.Size())
	ogletest.ExpectEq(os.ModeDir|0555, fi.Mode())
	ogletest.ExpectEq(0, t.Clock.Now().Sub(fi.ModTime()), "ModTime: %v", fi.ModTime())
	ogletest.ExpectTrue(fi.IsDir())

	// hello
	fi = entries[1]
	ogletest.ExpectEq("hello", fi.Name())
	ogletest.ExpectEq(len("Hello, world!"), fi.Size())
	ogletest.ExpectEq(0444, fi.Mode())
	ogletest.ExpectEq(0, t.Clock.Now().Sub(fi.ModTime()), "ModTime: %v", fi.ModTime())
	ogletest.ExpectFalse(fi.IsDir())
}

func (t *HelloFSTest) ReadDir_Dir() {
	entries, err := fusetesting.ReadDirPicky(path.Join(t.Dir, "dir"))

	ogletest.AssertEq(nil, err)
	ogletest.AssertEq(1, len(entries))
	var fi os.FileInfo

	// world
	fi = entries[0]
	ogletest.ExpectEq("world", fi.Name())
	ogletest.ExpectEq(len("Hello, world!"), fi.Size())
	ogletest.ExpectEq(0444, fi.Mode())
	ogletest.ExpectEq(0, t.Clock.Now().Sub(fi.ModTime()), "ModTime: %v", fi.ModTime())
	ogletest.ExpectFalse(fi.IsDir())
}

func (t *HelloFSTest) ReadDir_NonExistent() {
	_, err := fusetesting.ReadDirPicky(path.Join(t.Dir, "foobar"))

	ogletest.AssertNe(nil, err)
	ogletest.ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file")))
}

func (t *HelloFSTest) Stat_Hello() {
	fi, err := os.Stat(path.Join(t.Dir, "hello"))
	ogletest.AssertEq(nil, err)

	ogletest.ExpectEq("hello", fi.Name())
	ogletest.ExpectEq(len("Hello, world!"), fi.Size())
	ogletest.ExpectEq(0444, fi.Mode())
	ogletest.ExpectEq(0, t.Clock.Now().Sub(fi.ModTime()), "ModTime: %v", fi.ModTime())
	ogletest.ExpectFalse(fi.IsDir())
	ogletest.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *HelloFSTest) Stat_Dir() {
	fi, err := os.Stat(path.Join(t.Dir, "dir"))
	ogletest.AssertEq(nil, err)

	ogletest.ExpectEq("dir", fi.Name())
	ogletest.ExpectEq(0, fi.Size())
	ogletest.ExpectEq(0555|os.ModeDir, fi.Mode())
	ogletest.ExpectEq(0, t.Clock.Now().Sub(fi.ModTime()), "ModTime: %v", fi.ModTime())
	ogletest.ExpectTrue(fi.IsDir())
	ogletest.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *HelloFSTest) Stat_World() {
	fi, err := os.Stat(path.Join(t.Dir, "dir/world"))
	ogletest.AssertEq(nil, err)

	ogletest.ExpectEq("world", fi.Name())
	ogletest.ExpectEq(len("Hello, world!"), fi.Size())
	ogletest.ExpectEq(0444, fi.Mode())
	ogletest.ExpectEq(0, t.Clock.Now().Sub(fi.ModTime()), "ModTime: %v", fi.ModTime())
	ogletest.ExpectFalse(fi.IsDir())
	ogletest.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *HelloFSTest) Stat_NonExistent() {
	_, err := os.Stat(path.Join(t.Dir, "foobar"))

	ogletest.AssertNe(nil, err)
	ogletest.ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file")))
}

func (t *HelloFSTest) ReadFile_Hello() {
	slice, err := ioutil.ReadFile(path.Join(t.Dir, "hello"))

	ogletest.AssertEq(nil, err)
	ogletest.ExpectEq("Hello, world!", string(slice))
}

func (t *HelloFSTest) ReadFile_Dir() {
	_, err := ioutil.ReadFile(path.Join(t.Dir, "dir"))

	ogletest.AssertNe(nil, err)
	ogletest.ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("is a directory")))
}

func (t *HelloFSTest) ReadFile_World() {
	slice, err := ioutil.ReadFile(path.Join(t.Dir, "dir/world"))

	ogletest.AssertEq(nil, err)
	ogletest.ExpectEq("Hello, world!", string(slice))
}

func (t *HelloFSTest) OpenAndRead() {
	var buf []byte = make([]byte, 1024)
	var n int
	var off int64
	var err error

	// Open the file.
	f, err := os.Open(path.Join(t.Dir, "hello"))
	defer func() {
		if f != nil {
			ogletest.ExpectEq(nil, f.Close())
		}
	}()

	ogletest.AssertEq(nil, err)

	// Seeking shouldn't affect the random access reads below.
	_, err = f.Seek(7, 0)
	ogletest.AssertEq(nil, err)

	// Random access reads
	n, err = f.ReadAt(buf[:2], 0)
	ogletest.AssertEq(nil, err)
	ogletest.ExpectEq(2, n)
	ogletest.ExpectEq("He", string(buf[:n]))

	n, err = f.ReadAt(buf[:2], int64(len("Hel")))
	ogletest.AssertEq(nil, err)
	ogletest.ExpectEq(2, n)
	ogletest.ExpectEq("lo", string(buf[:n]))

	n, err = f.ReadAt(buf[:3], int64(len("Hello, wo")))
	ogletest.AssertEq(nil, err)
	ogletest.ExpectEq(3, n)
	ogletest.ExpectEq("rld", string(buf[:n]))

	// Read beyond end.
	n, err = f.ReadAt(buf[:3], int64(len("Hello, world")))
	ogletest.AssertEq(io.EOF, err)
	ogletest.ExpectEq(1, n)
	ogletest.ExpectEq("!", string(buf[:n]))

	// Seek then read the rest.
	off, err = f.Seek(int64(len("Hel")), 0)
	ogletest.AssertEq(nil, err)
	ogletest.AssertEq(len("Hel"), off)

	n, err = io.ReadFull(f, buf[:len("lo, world!")])
	ogletest.AssertEq(nil, err)
	ogletest.ExpectEq(len("lo, world!"), n)
	ogletest.ExpectEq("lo, world!", string(buf[:n]))
}

func (t *HelloFSTest) Open_NonExistent() {
	_, err := os.Open(path.Join(t.Dir, "foobar"))

	ogletest.AssertNe(nil, err)
	ogletest.ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file")))
}
