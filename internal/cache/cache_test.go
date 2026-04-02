package cache

import (
	"strings"
	"testing"
	"time"
)

func TestCacheStop(t *testing.T) {
	c := New(100, time.Minute)
	c.Set("key1", "value1")

	// Verify item is set
	val, found := c.Get("key1")
	if !found || val != "value1" {
		t.Errorf("expected to find key1")
	}

	// Stop should not panic
	c.Stop()
	c.Stop() // Calling multiple times should not panic
	c.Stop()

	// Give goroutine time to stop
	time.Sleep(10 * time.Millisecond)

	// Cache should still be accessible
	val, found = c.Get("key1")
	if !found || val != "value1" {
		t.Errorf("expected to still find key1")
	}
}

func TestCacheCleanup(t *testing.T) {
	c := New(100, 50*time.Millisecond)
	c.Set("key1", "value1")

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Item should be expired
	_, found := c.Get("key1")
	if found {
		t.Errorf("expected key1 to be expired")
	}

	c.Stop()
}

func TestCacheStopMultiple(t *testing.T) {
	c := New(100, time.Minute)

	// Multiple calls should not panic
	for i := 0; i < 10; i++ {
		c.Stop()
	}
}
func TestCache_Get(t *testing.T) {
	c := New(100, time.Minute)
	defer c.Stop()

	c.Set("key1", "value1")
	c.SetWithTTL("expired", "value", -time.Millisecond)

	tests := []struct {
		name      string
		key       string
		wantVal   interface{}
		wantFound bool
		hits      int64
		misses    int64
	}{
		{
			name:      "存在的key正常返回",
			key:       "key1",
			wantVal:   "value1",
			wantFound: true,
			hits:      1,
			misses:    0,
		},
		{
			name:      "不存在的key返回not found",
			key:       "not_exist",
			wantVal:   nil,
			wantFound: false,
			hits:      1,
			misses:    1,
		},
		{
			name:      "已过期的key返回not found",
			key:       "expired",
			wantVal:   nil,
			wantFound: false,
			hits:      1,
			misses:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotFound := c.Get(tt.key)
			if gotFound != tt.wantFound || gotVal != tt.wantVal {
				t.Errorf("Get(%q) = (%v, %v), want (%v, %v)", tt.key, gotVal, gotFound, tt.wantVal, tt.wantFound)
			}
		})
	}

	// 验证命中次数统计
	if c.hitCount != 1 {
		t.Errorf("hitCount = %d, want 1", c.hitCount)
	}
	if c.missCount != 2 {
		t.Errorf("missCount = %d, want 2", c.missCount)
	}

	// 确认过期key已被删除（再次Get应miss）
	if _, found := c.Get("expired"); found {
		t.Error("expected expired key to be deleted")
	}
}

func TestCache_Set(t *testing.T) {
	c := New(100, time.Minute)
	defer c.Stop()

	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{
			name:  "正常设置字符串值",
			key:   "str",
			value: "hello",
		},
		{
			name:  "正常设置整数值",
			key:   "int",
			value: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c.Set(tt.key, tt.value)
			got, found := c.Get(tt.key)
			if !found {
				t.Errorf("expected to find key %q after Set", tt.key)
			}
			if got != tt.value {
				t.Errorf("Get(%q) = %v, want %v", tt.key, got, tt.value)
			}
		})
	}
}

func TestCache_SetWithTTL(t *testing.T) {
	c := New(100, time.Minute)
	defer c.Stop()

	c.SetWithTTL("short", "value1", 50*time.Millisecond)
	c.SetWithTTL("long", "value2", time.Minute)

	// 立即检查，两个key都应该存在
	if _, found := c.Get("short"); !found {
		t.Error("expected short TTL key to exist immediately")
	}
	if _, found := c.Get("long"); !found {
		t.Error("expected long TTL key to exist immediately")
	}

	// 等待短TTL过期
	time.Sleep(100 * time.Millisecond)

	// short 应该过期，long 仍然存在
	if _, found := c.Get("short"); found {
		t.Error("expected short TTL key to be expired")
	}
	if _, found := c.Get("long"); !found {
		t.Error("expected long TTL key to still exist")
	}
}

