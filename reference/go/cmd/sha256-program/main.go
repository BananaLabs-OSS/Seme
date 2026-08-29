// Command sha256-program emits the bootstrap construction source for Seme's
// canonical SHA-256 file digester. Generated repetition is never authoritative;
// the lifted canonical Seme graph is.
package main

import "fmt"

var constants = [64]uint32{
	0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
	0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
	0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
	0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
	0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
	0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
	0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
	0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
}

func main() {
	fmt.Print(`# Generated SHA-256 bootstrap construction source.
ve 3
cp 7

fn rotr32 2 4
  lg 0 lg 1 sr
  lg 0 cu 32 lg 1 su sl or
  cu 4294967295 an re
fe

fn load32be 2 5
  lg 0 lg 1 bg cu 24 sl
  lg 0 lg 1 cu 1 ad bg cu 16 sl or
  lg 0 lg 1 cu 2 ad bg cu 8 sl or
  lg 0 lg 1 cu 3 ad bg or re
fe

fn append32be 2 4
  lg 0 lg 1 cu 24 sr cu 255 an bp ls 0
  lg 0 lg 1 cu 16 sr cu 255 an bp ls 0
  lg 0 lg 1 cu 8 sr cu 255 an bp ls 0
  lg 0 lg 1 cu 255 an bp re
fe

# round(a,b,c,d,e,f,g,h,w,k) -> (new-a << 32) | new-e
fn round 10 20
  lg 4 cu 6 ca rotr32 lg 4 cu 11 ca rotr32 xo
  lg 4 cu 25 ca rotr32 xo ls 10
  lg 4 lg 5 an lg 4 cu 4294967295 xo lg 6 an xo ls 11
  lg 7 lg 10 ad lg 11 ad lg 8 ad lg 9 ad
  cu 4294967295 an ls 12
  lg 0 cu 2 ca rotr32 lg 0 cu 13 ca rotr32 xo
  lg 0 cu 22 ca rotr32 xo ls 10
  lg 0 lg 1 an lg 0 lg 2 an xo lg 1 lg 2 an xo ls 11
  lg 10 lg 11 ad cu 4294967295 an ls 13
  lg 12 lg 13 ad cu 4294967295 an cu 32 sl
  lg 3 lg 12 ad cu 4294967295 an or re
fe

fn main 0 64
  ac cu 2 eq bf invocation
  cu 0 ag fr ls 0
  lg 0 bl ls 3
  cu 1048576 lg 3 lt bf size-ok
  cu 65 re
  lb size-ok
  cu 1048576 bn ls 1
  cu 0 ls 4
  lb copy-loop
  lg 4 lg 3 eq bf copy-byte ju pad-one
  lb copy-byte
  lg 1 lg 0 lg 4 bg bp ls 1
  lg 4 cu 1 ad ls 4 ju copy-loop
  lb pad-one
  lg 1 cu 128 bp ls 1
  lb pad-zero-loop
  lg 1 bl cu 8 ad cu 64 rm cu 0 eq bf pad-zero
  ju pad-length
  lb pad-zero
  lg 1 cu 0 bp ls 1 ju pad-zero-loop
  lb pad-length
  lg 3 cu 3 sl ls 5
  lg 1 cu 0 bp ls 1 lg 1 cu 0 bp ls 1
  lg 1 cu 0 bp ls 1 lg 1 cu 0 bp ls 1
  lg 1 lg 5 cu 24 sr cu 255 an bp ls 1
  lg 1 lg 5 cu 16 sr cu 255 an bp ls 1
  lg 1 lg 5 cu 8 sr cu 255 an bp ls 1
  lg 1 lg 5 cu 255 an bp ls 1

  cu 1779033703 ls 8 cu 3144134277 ls 9
  cu 1013904242 ls 10 cu 2773480762 ls 11
  cu 1359893119 ls 12 cu 2600822924 ls 13
  cu 528734635 ls 14 cu 1541459225 ls 15
  cu 0 ls 4
  lb block-loop
  lg 4 lg 1 bl eq bf block
  ju finish
  lb block
  lg 8 ls 16 lg 9 ls 17 lg 10 ls 18 lg 11 ls 19
  lg 12 ls 20 lg 13 ls 21 lg 14 ls 22 lg 15 ls 23
`)
	for t := 0; t < 64; t++ {
		w := 24 + t%16
		if t < 16 {
			fmt.Printf("  lg 1 lg 4 cu %d ad ca load32be ls %d\n", t*4, w)
		} else {
			w15 := 24 + (t-15)%16
			w2 := 24 + (t-2)%16
			w16 := 24 + (t-16)%16
			w7 := 24 + (t-7)%16
			fmt.Printf("  lg %d cu 7 ca rotr32 lg %d cu 18 ca rotr32 xo lg %d cu 3 sr xo ls 40\n", w15, w15, w15)
			fmt.Printf("  lg %d cu 17 ca rotr32 lg %d cu 19 ca rotr32 xo lg %d cu 10 sr xo ls 41\n", w2, w2, w2)
			fmt.Printf("  lg %d lg 40 ad lg %d ad lg 41 ad cu 4294967295 an ls %d\n", w16, w7, w)
		}
		fmt.Printf("  lg 16 lg 17 lg 18 lg 19 lg 20 lg 21 lg 22 lg 23 lg %d cu %d ca round ls 42\n", w, constants[t])
		fmt.Printf("  lg 22 ls 23 lg 21 ls 22 lg 20 ls 21 lg 42 cu 4294967295 an ls 20\n")
		fmt.Printf("  lg 18 ls 19 lg 17 ls 18 lg 16 ls 17 lg 42 cu 32 sr ls 16\n")
	}
	fmt.Print(`  lg 8 lg 16 ad cu 4294967295 an ls 8
  lg 9 lg 17 ad cu 4294967295 an ls 9
  lg 10 lg 18 ad cu 4294967295 an ls 10
  lg 11 lg 19 ad cu 4294967295 an ls 11
  lg 12 lg 20 ad cu 4294967295 an ls 12
  lg 13 lg 21 ad cu 4294967295 an ls 13
  lg 14 lg 22 ad cu 4294967295 an ls 14
  lg 15 lg 23 ad cu 4294967295 an ls 15
  lg 4 cu 64 ad ls 4 ju block-loop
  lb finish
  cu 32 bn ls 2
  lg 2 lg 8 ca append32be ls 2 lg 2 lg 9 ca append32be ls 2
  lg 2 lg 10 ca append32be ls 2 lg 2 lg 11 ca append32be ls 2
  lg 2 lg 12 ca append32be ls 2 lg 2 lg 13 ca append32be ls 2
  lg 2 lg 14 ca append32be ls 2 lg 2 lg 15 ca append32be ls 2
  cu 1 ag lg 2 fw re
  lb invocation
  cu 64 re
fe

en main
`)
}
