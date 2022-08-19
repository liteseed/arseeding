package arseeding

import (
	"encoding/json"
	"strings"

	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
)

func handleManifest(maniData []byte, path string, db *Store) ([]types.Tag, []byte, error) {
	mani := schema.ManifestData{}
	if err := json.Unmarshal(maniData, &mani); err != nil {
		return nil, nil, err
	}

	originalPath := path

	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		if mani.Index.Path == "" {
			return nil, nil, schema.ErrPageNotFound
		}
		path = mani.Index.Path
	}
	txId, ok := mani.Paths[path]
	if !ok {
		if strings.HasSuffix(originalPath, "/") {
			txId, ok = mani.Paths[path+"/"+"index.html"]
			if !ok {
				return nil, nil, schema.ErrPageNotFound
			}
		} else {
			return nil, nil, schema.ErrPageNotFound
		}
	}

	tags, data, err := getArTxOrItemData(txId.TxId, db)
	return tags, data, err
}

func getArTxOrItemData(id string, db *Store) (decodeTags []types.Tag, data []byte, err error) {
	// find arId
	txMeta, err := db.LoadTxMeta(id)
	if err == nil { // arTx id
		data, err = txDataByMeta(txMeta, db)
		if err != nil {
			return
		}
		decodeTags, err = utils.TagsDecode(txMeta.Tags)
		return
	}

	// not arId
	// find bundle item
	itemBinary, err := db.LoadItemBinary(id)
	if err == nil {
		var item *types.BundleItem
		item, err = utils.DecodeBundleItem(itemBinary)
		if err != nil {
			return
		}
		decodeTags = item.Tags
		data, err = utils.Base64Decode(item.Data)
		return
	}
	// txId not found in local, need proxy to gateway
	return nil, nil, schema.ErrLocalNotExist
}

func getBundleItemData(id string, db *Store) (decodeTags []types.Tag, data []byte, err error) {
	itemBinary, err := db.LoadItemBinary(id)
	if err == nil {
		var item *types.BundleItem
		item, err = utils.DecodeBundleItem(itemBinary)
		if err != nil {
			return
		}
		decodeTags = item.Tags
		data, err = utils.Base64Decode(item.Data)
		return
	}
	return
}
