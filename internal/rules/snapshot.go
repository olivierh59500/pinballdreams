package rules

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
)

// LampDigest fingerprints every original lamp flag and shared animation timer.
// It supports comparison with the separate interpreted original-code oracle.
func (r *Engine) LampDigest() string {
	return fmt.Sprintf("%x", sha256.Sum256(r.LampSnapshot()))
}

// LampSnapshot serializes the original shared timer and lamp record fields.
func (r *Engine) LampSnapshot() []byte {
	var data bytes.Buffer
	for _, g := range r.defs.Groups {
		binary.Write(&data, binary.BigEndian, uint32(g.ID))
		s := r.groups[g.ID]
		data.WriteByte(byte(s.Timer))
		if s.Direction {
			data.WriteByte(255)
		} else {
			data.WriteByte(0)
		}
	}
	ids := make([]int, 0, len(r.Lamps))
	for id := range r.Lamps {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		binary.Write(&data, binary.BigEndian, uint32(id))
		s := r.Lamps[id]
		data.WriteByte(s.Flags)
		data.WriteByte(s.Players)
		if s.Blink {
			data.WriteByte(255)
		} else {
			data.WriteByte(0)
		}
		if s.Phase {
			data.WriteByte(255)
		} else {
			data.WriteByte(0)
		}
	}
	return data.Bytes()
}
