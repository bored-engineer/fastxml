package fastxml

import "bytes"

// IsProcInst determines if a []byte is proc inst (ex: <?target inst>)
func IsProcInst(b []byte) bool {
	return b[1] == '?'
}

// ProcInst extracts the target and inst from a ProcInst (ex: `<?target inst>` -> (`target`, `inst`))
func ProcInst(b []byte) (target []byte, inst []byte) {
	if idx := bytes.IndexByte(b, ' '); idx != -1 {
		return b[2:idx], b[idx+1 : len(b)-2]
	}
	return b[2 : len(b)-2], nil
}
