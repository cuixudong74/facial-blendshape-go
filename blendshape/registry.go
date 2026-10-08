package blendshape

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Registry 为形变目标的并发安全注册表，支持运行时热加载与替换。
//
// 采用写时复制 + 原子指针：写操作在互斥锁内构造完整的新版本后一次性
// 原子发布；读路径无锁，每帧通过 Snapshot 取得一个不可变的完整版本。
// 因此替换期间正在进行的评估继续读旧版本，任一帧只会看到某个
// 已完整加载的版本，绝不会读到半成品状态。
type Registry struct {
	mu      sync.Mutex // 仅写路径
	cur     atomic.Pointer[targetSet]
	meshDim int // 基础网格顶点数，用于维度校验
}

type targetSet struct {
	targets map[string]*MorphTarget // 发布后不可变
	version uint64
}

// NewRegistry 创建注册表。vertexCount 为基础网格顶点数。
func NewRegistry(vertexCount int) *Registry {
	r := &Registry{meshDim: vertexCount}
	r.cur.Store(&targetSet{targets: map[string]*MorphTarget{}})
	return r
}

// Load 加载新目标；重名返回错误（替换请用 Replace）。
func (r *Registry) Load(t *MorphTarget) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.cur.Load()
	if _, exists := old.targets[t.Name]; exists {
		return fmt.Errorf("blendshape: target %q already loaded", t.Name)
	}
	return r.publishLocked(old, t)
}

// Replace 原子替换或新增目标。新版本先完整校验再发布。
func (r *Registry) Replace(t *MorphTarget) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.publishLocked(r.cur.Load(), t)
}

// publishLocked 校验目标并发布包含它的新版本。调用方须持有 r.mu。
func (r *Registry) publishLocked(old *targetSet, t *MorphTarget) error {
	if err := validateTarget(t, r.meshDim); err != nil {
		return fmt.Errorf("blendshape: load %q rejected: %w", t.Name, err)
	}
	next := make(map[string]*MorphTarget, len(old.targets)+1)
	for k, v := range old.targets {
		next[k] = v
	}
	next[t.Name] = t
	r.cur.Store(&targetSet{targets: next, version: old.version + 1})
	return nil
}

// Remove 移除目标；不存在时为空操作。
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.cur.Load()
	if _, ok := old.targets[name]; !ok {
		return
	}
	next := make(map[string]*MorphTarget, len(old.targets)-1)
	for k, v := range old.targets {
		if k != name {
			next[k] = v
		}
	}
	r.cur.Store(&targetSet{targets: next, version: old.version + 1})
}

// Snapshot 返回当前完整版本的只读视图与版本号（单调递增）。
// 返回的 map 不可修改；调用方在整帧内持有它即持有该版本。
func (r *Registry) Snapshot() (map[string]*MorphTarget, uint64) {
	s := r.cur.Load()
	return s.targets, s.version
}
