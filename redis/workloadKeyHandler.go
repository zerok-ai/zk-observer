package redis

import (
	"fmt"
	"github.com/google/uuid"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"log"
	"time"
)

const (
	TickerInterval = 1 * time.Minute
)

// WorkloadKeyHandler handles periodic tasks to manage workload keys in Redis.
type WorkloadKeyHandler struct {
	RedisHandler *RedisHandler
	UUID         string
	logTag       string
}

// NewWorkloadKeyHandler initializes a new WorkloadKeyHandler with a given RedisHandler.
func NewWorkloadKeyHandler(redisHandler *RedisHandler) *WorkloadKeyHandler {
	// Generate a new UUID for the instance
	uuid := uuid.New().String()

	handler := &WorkloadKeyHandler{
		RedisHandler: redisHandler,
		UUID:         uuid,
		logTag:       "WorkloadKeyHandler",
	}

	// Start the ticker
	go handler.startTicker()

	return handler
}

// startTicker starts a ticker that periodically invokes manageWorkloadKeys for each scenario.
func (wh *WorkloadKeyHandler) startTicker() {
	ticker := time.NewTicker(TickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wh.manageWorkloadKeys()
		}
	}
}

// manageWorkloadKeys retrieves scenarios from a store and calls ManageWorkloadKey for each one.
func (wh *WorkloadKeyHandler) manageWorkloadKeys() {
	// TODO: Retrieve scenarios from store and iterate over them
	// For now, we'll use a dummy slice of workload IDs for the example
	workloadIDs := []string{"workload1", "workload2", "workload3"}

	for _, workloadID := range workloadIDs {
		// Call the ManageWorkloadKey method for each workloadID
		err := wh.ManageWorkloadKey(wh.RedisHandler, workloadID, wh.UUID)
		if err != nil {
			log.Printf("[%s] Error managing workload key for '%s': %v", wh.logTag, workloadID, err)
		}
	}
}

func (wh *WorkloadKeyHandler) ManageWorkloadKey(h *RedisHandler, workloadID string, podUUID string) error {
	// 1. Create a key with value(rename_worker_<workload_id>) as the UUID of the pod with ttl as 1min
	keyName := fmt.Sprintf("rename_worker_%s", workloadID)
	if err := h.SetWithTTL(keyName, podUUID, time.Minute); err != nil {
		return fmt.Errorf("error setting key with TTL: %v", err)
	}

	// 2. Use the utility method to get keys by pattern
	pattern := fmt.Sprintf("%s_*", workloadID)
	keys, err := wh.RedisHandler.GetKeysByPattern(pattern)
	if err != nil {
		return fmt.Errorf("error retrieving keys with pattern %s: %v", pattern, err)
	}

	// Find the highest suffix.
	highestSuffix := -1
	for _, key := range keys {
		var suffix int
		_, err := fmt.Sscanf(key, workloadID+"_%d", &suffix)
		if err == nil && suffix > highestSuffix {
			highestSuffix = suffix
		}
	}

	// 3. Go back and check if the value of the key created in step 1 is still its own UUID.
	currentValue, err := wh.RedisHandler.Get(keyName)
	if err != nil {
		return fmt.Errorf("error getting value for key %s: %v", keyName, err)
	}

	if currentValue == podUUID {
		// 3.1 If it’s the same, then rename the key workload_latest to the new key calculated in step 2 and set the ttl as 15mins.
		newKeyName := fmt.Sprintf("%s_%02d", workloadID, (highestSuffix+1)%60)
		if err := h.Rename("workload_latest", newKeyName); err != nil {
			return fmt.Errorf("error renaming key: %v", err)
		}
		if err := h.SetWithTTL(newKeyName, podUUID, 15*time.Minute); err != nil {
			return fmt.Errorf("error setting TTL for new key: %v", err)
		}
	} else {
		// 3.2 If it’s not the same, then ignore.
		logger.Info(redisHandlerLogTag, "UUID mismatch, ignoring rename operation")
	}

	return nil
}
