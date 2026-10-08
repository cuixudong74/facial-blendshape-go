package blendshape

import (
	"sync"
	"testing"
)

func newTestEngine() *Engine {
	mesh := makeMesh(16)
	e := NewEngine(mesh)
	_ = e.Registry().Load(makeTarget("smile", LayerBase, 16, 0.1, 0, 0))
	_ = e.Registry().Load(makeTarget("jaw", LayerBase, 16, 0, 0.2, 0))
	return e
}

func smileAnim() *Animation {
	return &Animation{Name: "smile", Duration: 1.0, Tracks: []Track{
		{Target: "smile", Keys: []Keyframe{{0, 0}, {1, 1}}, Interp: InterpLinear},
		{Target: "jaw", Keys: []Keyframe{{0, 0}, {1, 0.5}}, Interp: InterpLinear},
	}}
}

func TestEngineEvaluateBasic(t *testing.T) {
	e := newTestEngine()
	e.AddChannel(NewChannel(smileAnim(), LoopOnce))
	e.Advance(0.5)
	out := make([]float32, 48)
	issues := e.Evaluate(out)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
	// t=0.5: smile=0.5, jaw=0.25，权重和 0.75 未超上限，不缩放。
	// 顶点 0: x += 0.5*0.1, y += 0.25*0.2
	if !close32(out[0], 0.05) || !close32(out[1], 0.05) {
		t.Fatalf("vertex0 = (%v,%v), want (0.05,0.05)", out[0], out[1])
	}
}

// 多 goroutine 并发驱动：各自通道 + 各自输出缓冲，结果与单线程一致。
func TestEngineConcurrentEvaluateConsistent(t *testing.T) {
	e := newTestEngine()
	const goroutines = 8
	const frames = 50

	// 参考：单线程结果。
	ref := make([][]float32, frames)
	{
		c := NewChannel(smileAnim(), LoopRepeat)
		for f := 0; f < frames; f++ {
			c.Advance(1.0 / 60.0)
			out := make([]float32, 48)
			w := map[string]float64{}
			c.Evaluate(w)
			targets, _ := e.Registry().Snapshot()
			var iss IssueList
			e.Blender().LayeredBlend(out, w, targets, DefaultNormalizePolicy, &iss)
			ref[f] = out
		}
	}

	var wg sync.WaitGroup
	errs := make(chan string, goroutines)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := NewChannel(smileAnim(), LoopRepeat)
			for f := 0; f < frames; f++ {
				c.Advance(1.0 / 60.0)
				out := make([]float32, 48)
				w := map[string]float64{}
				c.Evaluate(w)
				targets, _ := e.Registry().Snapshot()
				var iss IssueList
				e.Blender().LayeredBlend(out, w, targets, DefaultNormalizePolicy, &iss)
				for i := range out {
					if out[i] != ref[f][i] {
						errs <- "mismatch"
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for msg := range errs {
		t.Fatal(msg)
	}
}

// 动画播放期间并发热替换：Evaluate 不崩溃，输出始终来自某个完整版本。
func TestEngineEvaluateDuringHotSwap(t *testing.T) {
	e := newTestEngine()
	e.AddChannel(NewChannel(smileAnim(), LoopRepeat))
	stop := make(chan struct{})

	var writers sync.WaitGroup
	writers.Add(1)
	go func() {
		defer writers.Done()
		for i := 0; i < 100; i++ {
			v := float32(i%7) * 0.01
			_ = e.Registry().Replace(makeTarget("smile", LayerBase, 16, v, v, v))
		}
	}()

	var readers sync.WaitGroup
	for g := 0; g < 4; g++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			out := make([]float32, 48)
			for {
				select {
				case <-stop:
					return
				default:
				}
				e.Advance(1.0 / 60.0)
				_ = e.Evaluate(out)
				for _, v := range out {
					if v != v { // NaN 即半成品/损坏
						t.Errorf("NaN in output during hot swap")
						return
					}
				}
			}
		}()
	}
	writers.Wait()
	close(stop)
	readers.Wait()
}

// LoopOnce 通道播放完毕后被自动清理。
func TestEngineRemovesFinishedChannels(t *testing.T) {
	e := newTestEngine()
	e.AddChannel(NewChannel(smileAnim(), LoopOnce))
	e.Advance(2.0) // 超过 Duration=1
	e.Advance(0.1)
	e.mu.Lock()
	n := len(e.channels)
	e.mu.Unlock()
	if n != 0 {
		t.Fatalf("finished channel not removed, %d left", n)
	}
}
