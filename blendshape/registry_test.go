package blendshape

import (
	"sync"
	"testing"
)

func TestRegistryLoadReplaceRemove(t *testing.T) {
	r := NewRegistry(2)
	if err := r.Load(makeTarget("a", LayerBase, 2, 1, 0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := r.Load(makeTarget("a", LayerBase, 2, 2, 0, 0)); err == nil {
		t.Fatal("duplicate Load should fail")
	}
	if err := r.Replace(makeTarget("a", LayerBase, 2, 3, 0, 0)); err != nil {
		t.Fatal(err)
	}
	targets, v1 := r.Snapshot()
	if targets["a"].Deltas[0] != 3 {
		t.Fatalf("replace not visible, delta = %v", targets["a"].Deltas[0])
	}
	r.Remove("a")
	_, v2 := r.Snapshot()
	if v2 <= v1 {
		t.Fatalf("version not increasing: %v -> %v", v1, v2)
	}
	targets2, _ := r.Snapshot()
	if _, ok := targets2["a"]; ok {
		t.Fatal("remove not visible")
	}
	// 旧快照不受后续修改影响（写时复制）。
	if _, ok := targets["a"]; !ok {
		t.Fatal("old snapshot mutated by Remove")
	}
}

func TestRegistryRejectsCorrupt(t *testing.T) {
	r := NewRegistry(2)
	bad := makeTarget("bad", LayerBase, 3, 1, 1, 1) // 顶点数不符
	if err := r.Load(bad); err == nil {
		t.Fatal("dimension mismatch should be rejected")
	}
	if _, v := r.Snapshot(); v != 0 {
		t.Fatal("rejected load must not bump version")
	}
}

// 热替换期间并发读取：任一帧只能读到某个完整版本，绝不出现半成品。
func TestRegistryHotSwapConsistency(t *testing.T) {
	r := NewRegistry(1)
	if err := r.Load(makeTarget("x", LayerBase, 1, 1, 1, 1)); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})

	// 写方：不断用自洽的新版本替换（delta 三个分量始终相等）。
	var writers sync.WaitGroup
	for w := 0; w < 4; w++ {
		writers.Add(1)
		go func(v float32) {
			defer writers.Done()
			for i := 0; i < 200; i++ {
				_ = r.Replace(makeTarget("x", LayerBase, 1, v, v, v))
			}
		}(float32(w + 2))
	}

	// 读方：每帧快照内数据必须自洽。
	var readers sync.WaitGroup
	for i := 0; i < 8; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				targets, _ := r.Snapshot()
				tg := targets["x"]
				if tg == nil {
					continue
				}
				d := tg.Deltas
				if d[0] != d[1] || d[1] != d[2] {
					t.Errorf("torn read: deltas %v", d)
					return
				}
			}
		}()
	}

	writers.Wait() // 写方有限次，先完成
	close(stop)    // 再通知读方退出
	readers.Wait()
}
