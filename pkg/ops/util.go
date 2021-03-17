package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/orion-labs/genkeyset/pkg/genkeyset"
	"github.com/pkg/errors"
	"io/ioutil"
	"text/template"
	"time"
)

// RetryUntil takes a function, and calls it every 20 seconds until it succeeds.  Useful for polling endpoints in k8s that will eventually start working.  Returns an error if the provided timeoutMinutes elapses.  Otherwise returns the elapsed duration from start to finish.
func RetryUntil(thing func() (err error), timeoutMinutes int) (elapsed time.Duration, err error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(int32(timeoutMinutes))*time.Minute)

	defer cancel()

	statusReady := false

	for {
		select {
		case <-time.After(20 * time.Second):

			err = thing()
			if err != nil {
				ts := time.Now()
				h, m, s := ts.Clock()
				// print the timestamp, and the error from the thing() function
				fmt.Printf("  %02d:%02d:%02d %s.\n", h, m, s, err)
			} else {
				statusReady = true
			}

		case <-ctx.Done():
			err = errors.New("Timeout exceeded")
			finish := time.Now()

			elapsed = finish.Sub(start)

			return elapsed, err
		}

		if statusReady {
			break
		}
	}

	finish := time.Now()

	elapsed = finish.Sub(start)

	return elapsed, err
}

// CreateConfig Creates an orion-ptt-system kots config file from a local template.  The template itself is not distributed with this package to avoid leaking sensitive information.  To get one, you'll have to purchase an Orion PTT System license.
func (s *Stack) CreateConfig() (content string, err error) {
	keyset, err := genkeyset.GenerateKeySet(3)
	if err != nil {
		err = errors.Wrapf(err, "failed to generate keyset")
		return content, err
	}

	jsonbuf, err := json.Marshal(keyset)
	if err != nil {
		err = errors.Wrapf(err, "failed to marshall JWK KeySet into json")
		return content, err
	}

	tmplBytes, err := ioutil.ReadFile(s.Config.ConfigTemplate)
	if err != nil {
		err = errors.Wrapf(err, "failed reading template file %q", s.Config.ConfigTemplate)
		return content, err
	}

	tmpl, err := template.New("stack config").Parse(string(tmplBytes))
	if err != nil {
		err = errors.Wrapf(err, "failed to create template")
	}

	contentBytes := make([]byte, 0)

	buf := bytes.NewBuffer(contentBytes)

	data := OnpremConfig{
		Keystore:  string(jsonbuf),
		StackName: s.Config.StackName,
	}

	err = tmpl.Execute(buf, data)
	if err != nil {
		err = errors.Wrapf(err, "failed to execute template")
		return content, err
	}

	content = buf.String()

	return content, err
}