func TestCache_Delete(t *testing.T) {
	c := New(100, time.Minute)
	defer c.Stop()

	tests := []struct {
		name string
		key  string
		set  bool
	}{
		{
			name: "删除存在的key",
			key:  "exist",
			set:  true,
		},
		{
			name: "删除不存在的key不panic",
			key:  "not_exist",
			set:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				c.Set(tt.key, "value")
			}
			c.Delete(tt.key)
			if _, found := c.Get(tt.key); found {
				t.Errorf("expected key %q to be deleted", tt.key)
			}
		})
	}
}

func TestCache_Clear(t *testing.T) {
	c := New(100, time.Minute)
	defer c.Stop()

	c.Set("key1", "value1")
	c.Set("key2", "value2")

	// 先产生一些命中和未命中
	c.Get("key1")
	c.Get("not_exist")

	c.Clear()

	// 清空后所有key都不存在
	if _, found := c.Get("key1"); found {
		t.Error("expected key1 to be cleared")
	}
	if _, found := c.Get("key2"); found {
		t.Error("expected key2 to be cleared")
	}

	// 清空后stats中的size应为0（hit/miss统计保留）
	stats := c.Stats()
	if stats["size"] != 0 {
		t.Errorf("stats size = %d, want 0", stats["size"])
	}
	if stats["max_size"] != 100 {
		t.Errorf("stats max_size = %d, want 100", stats["max_size"])
	}
}

func TestCache_Eviction(t *testing.T) {
	c := New(3, time.Minute)
	defer c.Stop()

	// 设置已过期项目，evictOldest 会按最早过期时间淘汰
	c.SetWithTTL("old1", "old1_value", -time.Hour)
	c.SetWithTTL("old2", "old2_value", -time.Minute)
	c.SetWithTTL("old3", "old3_value", -time.Second)

	// 每次 Set 触发 evictOldest，依次淘汰最老的过期项
	c.Set("new1", "value1")

	// 验证 old1 已被移除（最早过期）
	if _, found := c.Get("old1"); found {
		t.Error("expected old1 to be evicted after first Set")
	}

	c.Set("new2", "value2")

	// 验证 old2 已被移除（第二早过期）
	if _, found := c.Get("old2"); found {
		t.Error("expected old2 to be evicted after second Set")
	}

	c.Set("new3", "value3")

	// 验证 old3 已被移除（第三早过期）
	if _, found := c.Get("old3"); found {
		t.Error("expected old3 to be evicted after third Set")
	}

	// 淘汰后 size 不超过 maxSize
	if len(c.items) > c.maxSize {
		t.Errorf("cache size = %d, want <= %d", len(c.items), c.maxSize)
	}

	// 确认新的 key 都存在
	for _, key := range []string{"new1", "new2", "new3"} {
		if _, found := c.Get(key); !found {
			t.Errorf("expected key %q to exist after eviction", key)
		}
	}
}

func TestCache_HitRate(t *testing.T) {
	c := New(100, time.Minute)
	defer c.Stop()

	// 初始命中率为0
	if rate := c.HitRate(); rate != 0 {
		t.Errorf("initial hit rate = %v, want 0", rate)
	}

	c.Set("key1", "value1")
	c.Get("key1")      // hit
	c.Get("key1")      // hit
	c.Get("not_exist") // miss

	// 命中和未命中后计算正确: 2 hits / 3 total = 0.666...
	want := 2.0 / 3.0
	if got := c.HitRate(); got != want {
		t.Errorf("hit rate = %v, want %v", got, want)
	}
}

