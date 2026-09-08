package arrayproof

import "testing"

func TestPick(t *testing.T) {
	if Pick(-7, 0, 42, 0) != -7 || Pick(-7, 0, 42, 1) != 0 || Pick(-7, 0, 42, 2) != 42 {
		t.Fatal("fixed array index returned the wrong element")
	}
}

func TestPickPanicsOutsideBounds(t *testing.T) {
	for _, index := range []int64{-1, 3} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("index %d did not panic", index)
				}
			}()
			Pick(1, 2, 3, index)
		}()
	}
}
