//go:build arm64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestSIMDBatchAB times decoding of MINLZ_AB_FILE with the old and batched NEON loops, interleaved in one process.
// MINLZ_AB_LEVELS, MINLZ_AB_BS (KiB), MINLZ_AB_MB (prefix), MINLZ_AB_ROUNDS, MINLZ_AB_DUR.
func TestSIMDBatchAB(t *testing.T) {
	file := os.Getenv("MINLZ_AB_FILE")
	if file == "" {
		t.Skip("MINLZ_AB_FILE not set")
	}
	env := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	ints := func(s string) (r []int) {
		for _, f := range strings.Split(s, ",") {
			v, err := strconv.Atoi(f)
			if err != nil {
				t.Fatal(err)
			}
			r = append(r, v)
		}
		return r
	}
	levels, bss := ints(env("MINLZ_AB_LEVELS", "4,5,6,7,262")), ints(env("MINLZ_AB_BS", "64,1024,4096"))
	mb, _ := strconv.Atoi(env("MINLZ_AB_MB", "128"))
	rounds, _ := strconv.Atoi(env("MINLZ_AB_ROUNDS", "5"))
	dur, _ := time.ParseDuration(env("MINLZ_AB_DUR", "1s"))
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > mb<<20 {
		data = data[:mb<<20]
	}
	name := file[strings.LastIndex(file, "/")+1:]
	type variant struct {
		name    string
		batch   bool
		variant int
	}
	variants := []variant{{"old", false, 0}, {"batch", true, 0}, {"noahead", true, 1}, {"plain", true, 2}}
	if only := os.Getenv("MINLZ_AB_ONLY"); only != "" {
		var keep []variant
		for _, v := range variants {
			if strings.Contains(","+only+",", ","+v.name+",") {
				keep = append(keep, v)
			}
		}
		variants = keep
	}
	defer func() { simdBatchNEON, simdBatchVariant = true, 0 }()
	for _, level := range levels {
		for _, bs := range bss {
			bs <<= 10
			var enc [][]byte
			size := 0
			for i := 0; i < len(data); i += bs {
				e, err := Encode(nil, data[i:min(i+bs, len(data))], level)
				if err != nil {
					t.Fatal(err)
				}
				enc = append(enc, e)
				size += len(e)
			}
			dst := make([]byte, bs)
			for _, v := range variants {
				simdBatchNEON, simdBatchVariant = v.batch, v.variant
				for i, e := range enc {
					got, err := Decode(dst, e)
					if err != nil || !bytes.Equal(got, data[i*bs:min((i+1)*bs, len(data))]) {
						t.Fatalf("%s level %d bs %d variant %s block %d: %v", name, level, bs, v.name, i, err)
					}
				}
			}
			for r := range rounds {
				for j := range variants {
					v := variants[(j+r)%len(variants)]
					simdBatchNEON, simdBatchVariant = v.batch, v.variant
					n := 0
					start := time.Now()
					for time.Since(start) < dur {
						for _, e := range enc {
							Decode(dst, e)
						}
						n++
					}
					el := time.Since(start)
					fmt.Printf("AB\t%s\t%d\t%d\t%s\t%d\t%.1f\t%.4f\n", name, level, bs>>10, v.name, r, float64(n*len(data))/el.Seconds()/1e6, float64(size)/float64(len(data)))
				}
			}
		}
	}
}
