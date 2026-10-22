// Copyright 2026 MinIO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build arm64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"testing"

	"github.com/minio/minlz/internal/cpu"
)

// simdTiers runs f as a subtest for each SIMD block tier, with that tier forced.
// The neon tier runs the per-operation loops, and neon-batch the batched loops.
func simdTiers(t *testing.T, f func(t *testing.T)) {
	neon, batch, variant := cpu.NEON, simdBatchNEON, simdBatchVariant
	defer func() { cpu.NEON, simdBatchNEON, simdBatchVariant = neon, batch, variant }()
	for _, tier := range []struct {
		name        string
		neon, batch bool
		variant     int
	}{{"go", false, false, 0}, {"neon", true, false, 0}, {"neon-batch", true, true, 0},
		{"neon-batch-noahead", true, true, 1}, {"neon-batch-plain", true, true, 2}} {
		cpu.NEON, simdBatchNEON, simdBatchVariant = tier.neon, tier.batch, tier.variant
		t.Run(tier.name, f)
	}
}
