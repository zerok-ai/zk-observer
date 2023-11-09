package redis

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/zerok-ai/zk-otlp-receiver/config"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	zkmodel "github.com/zerok-ai/zk-utils-go/scenario/model"
	zkredis "github.com/zerok-ai/zk-utils-go/storage/redis"
	"github.com/zerok-ai/zk-utils-go/storage/redis/clientDBNames"
	zktick "github.com/zerok-ai/zk-utils-go/ticker"
	"time"
)

const (
	TickerInterval = 1 * time.Minute
)

var workloadLogTag = "WorkloadKeyHandler"

// WorkloadKeyHandler handles periodic tasks to manage workload keys in Redis.
type WorkloadKeyHandler struct {
	RedisHandler  *RedisHandler
	UUID          string
	logTag        string
	scenarioStore *zkredis.VersionedStore[zkmodel.Scenario]
	ticker        *zktick.TickerTask
}

func NewWorkloadKeyHandler(cfg *config.OtlpConfig, store *zkredis.VersionedStore[zkmodel.Scenario]) (*WorkloadKeyHandler, error) {
	// Generate a new ID for the instance
	uniqueId := uuid.New().String()

	redisHandler, err := NewRedisHandlerWithoutTicker(&cfg.Redis, clientDBNames.FilteredTracesDBName, workloadLogTag)
	if err != nil {
		logger.Error(resourceLogTag, "Error while creating resource redis handler:", err)
		return nil, err
	}

	handler := &WorkloadKeyHandler{
		RedisHandler:  redisHandler,
		UUID:          uniqueId,
		logTag:        "WorkloadKeyHandler",
		scenarioStore: store,
	}

	handler.ticker = zktick.GetNewTickerTask("workload_rename", TickerInterval, handler.manageWorkloadKeys)
	handler.ticker.Start()

	return handler, nil
}

// manageWorkloadKeys retrieves scenarios from a store and calls ManageWorkloadKey for each one.
func (wh *WorkloadKeyHandler) manageWorkloadKeys() {
	scenarios := wh.scenarioStore.GetAllValues()
	for _, scenario := range scenarios {
		if scenario == nil || scenario.Workloads == nil {
			logger.Error(workloadLogTag, "Scenario or workloads in nil.")
			continue
		}
		for workloadID, _ := range *scenario.Workloads {
			// Call the ManageWorkloadKey method for each workloadID
			err := wh.ManageWorkloadKey(workloadID)
			if err != nil {
				logger.Error(workloadLogTag, "Error managing workload key for ", workloadID, " err: ", err)
			}
		}
	}
}

func (wh *WorkloadKeyHandler) ManageWorkloadKey(workloadID string) error {
	// 1. Create a key with value(rename_worker_<workload_id>) as the UUID of the pod with ttl as 1min
	keyName := fmt.Sprintf("rename_worker_%s", workloadID)
	if err := wh.RedisHandler.SetWithTTL(keyName, wh.UUID, time.Minute); err != nil {
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

	if currentValue == wh.UUID {
		// If it’s the same, then rename the key workload_latest to the new key calculated in step 2 and set the ttl as 15mins.
		newKeyName := fmt.Sprintf("%s_%02d", workloadID, (highestSuffix+1)%60)
		if err := wh.RedisHandler.Rename("workload_latest", newKeyName); err != nil {
			return fmt.Errorf("error renaming key: %v", err)
		}
		if err := wh.RedisHandler.SetWithTTL(newKeyName, wh.UUID, 15*time.Minute); err != nil {
			return fmt.Errorf("error setting TTL for new key: %v", err)
		}
	} else {
		// If it’s not the same, then ignore.
		logger.Info(workloadLogTag, "UUID mismatch, ignoring rename operation")
	}

	return nil
}
