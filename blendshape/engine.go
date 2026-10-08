package blendshape

import "sync"

// Engine 为表情动画运行时入口：组合注册表、动画通道与混合器。
//
// 并发模型：基础网格与已发布的目标数据只读；每个 goroutine 持有自己的
// Channel 与输出缓冲即可并发驱动，无需加锁，输出在相同输入下一致。
// channels 切片本身由互斥锁保护。
type Engine struct {
	reg     *Registry
	blender *Blender
	policy  NormalizePolicy

	mu       sync.Mutex // 仅保护 channels 切片
	channels []*Channel
}

// NewEngine 创建运行时。mesh 在运行期只读。
func NewEngine(mesh *Mesh) *Engine {
	return &Engine{
		reg:     NewRegistry(mesh.VertexCount),
		blender: NewBlender(mesh),
		policy:  DefaultNormalizePolicy,
	}
}

// Registry 返回目标注册表（用于热加载/替换）。
func (e *Engine) Registry() *Registry { return e.reg }

// Blender 返回混合器。
func (e *Engine) Blender() *Blender { return e.blender }

// SetNormalizePolicy 覆盖默认权重归一化策略。
func (e *Engine) SetNormalizePolicy(p NormalizePolicy) { e.policy = p }

// AddChannel 注册一路动画通道。
func (e *Engine) AddChannel(c *Channel) {
	e.mu.Lock()
	e.channels = append(e.channels, c)
	e.mu.Unlock()
}

// Advance 推进所有通道并清理已结束的 LoopOnce 通道。
func (e *Engine) Advance(dt float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	kept := e.channels[:0]
	for _, c := range e.channels {
		c.Advance(dt)
		if !c.Done() {
			kept = append(kept, c)
		}
	}
	// 清空尾部引用，避免泄漏。
	for i := len(kept); i < len(e.channels); i++ {
		e.channels[i] = nil
	}
	e.channels = kept
}

// Evaluate 评估当前帧：合并各通道权重 → 分层混合（基础层归一化），
// 结果写入 out（长度必须为 3*VertexCount，可跨帧复用，避免每帧整份拷贝）。
// 损坏/缺失目标按既定策略跳过，返回可定位到具体目标的问题列表（可为空）。
//
// 通道状态是引擎共享的可变状态，采样在锁内完成（代价 O(轨道数)）；
// 逐顶点混合只读不可变数据，在锁外并发执行。
// 整帧只取一次注册表快照，保证一帧内目标版本一致。
func (e *Engine) Evaluate(out []float32) IssueList {
	e.mu.Lock()
	weights := make(map[string]float64, 16)
	for _, c := range e.channels {
		c.Evaluate(weights)
	}
	e.mu.Unlock()

	targets, _ := e.reg.Snapshot()
	var issues IssueList
	e.blender.LayeredBlend(out, weights, targets, e.policy, &issues)
	return issues
}
