package seeding

import (
	"errors"
	"fmt"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"strconv"
)

func (s *Server) processSubmitTx(arTx types.Transaction) error {
	// 1. verify ar tx
	if err := utils.VerifyTransaction(arTx); err != nil {
		log.Error("utils.VerifyTransaction(arTx)", "err", err, "arTx", arTx.ID)
		return err
	}

	// 2. check meta exist
	if s.store.IsExistTxMeta(arTx.ID) {
		return errors.New("arTx meta exist")
	}

	// 3. save tx meta
	if err := s.store.SaveTxMeta(arTx); err != nil {
		log.Error("s.store.SaveTxMeta(arTx)", "err", err, "arTx", arTx.ID)
		return err
	}

	s.submitLocker.Lock()
	defer s.submitLocker.Unlock()

	// 4. check whether update allDataEndOffset
	if s.store.IsExistTxDataEndOffset(arTx.DataRoot, arTx.DataSize) {
		return nil
	}
	// add txDataEndOffset
	if err := addTxDataEndOffset(s.store, arTx.DataRoot, arTx.DataSize); err != nil {
		log.Error("addTxDataEndOffset(s.store,arTx.DataRoot,arTx.DataSize)", "err", err, "arTx", arTx.ID)
		return err
	}

	// 5. save tx data chunk if exist
	txSize, err := strconv.ParseUint(arTx.DataSize, 10, 64)
	if err != nil {
		return err
	}
	if len(arTx.Data) > 0 && txSize <= types.MAX_CHUNK_SIZE {
		// generate chunk
		dataBy, err := utils.Base64Decode(arTx.Data)
		if err != nil {
			log.Error("utils.Base64Decode(arTx.Data)", "err", err, "data", arTx.Data)
			return err
		}
		utils.PrepareChunks(&arTx, dataBy)
		// only one chunk
		if len(arTx.Chunks.Chunks) != 1 {
			return fmt.Errorf("arTx must only one chunk, arTx:%s", arTx.ID)
		}
		chunk, err := utils.GetChunk(arTx, 0, dataBy)
		if err != nil {
			log.Error("utils.GetChunk(arTx,0,dataBy)", "err", err, "arTx", arTx.ID)
			return err
		}
		// store chunk
		if err := storeChunk(*chunk, s.store); err != nil {
			log.Error("storeChunk(*chunk,s.store)", "err", err, "chunk", *chunk)
			return err
		}
	}
	log.Debug("success process a new arTx", "arTx", arTx.ID)
	return nil
}

func (s *Server) processSubmitChunk(chunk types.GetChunk) error {
	// 1. verify chunk
	err, ok := verifyChunk(chunk)
	if err != nil || !ok {
		log.Error("verifyChunk(chunk) failed", "err", err, "chunk", chunk)
		return fmt.Errorf("verifyChunk error:%v", err)
	}

	s.submitLocker.Lock()
	defer s.submitLocker.Unlock()

	// 2. check TxDataEndOffset exist
	if !s.store.IsExistTxDataEndOffset(chunk.DataRoot, chunk.DataSize) {
		// add TxDataEndOffset
		if err := addTxDataEndOffset(s.store, chunk.DataRoot, chunk.DataSize); err != nil {
			log.Error("addTxDataEndOffset(s.store,chunk.DataRoot,chunk.DataSize)", "err", err, "chunk", chunk)
			return err
		}
	}

	// 3. store chunk
	if err := storeChunk(chunk, s.store); err != nil {
		log.Error("storeChunk(chunk,s.store)", "err", err, "chunk", chunk)
		return err
	}

	return nil
}

func storeChunk(chunk types.GetChunk, db *Store) error {
	// generate chunkStartOffset
	txDataEndOffset, err := db.LoadTxDataEndOffSet(chunk.DataRoot, chunk.DataSize)
	if err != nil {
		log.Error("db.LoadTxDataEndOffSet(chunk.DataRoot,chunk.DataSize)", "err", err, "root", chunk.DataRoot, "size", chunk.DataSize)
		return err
	}
	txSize, err := strconv.ParseUint(chunk.DataSize, 10, 64)
	if err != nil {
		return err
	}
	txDataStartOffset := txDataEndOffset - txSize + 1

	offset, err := strconv.ParseUint(chunk.Offset, 10, 64)
	if err != nil {
		return err
	}
	chunkEndOffset := txDataStartOffset + offset
	chunkDataBy, err := utils.Base64Decode(chunk.Chunk)
	if err != nil {
		return err
	}
	chunkStartOffset := chunkEndOffset - uint64(len(chunkDataBy))
	// save
	if err := db.SaveChunk(chunkStartOffset, chunk); err != nil {
		log.Error("s.store.SaveChunk(chunkStartOffset, *chunk)", "err", err)
		return err
	}
	return nil
}

func addTxDataEndOffset(db *Store, dataRoot, dataSize string) error {
	// update allDataEndOffset
	txSize, err := strconv.ParseUint(dataSize, 10, 64)
	if err != nil {
		log.Error("strconv.ParseUint(arTx.DataSize,10,64)", "err", err)
		return err
	}
	curEndOffset := db.LoadAllDataEndOffset()
	newEndOffset := curEndOffset + txSize

	// must use tx db
	boltTx, err := db.BoltDb.Begin(true)
	if err != nil {
		log.Error("s.store.BoltDb.Begin(true)", "err", err)
		return err
	}
	if err := db.SaveAllDataEndOffset(newEndOffset, boltTx); err != nil {
		boltTx.Rollback()
		log.Error("s.store.SaveAllDataEndOffset(newEndOffset)", "err", err)
		return err
	}
	// SaveTxDataEndOffSet
	if err := db.SaveTxDataEndOffSet(dataRoot, dataSize, newEndOffset, boltTx); err != nil {
		boltTx.Rollback()
		return err
	}
	// commit
	if err := boltTx.Commit(); err != nil {
		boltTx.Rollback()
		return err
	}
	return nil
}

func verifyChunk(chunk types.GetChunk) (err error, ok bool) {
	dataRoot, err := utils.Base64Decode(chunk.DataRoot)
	if err != nil {
		return
	}
	offset, err := strconv.Atoi(chunk.Offset)
	if err != nil {
		return
	}
	dataSize, err := strconv.Atoi(chunk.DataSize)
	if err != nil {
		return
	}
	path, err := utils.Base64Decode(chunk.DataPath)
	if err != nil {
		return
	}
	_, ok = utils.ValidatePath(dataRoot, offset, 0, dataSize, path)
	return
}
