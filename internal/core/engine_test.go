package core

import "testing"

func TestEngineOpts_Defaults(t *testing.T) {
	opts := EngineOpts{}
	if opts.DryRun != false {
		t.Fatal("expected DryRun false by default")
	}
	if opts.Limit != 0 {
		t.Fatal("expected Limit 0 by default")
	}
}

func TestEngineResult(t *testing.T) {
	r := &EngineResult{Success: true, Data: map[string]any{"key": "val"}}
	if !r.Success {
		t.Fatal("expected Success true")
	}
}
