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

package fusetesting

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/jacobsa/ogletest"
	"github.com/jacobsa/syncutil"
)

// Run an ogletest test that checks expectations for parallel calls to open(2)
// with O_CREAT.
func RunCreateInParallelTest_NoTruncate(
	ctx context.Context,
	dir string) {
	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Try for awhile to see if anything breaks.
	const duration = 500 * time.Millisecond
	startTime := time.Now()
	for time.Since(startTime) < duration {
		filename := path.Join(dir, "foo")

		// Set up a function that opens the file with O_CREATE and then appends a
		// byte to it.
		worker := func(id byte) error {
			f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				return fmt.Errorf("Worker %d: Open: %v", id, err)
			}
			defer f.Close()

			if _, err := f.Write([]byte{id}); err != nil {
				return fmt.Errorf("Worker %d: Write: %v", id, err)
			}

			return nil
		}

		// Run several workers in parallel.
		const numWorkers = 16
		b := syncutil.NewBundle(ctx)
		for i := 0; i < numWorkers; i++ {
			id := byte(i)
			b.Add(func(ctx context.Context) error {
				return worker(id)
			})
		}

		err := b.Join()
		ogletest.AssertEq(nil, err)

		// Read the contents of the file. We should see each worker's ID once.
		contents, err := ioutil.ReadFile(filename)
		ogletest.AssertEq(nil, err)

		idsSeen := make(map[byte]struct{})
		for i := range contents {
			id := contents[i]
			ogletest.AssertLt(id, numWorkers)

			if _, ok := idsSeen[id]; ok {
				ogletest.AddFailure("Duplicate ID: %d", id)
			}

			idsSeen[id] = struct{}{}
		}

		ogletest.AssertEq(numWorkers, len(idsSeen))

		// Delete the file.
		err = os.Remove(filename)
		ogletest.AssertEq(nil, err)
	}
}

// Run an ogletest test that checks expectations for parallel calls to open(2)
// with O_CREAT|O_TRUNC.
func RunCreateInParallelTest_Truncate(
	ctx context.Context,
	dir string) {
	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Try for awhile to see if anything breaks.
	const duration = 500 * time.Millisecond
	startTime := time.Now()
	for time.Since(startTime) < duration {
		filename := path.Join(dir, "foo")

		// Set up a function that opens the file with O_CREATE and O_TRUNC and then
		// appends a byte to it.
		worker := func(id byte) (err error) {
			f, err := os.OpenFile(
				filename,
				os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC,
				0600)
			if err != nil {
				return fmt.Errorf("Worker %d: Open: %v", id, err)
			}
			defer f.Close()

			if _, err := f.Write([]byte{id}); err != nil {
				return fmt.Errorf("Worker %d: Write: %v", id, err)
			}

			return nil
		}

		// Run several workers in parallel.
		const numWorkers = 16
		b := syncutil.NewBundle(ctx)
		for i := 0; i < numWorkers; i++ {
			id := byte(i)
			b.Add(func(ctx context.Context) error {
				return worker(id)
			})
		}

		err := b.Join()
		ogletest.AssertEq(nil, err)

		// Read the contents of the file. We should see at least one ID (the last
		// one that truncated), and at most all of them.
		contents, err := ioutil.ReadFile(filename)
		ogletest.AssertEq(nil, err)

		idsSeen := make(map[byte]struct{})
		for i := range contents {
			id := contents[i]
			ogletest.AssertLt(id, numWorkers)

			if _, ok := idsSeen[id]; ok {
				ogletest.AddFailure("Duplicate ID: %d", id)
			}

			idsSeen[id] = struct{}{}
		}

		ogletest.AssertGe(len(idsSeen), 1)
		ogletest.AssertLe(len(idsSeen), numWorkers)

		// Delete the file.
		err = os.Remove(filename)
		ogletest.AssertEq(nil, err)
	}
}

// Run an ogletest test that checks expectations for parallel calls to open(2)
// with O_CREAT|O_EXCL.
func RunCreateInParallelTest_Exclusive(
	ctx context.Context,
	dir string) {
	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Try for awhile to see if anything breaks.
	const duration = 500 * time.Millisecond
	startTime := time.Now()
	for time.Since(startTime) < duration {
		filename := path.Join(dir, "foo")

		// Set up a function that opens the file with O_CREATE and O_EXCL, and then
		// appends a byte to it if it was successfully opened.
		var openCount uint64
		worker := func(id byte) (err error) {
			f, err := os.OpenFile(
				filename,
				os.O_CREATE|os.O_EXCL|os.O_WRONLY|os.O_APPEND,
				0600)

			// If we failed to open due to the file already existing, just leave.
			if os.IsExist(err) {
				return nil
			}

			// Propgate other errors.
			if err != nil {
				return fmt.Errorf("Worker %d: Open: %v", id, err)
			}

			atomic.AddUint64(&openCount, 1)
			defer f.Close()

			if _, err := f.Write([]byte{id}); err != nil {
				return fmt.Errorf("Worker %d: Write: %v", id, err)
			}

			return nil
		}

		// Run several workers in parallel.
		const numWorkers = 16
		b := syncutil.NewBundle(ctx)
		for i := 0; i < numWorkers; i++ {
			id := byte(i)
			b.Add(func(ctx context.Context) error {
				return worker(id)
			})
		}

		err := b.Join()
		ogletest.AssertEq(nil, err)

		// Exactly one worker should have opened successfully.
		ogletest.AssertEq(1, openCount)

		// Read the contents of the file. It should contain that one worker's ID.
		contents, err := ioutil.ReadFile(filename)
		ogletest.AssertEq(nil, err)

		ogletest.AssertEq(1, len(contents))
		ogletest.AssertLt(contents[0], numWorkers)

		// Delete the file.
		err = os.Remove(filename)
		ogletest.AssertEq(nil, err)
	}
}

