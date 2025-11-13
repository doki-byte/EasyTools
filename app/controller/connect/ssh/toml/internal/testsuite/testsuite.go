// Package testsuite provides helper functions for interoperating with the
// language-agnostic TOML test suite at github.com/BurntSushi/toml-test.
package testsuite

import (
	toml2 "EasyTools/app/controller/connect/ssh/toml"
	"encoding/json"
	"fmt"
	"os"
)

// Marshal is a helpfer function for calling toml.Marshal
//
// Only needed to avoid package import loops.
func Marshal(v any) ([]byte, error) {
	return toml2.Marshal(v)
}

// Unmarshal is a helper function for calling toml.Unmarshal.
//
// Only needed to avoid package import loops.
func Unmarshal(data []byte, v any) error {
	return toml2.Unmarshal(data, v)
}

// ValueToTaggedJSON takes a data structure and returns the tagged JSON
// representation.
func ValueToTaggedJSON(doc any) ([]byte, error) {
	return json.MarshalIndent(addTag("", doc), "", "  ")
}

// DecodeStdin is a helper function for the toml-test binary interface.  TOML input
// is read from STDIN and a resulting tagged JSON representation is written to
// STDOUT.
func DecodeStdin() error {
	var decoded map[string]any

	if err := toml2.NewDecoder(os.Stdin).Decode(&decoded); err != nil {
		return fmt.Errorf("Error decoding TOML: %s", err)
	}

	j := json.NewEncoder(os.Stdout)
	j.SetIndent("", "  ")
	if err := j.Encode(addTag("", decoded)); err != nil {
		return fmt.Errorf("Error encoding JSON: %s", err)
	}

	return nil
}

// EncodeStdin is a helper function for the toml-test binary interface.  Tagged
// JSON is read from STDIN and a resulting TOML representation is written to
// STDOUT.
func EncodeStdin() error {
	var j any
	err := json.NewDecoder(os.Stdin).Decode(&j)
	if err != nil {
		return err
	}

	rm, err := rmTag(j)
	if err != nil {
		return fmt.Errorf("removing tags: %w", err)
	}

	return toml2.NewEncoder(os.Stdout).Encode(rm)
}
