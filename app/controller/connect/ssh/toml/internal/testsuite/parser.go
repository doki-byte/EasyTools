package testsuite

import (
	toml2 "EasyTools/app/controller/connect/ssh/toml"
	"bytes"
	"encoding/json"
	"fmt"
)

type parser struct{}

func (p parser) Decode(input string) (output string, outputIsError bool, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			switch rr := r.(type) {
			case error:
				retErr = rr
			default:
				retErr = fmt.Errorf("%s", rr)
			}
		}
	}()

	var v any

	if err := toml2.Unmarshal([]byte(input), &v); err != nil {
		return err.Error(), true, nil
	}

	j, err := json.MarshalIndent(addTag("", v), "", "  ")
	if err != nil {
		return "", false, retErr
	}

	return string(j), false, retErr
}

func (p parser) Encode(input string) (output string, outputIsError bool, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			switch rr := r.(type) {
			case error:
				retErr = rr
			default:
				retErr = fmt.Errorf("%s", rr)
			}
		}
	}()

	var tmp any
	err := json.Unmarshal([]byte(input), &tmp)
	if err != nil {
		return "", false, err
	}

	rm, err := rmTag(tmp)
	if err != nil {
		return err.Error(), true, retErr
	}

	buf := new(bytes.Buffer)
	err = toml2.NewEncoder(buf).Encode(rm)
	if err != nil {
		return err.Error(), true, retErr
	}

	return buf.String(), false, retErr
}