func TestCache_Stats(t *testing.T) {
	c := New(100, time.Minute)
	defer c.Stop()

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Get("key1")      // hit
	c.Get("not_exist") // miss

	stats := c.Stats()

	// 返回的stats包含所有字段
	expectedKeys := []string{"size", "max_size", "hit_count", "miss_count", "hit_rate"}
	for _, key := range expectedKeys {
		if _, ok := stats[key]; !ok {
			t.Errorf("stats missing key %q", key)
		}
	}

	// 数据准确性验证
	if stats["size"] != 2 {
		t.Errorf("stats[size] = %d, want 2", stats["size"])
	}
	if stats["max_size"] != 100 {
		t.Errorf("stats[max_size] = %d, want 100", stats["max_size"])
	}
	if stats["hit_count"] != int64(1) {
		t.Errorf("stats[hit_count] = %v, want 1", stats["hit_count"])
	}
	if stats["miss_count"] != int64(1) {
		t.Errorf("stats[miss_count] = %v, want 1", stats["miss_count"])
	}
	wantRate := 1.0 / 2.0
	if stats["hit_rate"] != wantRate {
		t.Errorf("stats[hit_rate] = %v, want %v", stats["hit_rate"], wantRate)
	}
}

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		args   []interface{}
		want   string
	}{
		{
			name:   "相同输入生成相同key",
			prefix: "test",
			args:   []interface{}{1, "a"},
			want:   generateKey("test", 1, "a"),
		},
		{
			name:   "prefix正确附加",
			prefix: "prefix",
			args:   []interface{}{"arg"},
		},
		{
			name:   "不同输入生成不同key",
			prefix: "test",
			args:   []interface{}{1, "b"},
		},
	}

	key1 := generateKey("test", 1, "a")
	key2 := generateKey("test", 1, "a")
	if key1 != key2 {
		t.Errorf("相同输入生成不同key: %q vs %q", key1, key2)
	}

	key3 := generateKey("test", 1, "b")
	if key1 == key3 {
		t.Errorf("不同输入生成相同key: %q vs %q", key1, key3)
	}

	key4 := generateKey("prefix", "arg")
	if !strings.HasPrefix(key4, "prefix:") {
		t.Errorf("expected key to have prefix 'prefix:', got %q", key4)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateKey(tt.prefix, tt.args...)
			if got == "" {
				t.Error("generateKey returned empty string")
			}
		})
	}
}

func TestPetCache(t *testing.T) {
	c := NewPetCache(50, time.Minute)
	defer c.Stop()

	// NewPetCache创建正常
	if c == nil {
		t.Fatal("NewPetCache returned nil")
	}
	if c.Cache == nil {
		t.Fatal("PetCache.Cache is nil")
	}

	// 验证构造函数参数正确记录到实例字段
	if c.maxSize != 50 {
		t.Errorf("PetCache maxSize = %d, want 50", c.maxSize)
	}
	if c.expiration != time.Minute {
		t.Errorf("PetCache expiration = %v, want 1m", c.expiration)
	}

	// GetPetKey生成正确key格式
	petKey := GetPetKey(123)
	if !strings.HasPrefix(petKey, "pet:") {
		t.Errorf("GetPetKey prefix incorrect: %q", petKey)
	}
	// 验证 GetPetKey 生成确定性的key（相同输入产生相同输出）
	petKey2 := GetPetKey(123)
	if petKey != petKey2 {
		t.Errorf("GetPetKey(123) inconsistent: %q vs %q", petKey, petKey2)
	}
	// 验证不同id生成不同key
	petKeyDifferent := GetPetKey(456)
	if petKey == petKeyDifferent {
		t.Errorf("GetPetKey should generate different keys for different ids: both %q", petKey)
	}

	// GetPetsListKey生成正确key格式
	listKey := GetPetsListKey(1, 10, "cat")
	if !strings.HasPrefix(listKey, "pets_list:") {
		t.Errorf("GetPetsListKey prefix incorrect: %q", listKey)
	}
	// 验证 GetPetsListKey 生成确定性的key（相同参数产生相同输出）
	listKey2 := GetPetsListKey(1, 10, "cat")
	if listKey != listKey2 {
		t.Errorf("GetPetsListKey(1, 10, \"cat\") inconsistent: %q vs %q", listKey, listKey2)
	}
	// 验证不同参数生成不同key
	listKeyDifferent := GetPetsListKey(2, 10, "cat")
	if listKey == listKeyDifferent {
		t.Errorf("GetPetsListKey should generate different keys for different page: both %q", listKey)
	}

	// 相同参数生成相同key
	if GetPetKey(123) != GetPetKey(123) {
		t.Error("GetPetKey相同输入生成不同key")
	}
	if GetPetsListKey(1, 10, "cat") != GetPetsListKey(1, 10, "cat") {
		t.Error("GetPetsListKey相同输入生成不同key")
	}
}

func TestSessionCache(t *testing.T) {
	c := NewSessionCache()
	defer c.Stop()

	// NewSessionCache创建正常
	if c == nil {
		t.Fatal("NewSessionCache returned nil")
	}
	if c.Cache == nil {
		t.Fatal("SessionCache.Cache is nil")
	}

	// 默认配置正确（10000容量，30分钟过期）
	if c.maxSize != 10000 {
		t.Errorf("session cache maxSize = %d, want 10000", c.maxSize)
	}
	if c.expiration != 30*time.Minute {
		t.Errorf("session cache expiration = %v, want 30m", c.expiration)
	}
}
