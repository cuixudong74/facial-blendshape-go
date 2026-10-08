// Package blendshape 提供面部形变目标（blendshape/morph target）驱动的
// 表情动画运行时：多路加权混合、分层组合、关键帧动画、热加载与并发驱动。
package blendshape

// Layer 标识形变目标所处的分层。
type Layer int

const (
	// LayerBase 基础形变层：权重归一化后加权混合。
	LayerBase Layer = iota
	// LayerAdditive 叠加形变层：在基础层结果之上按权重直接叠加。
	LayerAdditive
	// LayerCorrective 校正形变层：最后应用，用于修正前述层组合产生的伪影。
	LayerCorrective
)

// MorphTarget 表示一个形变目标：相对基础网格的逐顶点位移增量。
// Deltas 长度为 3*VertexCount，布局为 [x0 y0 z0 x1 y1 z1 ...]。
// 数据加载后视为不可变，热替换通过整体替换指针完成。
type MorphTarget struct {
	Name        string
	Layer       Layer
	Deltas      []float32
	VertexCount int
	// Drivers 仅对 LayerCorrective 有意义：校正目标的有效权重为
	// 自身权重与各驱动目标当前权重（基础层归一化后）的乘积。
	// 为空时有效权重即自身权重。
	Drivers []string
}

// Mesh 是基础网格数据，运行期只读。
type Mesh struct {
	Positions   []float32 // 长度 3*VertexCount
	VertexCount int
}
