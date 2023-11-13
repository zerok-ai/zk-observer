package tests

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/zerok-ai/zk-otlp-receiver/redis"
	"github.com/zerok-ai/zk-otlp-receiver/tests/mocks"
	"os"
	"testing"
	"time"
	// other necessary imports
)

func TestMain(m *testing.M) {
	fmt.Println("TestMain")
	// Setup code goes here
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() {
	fmt.Println("setup")
}

func teardown() {
	fmt.Println("teardown")
}

func TestRenameKeys(t *testing.T) {
	redisHandler := mocks.NewMockRedisHandler()
	redisHandler.Set("workload1_1", "data1")
	redisHandler.Set("workload1_2", "data2")
	redisHandler.Set("workload1_latest", "data3")

	workLoadKeyHandler := redis.WorkloadKeyHandler{}
	workLoadKeyHandler.RedisHandler = redisHandler

	fmt.Println(redisHandler.GetAllKeys())

	workLoadKeyHandler.ManageWorkloadKey("workload1")

	expectedKeys := []string{"rename_worker_workload1", "workload1_1", "workload1_2", "workload1_3"}

	actualKeys, _ := redisHandler.GetAllKeys()

	for i, key := range expectedKeys {
		assert.Equal(t, key, actualKeys[i])
	}
}

func TestLock(t *testing.T) {
	redisHandler := mocks.NewMockRedisHandler()
	redisHandler.Set("workload1_1", "data1")
	redisHandler.Set("workload1_2", "data2")
	redisHandler.Set("workload1_latest", "data3")

	testUUID := "TestUUID"

	workLoadKeyHandler := redis.WorkloadKeyHandler{}
	workLoadKeyHandler.RedisHandler = redisHandler
	workLoadKeyHandler.UUID = testUUID

	fmt.Println(redisHandler.GetAllKeys())

	workLoadKeyHandler.ManageWorkloadKey("workload1")

	lockKeyName := "rename_worker_workload1"

	actualValue, _ := redisHandler.Get(lockKeyName)

	fmt.Println(actualValue)

	assert.Equal(t, testUUID, actualValue)
}

func TestLockTtl(t *testing.T) {
	redisHandler := mocks.NewMockRedisHandler()
	redisHandler.Set("workload1_1", "data1")
	redisHandler.Set("workload1_2", "data2")
	redisHandler.Set("workload1_latest", "data3")

	testUUID := "TestUUID"

	workLoadKeyHandler := redis.WorkloadKeyHandler{}
	workLoadKeyHandler.RedisHandler = redisHandler
	workLoadKeyHandler.UUID = testUUID

	fmt.Println(redisHandler.GetAllKeys())

	workLoadKeyHandler.ManageWorkloadKey("workload1")

	lockKeyName := "rename_worker_workload1"

	actualValue, _ := redisHandler.GetTtl(lockKeyName)

	fmt.Println(actualValue)

	assert.Equal(t, time.Minute, actualValue)
}

func TestWorkloadKeyTtl(t *testing.T) {
	redisHandler := mocks.NewMockRedisHandler()
	redisHandler.Set("workload1_1", "data1")
	redisHandler.Set("workload1_2", "data2")
	redisHandler.Set("workload1_latest", "data3")

	testUUID := "TestUUID"

	workLoadKeyHandler := redis.WorkloadKeyHandler{}
	workLoadKeyHandler.RedisHandler = redisHandler
	workLoadKeyHandler.UUID = testUUID

	fmt.Println(redisHandler.GetAllKeys())

	workLoadKeyHandler.ManageWorkloadKey("workload1")

	newKeyName := "workload1_3"

	actualValue, _ := redisHandler.GetTtl(newKeyName)

	fmt.Println(actualValue)

	assert.Equal(t, 15*time.Minute, actualValue)
}