// Run an ogletest test that checks expectations for parallel calls to mkdir(2).
func RunMkdirInParallelTest(
	ctx context.Context,
	dir string) {
	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Try for awhile to see if anything breaks.
	const duration = 500 * time.Millisecond
	startTime := time.Now()
	for time.Since(startTime) < duration {
		filename := path.Join(dir, "foo")

		// Set up a function that creates the directory, ignoring EEXIST errors.
		worker := func(id byte) error {
			err := os.Mkdir(filename, 0700)
			if os.IsExist(err) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("Worker %d: Mkdir: %v", id, err)
			}

			return nil
		}

		// Run several workers in parallel.
		const numWorkers = 16
		b := syncutil.NewBundle(ctx)
		for i := 0; i < numWorkers; i++ {
			id := byte(i)
			b.Add(func(ctx context.Context) error {
				return worker(id)
			})
		}

		err := b.Join()
		ogletest.AssertEq(nil, err)

		// The directory should have been created, once.
		entries, err := ReadDirPicky(dir)
		ogletest.AssertEq(nil, err)
		ogletest.AssertEq(1, len(entries))
		ogletest.AssertEq("foo", entries[0].Name())

		// Delete the directory.
		err = os.Remove(filename)
		ogletest.AssertEq(nil, err)
	}
}

// Run an ogletest test that checks expectations for parallel calls to
// symlink(2).
func RunSymlinkInParallelTest(
	ctx context.Context,
	dir string) {
	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Try for awhile to see if anything breaks.
	const duration = 500 * time.Millisecond
	startTime := time.Now()
	for time.Since(startTime) < duration {
		filename := path.Join(dir, "foo")

		// Set up a function that creates the symlink, ignoring EEXIST errors.
		worker := func(id byte) error {
			err := os.Symlink("blah", filename)
			if os.IsExist(err) {
				return nil
			}

			if err != nil {
				return fmt.Errorf("Worker %d: Symlink: %v", id, err)
			}

			return nil
		}

		// Run several workers in parallel.
		const numWorkers = 16
		b := syncutil.NewBundle(ctx)
		for i := 0; i < numWorkers; i++ {
			id := byte(i)
			b.Add(func(ctx context.Context) error {
				return worker(id)
			})
		}

		err := b.Join()
		ogletest.AssertEq(nil, err)

		// The symlink should have been created, once.
		entries, err := ReadDirPicky(dir)
		ogletest.AssertEq(nil, err)
		ogletest.AssertEq(1, len(entries))
		ogletest.AssertEq("foo", entries[0].Name())

		// Delete the directory.
		err = os.Remove(filename)
		ogletest.AssertEq(nil, err)
	}
}

// Run an ogletest test that checks expectations for parallel calls to
// link(2).
func RunHardlinkInParallelTest(
	ctx context.Context,
	dir string) {
	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Create a file.
	originalFile := path.Join(dir, "original_file")
	const contents = "Hello\x00world"

	err := ioutil.WriteFile(originalFile, []byte(contents), 0444)
	ogletest.AssertEq(nil, err)

	// Try for awhile to see if anything breaks.
	const duration = 500 * time.Millisecond
	startTime := time.Now()
	for time.Since(startTime) < duration {
		filename := path.Join(dir, "foo")

		// Set up a function that creates the symlink, ignoring EEXIST errors.
		worker := func(id byte) error {
			err := os.Link(originalFile, filename)
			if os.IsExist(err) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("Worker %d: Link: %v", id, err)
			}

			return nil
		}

		// Run several workers in parallel.
		const numWorkers = 16
		b := syncutil.NewBundle(ctx)
		for i := 0; i < numWorkers; i++ {
			id := byte(i)
			b.Add(func(ctx context.Context) error {
				return worker(id)
			})
		}

		err := b.Join()
		ogletest.AssertEq(nil, err)

		// The symlink should have been created, once.
		entries, err := ReadDirPicky(dir)
		ogletest.AssertEq(nil, err)
		ogletest.AssertEq(2, len(entries))
		ogletest.AssertEq("foo", entries[0].Name())
		ogletest.AssertEq("original_file", entries[1].Name())

		// Remove the link.
		err = os.Remove(filename)
		ogletest.AssertEq(nil, err)
	}

	// Clean up the original file at the end.
	err = os.Remove(originalFile)
	ogletest.AssertEq(nil, err)
}
