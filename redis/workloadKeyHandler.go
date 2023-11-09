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
		scenarioStore: store,
	}

	handler.ticker = zktick.GetNewTickerTask("workload_rename", TickerInterval, handler.manageWorkloadKeys)
	handler.ticker.Start()

	return handler, nil
}

// manageWorkloadKeys retrieves scenarios from scenario store and calls ManageWorkloadKey for all workloads.
func (wh *WorkloadKeyHandler) manageWorkloadKeys() {
	scenarios := wh.scenarioStore.GetAllValues()
	for _, scenario := range scenarios {
		if scenario == nil || scenario.Workloads == nil {
			logger.Error(workloadLogTag, "Scenario or workloads in nil.")
			continue
		}
		for workloadID, _ := range *scenario.Workloads {
			err := wh.ManageWorkloadKey(workloadID)
			if err != nil {
				logger.Error(workloadLogTag, "Error managing workload key for ", workloadID, " err: ", err)
			}
		}
	}
}

func (wh *WorkloadKeyHandler) ManageWorkloadKey(workloadID string) error {

	lockKeyName := fmt.Sprintf("rename_worker_%s", workloadID)
	currentValue, err := wh.RedisHandler.Get(lockKeyName)
	if err != nil {
		return fmt.Errorf("error getting value for lock key %s: %v", lockKeyName, err)
	}

	if currentValue != "" {
		logger.Info(workloadLogTag, "Another UUID already present, ignoring the operation for workload ", workloadID)
		return nil
	}

	// 2. Create a key with value(rename_worker_<workload_id>) as the UUID of the pod with ttl as 1min

	if err := wh.RedisHandler.SetWithTTL(lockKeyName, wh.UUID, time.Minute); err != nil {
		return fmt.Errorf("error setting key with TTL: %v", err)
	}

	// 3. Use the utility method to get keys by pattern
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

	// 4. Go back and check if the value of the key created in step 1 is still its own UUID.
	currentValue, err = wh.RedisHandler.Get(lockKeyName)
	if err != nil {
		return fmt.Errorf("error getting value for key %s: %v", lockKeyName, err)
	}

	if currentValue != wh.UUID {
		logger.Info(workloadLogTag, "UUID mismatch, ignoring rename operation")
		return nil
	}

	// If it’s the same, then rename the key workload_latest to the new key calculated in step 2 and set the ttl as 15mins.
	newKeyName := fmt.Sprintf("%s_%d", workloadID, (highestSuffix+1)%60)
	oldKeyName := fmt.Sprintf("%s_latest", workloadID)
	if err := wh.RedisHandler.RenameKeyWithTTL(oldKeyName, newKeyName, 15*time.Minute); err != nil {
		err := wh.RedisHandler.RemoveKey(lockKeyName)
		if err != nil {
			logger.Error(workloadLogTag, "Error removing the lock key for workloadId ", workloadID)
			return fmt.Errorf("error removing the lock key for workloadId %s", workloadID)
		}
		return fmt.Errorf("error renaming key: %v", err)
	}

	return nil
}
