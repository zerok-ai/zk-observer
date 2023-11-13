package mocks

import (
	"errors"
	"regexp"
	"sort"
	"time"
)

type MockRedisHandler struct {
	store map[string]interface{}
	ttl   map[string]time.Duration
}

func NewMockRedisHandler() *MockRedisHandler {
	mockHandler := &MockRedisHandler{}
	mockHandler.InitializeRedisConn()
	return mockHandler
}

func (r *MockRedisHandler) InitializeRedisConn() error {
	r.store = make(map[string]interface{})
	r.ttl = make(map[string]time.Duration)
	return nil
}

func (r *MockRedisHandler) Get(key string) (string, error) {
	val, exists := r.store[key]
	if !exists {
		return "", nil
	}

	strVal, ok := val.(string)
	if !ok {
		return "", errors.New("value is not a string")
	}

	return strVal, nil
}

func (r *MockRedisHandler) GetAllKeys() ([]string, error) {
	keys := []string{}
	for key, _ := range r.store {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

func (r *MockRedisHandler) Set(key string, value interface{}) error {
	r.store[key] = value
	return nil
}

func (r *MockRedisHandler) SetNX(key string, value interface{}) error {
	return r.Set(key, value)
}

func (r *MockRedisHandler) HSet(key string, value interface{}) error {
	return r.Set(key, value)
}

func (r *MockRedisHandler) HMSet(key string, value interface{}) error {
	return r.Set(key, value)
}

func (r *MockRedisHandler) HMSetPipeline(key string, value map[string]string, expiration time.Duration) error {
	r.ttl[key] = expiration
	return r.Set(key, value)
}

func (r *MockRedisHandler) SetNXPipeline(key string, value interface{}, expiration time.Duration) error {
	r.ttl[key] = expiration
	return r.Set(key, value)
}

func (r *MockRedisHandler) SAddPipeline(key string, value interface{}, expiration time.Duration) error {
	r.ttl[key] = expiration
	return r.Set(key, value)
}

func (r *MockRedisHandler) SetWithTTL(key string, value interface{}, expiration time.Duration) error {
	r.ttl[key] = expiration
	return r.Set(key, value)
}

func (r *MockRedisHandler) GetTtl(key string) (time.Duration, error) {
	val, exists := r.ttl[key]
	if !exists {
		return -1, nil
	}
	return val, nil
}

func (r *MockRedisHandler) GetKeysByPattern(pattern string) ([]string, error) {
	re := regexp.MustCompile(pattern)
	var keys []string
	for k := range r.store {
		if re.MatchString(k) { // Simplified pattern matching
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (r *MockRedisHandler) RemoveKey(key string) error {
	delete(r.store, key)
	return nil
}

func (r *MockRedisHandler) RenameKeyWithTTL(oldKey string, newKey string, expiration time.Duration) error {
	val, exists := r.store[oldKey]
	if !exists {
		return errors.New("old key not found")
	}

	delete(r.store, oldKey)
	delete(r.ttl, oldKey)
	r.store[newKey] = val
	r.ttl[newKey] = expiration
	return nil
}

func (r *MockRedisHandler) FlushAllKeys() {
	r.store = make(map[string]interface{})
}
