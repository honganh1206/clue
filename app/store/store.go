package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

type ModelConfig struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	MaxTokens int64  `json:"max_tokens"`
}

type Store struct {
	ID           string `json:"id"`
	FirstTimeRun bool   `json:"first_time_run"`
}

var (
	lock  sync.Mutex
	store Store
)

func GetID() string {
	lock.Lock()
	defer lock.Unlock()
	if store.ID == "" {
		initStore()
	}
	return store.ID
}

func GetFirstTimeRun() bool {
	lock.Lock()
	defer lock.Unlock()
	if store.ID == "" {
		initStore()
	}
	return store.FirstTimeRun
}

func SetFirstTimeRun(val bool) {
	lock.Lock()
	defer lock.Unlock()
	if store.FirstTimeRun == val {
		return
	}
	store.FirstTimeRun = val
	writeStore(getStorePath())
}

func initStore() {
	storeFile, err := os.Open(getStorePath())
	if err == nil {
		defer storeFile.Close()
		// Structure of this store? only ID and a boolean?
		err = json.NewDecoder(storeFile).Decode(&store)
		if err == nil {
			slog.Debug(fmt.Sprintf("loaded existing store %s - ID: %s", getStorePath(), store.ID))
			return
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		slog.Debug(fmt.Sprintf("unexpected error searching for store: %s", err))
	}

	slog.Debug("initializing new store")
	store.ID = uuid.NewString()
	writeStore(getStorePath())
}

func writeStore(storeFileName string) {
	storeDir := filepath.Dir(storeFileName)
	_, err := os.Stat(storeDir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(storeDir, 0o755); err != nil {
			slog.Error(fmt.Sprintf("create store dir for clue %s: %v", storeDir, err))
			return
		}
	}
	payload, err := json.Marshal(store)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to marshal store: %s", err))
		return
	}

	// Write only, create, and empty the file before writing
	fp, err := os.OpenFile(storeFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		slog.Error(fmt.Sprintf("error when writing store payload %s: %v", storeFileName, err))
		return
	}

	defer fp.Close()
	if n, err := fp.Write(payload); err != nil || n != len(payload) {
		slog.Error(fmt.Sprintf("write store payload %s: %d vs %d -- %v", storeFileName, n, len(payload), err))
		return
	}

	slog.Debug("Store contents: " + string(payload))
	slog.Info(fmt.Sprintf("wrote store: %s", storeFileName))
}
