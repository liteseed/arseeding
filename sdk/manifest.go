package sdk

import (
	"encoding/json"
	seedSchema "github.com/everFinance/arseeding/schema"
	"github.com/everFinance/arseeding/sdk/schema"
	"github.com/everFinance/goar/types"
	"github.com/panjf2000/ants/v2"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

func (s *SDK) UploadFolder(rootPath string, batchSize int, indexFile string, currency string) ([]*seedSchema.RespOrder, error) {
	if indexFile == "" {
		indexFile = "index.html"
	}

	// create manifest data
	manifestFile := &seedSchema.ManifestData{
		Manifest: "arweave/paths",
		Version:  "0.1.0",
		Index: seedSchema.IndexPath{
			Path: indexFile,
		},
		Paths: make(map[string]seedSchema.Resource),
	}

	pathFiles, err := getPathFiles(rootPath)
	if err != nil {
		return nil, err
	}

	orders := make([]*seedSchema.RespOrder, 0, len(pathFiles))

	var (
		lock sync.Mutex
		wg   sync.WaitGroup
	)

	if batchSize == 0 {
		batchSize = 10
	}
	p, _ := ants.NewPoolWithFunc(batchSize, func(i interface{}) {
		fp := i.(string)
		data, err := readFileData(rootPath, fp)
		if err != nil {
			panic(err)
		}
		// bundle item and send to arseeding
		order, err := s.SendData(data, currency, nil)
		if err != nil {
			panic(err)
		}
		lock.Lock()
		orders = append(orders, order)
		// add manifest file
		manifestFile.Paths[fp] = seedSchema.Resource{
			TxId: order.ItemId,
		}
		lock.Unlock()
		wg.Done()
	}, ants.WithPanicHandler(func(err interface{}) {
		panic(err)
	}))

	defer p.Release()

	for _, fp := range pathFiles {
		wg.Add(1)
		_ = p.Invoke(fp)
	}
	wg.Wait()

	// submit manifest file
	manifestFileBy, err := json.Marshal(manifestFile)
	if err != nil {
		return nil, err
	}
	order, err := s.SendData(manifestFileBy, currency, &schema.OptionItem{
		Tags: []types.Tag{{Name: "Type", Value: "manifest"}, {Name: "Content-Type", Value: "application/x.arweave-manifest+json"}},
	})
	if err != nil {
		return nil, err
	}
	orders = append(orders, order)
	return orders, nil
}

func readFileData(rootPath, filePath string) ([]byte, error) {
	allPath := path.Join(rootPath, filePath)
	data, err := ioutil.ReadFile(allPath)
	return data, err
}

func getPathFiles(rootPath string) ([]string, error) {
	var files []string
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rootAbs, err := filepath.Abs(rootPath)
		if err != nil {
			return err
		}

		fileAbs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		pp := strings.TrimPrefix(fileAbs, rootAbs+"/")
		files = append(files, pp)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
