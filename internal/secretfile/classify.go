package secretfile

import (
	"bytes"
	"fmt"
	"os"

	"github.com/jrpbuilds/keyseal/internal/schema"
)

// State describes the on-disk state of a Keyseal-managed secret file.
type State string

const (
	StateMissing     State = "missing"
	StatePlaceholder State = "placeholder"
	StateEncrypted   State = "encrypted"
	StatePlaintext   State = "plaintext"
)

// Classification summarizes how a secret file should be handled before any
// decrypt attempt is made.
type Classification struct {
	State State
	Raw   []byte
}

// Classify inspects an encrypted-path file and distinguishes missing files,
// empty placeholders, SOPS-encrypted content, and non-empty plaintext content.
func Classify(path string) (Classification, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Classification{State: StateMissing}, nil
		}
		return Classification{}, fmt.Errorf("read secret file: %w", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return Classification{State: StatePlaceholder, Raw: raw}, nil
	}
	if schema.HasSOPSMetadata(raw) {
		return Classification{State: StateEncrypted, Raw: raw}, nil
	}
	return Classification{State: StatePlaintext, Raw: raw}, nil
}
