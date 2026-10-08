# 性能说明

## 复杂度

设 V = 顶点数，T = 参与求值的目标数。

- 单次分层求值：**O(V·T)** 次乘加；每层只遍历一次激活目标。
- 权重归一化/排序簿记：O(T log T)（仅目标数量，与 V 无关）。
- 动画采样：每轨 O(log K)（K = 关键帧数，二分查找），一个 Clip O(Σlog K)。
- 快照发布：O(V·T) 拷贝一次（仅发生在热替换时，不在逐帧路径）。
- 逐帧路径**不拷贝中性网格、不深拷贝目标**；目标来自不可变快照，直接读。

## 内存与分配

- 调用方复用输出缓冲（`Evaluate(..., out []Vec3)`）时，逐帧热路径唯一的
  顶点级写就是这份 out；其余分配仅是每层的 map/小切片，**只随 T 增长，
  与 V 无关**。
- 引擎缓存“当前版本的有序目标切片”，只有**版本变更**时才重新分配，
  而不是每帧。

## 基准（本机：Xeon Platinum 8255C @ 2.5GHz, Go）

数值随机器浮动，量级可复现：

```
BenchmarkMix_1k_verts_8_targets-4      ~  70 µs/op    ~1.4 KB/op   18 allocs/op
BenchmarkMix_10k_verts_32_targets-4    ~ 3.0 ms/op    ~9.8 KB/op   33 allocs/op
BenchmarkMix_50k_verts_64_targets-4    ~ 39  ms/op     ~19 KB/op   38 allocs/op
BenchmarkEngineFrame_10k_verts_32_targets-4 ~ 3.1 ms/op ~8.3 KB/op 58 allocs/op
```

关键观察：

- 时间随 V·T 线性放大；
- **每帧分配字节几乎不随 V 增长**（1k→50k 顶点，分配仅从 1.4KB 到 19KB，
  且主要来自目标数量 8→64），说明没有每帧复制整份网格；
- 50k 顶点 × 64 目标约 40ms 单线程；常见面部 rig（1万级顶点、几十个目标）
  在 3ms 量级，满足实时帧率；多角色可通过每角色独立引擎并行
  （求值无共享可变状态）。

复现：

```bash
go test ./pkg/blendshape/ -bench . -benchmem
go test ./pkg/engine/   -bench . -benchmem
```
