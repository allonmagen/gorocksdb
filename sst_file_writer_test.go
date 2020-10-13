package gorocksdb

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
)

func TestCreateSstFile(t *testing.T) {
	pCpu := fmt.Sprintf("/Users/amagen/pprof/cpuprofile_%v.pprof", time.Now().UnixNano())
	pMem := fmt.Sprintf("/Users/amagen/pprof/memprofile_%v.pprof", time.Now().UnixNano())
	_ = os.MkdirAll("/tmp/profiles", os.ModePerm)

	cpuProfile, _ := os.Create(filepath.Join(pCpu))
	memProfile, _ := os.Create(filepath.Join(pMem))

	pprof.StartCPUProfile(cpuProfile)

	root := getTmpPath()
	_ = os.MkdirAll(root, os.ModePerm)
	paths := make([]string, 0, 10)
	dbPath := filepath.Join(root, "test.rocksdb")
	options := NewDefaultOptions()
	options.SetCreateIfMissing(true)
	options.PrepareForBulkLoad()
	wg := sync.WaitGroup{}
	for j := 0; j < 10; j++ {
		wg.Add(1)
		h := j
		path := filepath.Join(root, fmt.Sprintf("%d.sst", j))
		paths = append(paths, path)
		sstWriter := NewSSTFileWriter(NewDefaultEnvOptions(), options)
		sstWriter.Open(path)
		go func() {
			defer wg.Done()
			bufK := make([]byte, 4)
			bufV := make([]byte, 20)

			for i := 0; i < 10000000; i++ {
				binary.BigEndian.PutUint32(bufK, uint32(i + 10000000*h))
				binary.BigEndian.PutUint32(bufV, uint32(i))
				err := sstWriter.Add(bufK, bufV)
				if err != nil {
					t.Fatal(err)
				}
			}

			err := sstWriter.Finish()
			sstWriter.Destroy()
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println(time.Now(), path)
		}()
	}
	wg.Wait()
	db, err := OpenDb(options, dbPath)
	if err != nil {
		t.Fatal(err)
	}

	err = db.IngestExternalFile(paths, NewDefaultIngestExternalFileOptions())
	if err != nil {
		t.Fatal(err)
	}

	bufK := make([]byte, 4)
	binary.BigEndian.PutUint32(bufK, 1)

	v, err := db.Get(NewDefaultReadOptions(), bufK)
	fmt.Println(bufK, v.Data(), err)

	st := time.Now()
	runtime.GC()
	fmt.Println(time.Now().Sub(st))

	pprof.StopCPUProfile()
	pprof.WriteHeapProfile(memProfile)
	fmt.Println(pCpu)
	fmt.Println(pMem)
	fmt.Println(root)



	_ = os.RemoveAll(root)
}

func getTmpPath() string {
	p := fmt.Sprintf("/tmp/%v", time.Now().UnixNano())
	_ = os.MkdirAll(p, os.ModePerm)
	return p
}
