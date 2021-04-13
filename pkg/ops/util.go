package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mitchellh/go-homedir"
	"github.com/orion-labs/genkeyset/pkg/genkeyset"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"regexp"
	"text/template"
	"time"
)

const DEFAULT_TEMPLATE_FILE = ".orion-ptt-system.tmpl"
const DEFAULT_NETWORK_CONFIG_FILE = ".orion-ptt-system-network.json"

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

	h, err := homedir.Dir()
	if err != nil {
		err = errors.Wrapf(err, "failed to detect homedir")
		return content, err
	}

	templatePath := s.Config.ConfigTemplate
	defaultPath := fmt.Sprintf("%s/%s", h, DEFAULT_TEMPLATE_FILE)

	isS3, s3Meta := S3Url(templatePath)

	// Look at the templatePath.  If it's an s3 url, fetch it, and stick it in the default location
	if isS3 {
		fmt.Printf("Fetching config template from S3.\n")
		err = FetchFileS3(s3Meta, defaultPath)
		if err != nil {
			err = errors.Wrapf(err, "failed to fetch template from %s", templatePath)
			return content, err
		}

		templatePath = defaultPath
	} else if isGit(templatePath) {
		repo, path := SplitRepoPath(templatePath)
		fmt.Printf("pulling templates from git.  Repo: %s Path: %s\n", repo, path)
		gitContent, err := GitContent(repo, path)
		if err != nil {
			err = errors.Wrapf(err, "error cloning %s", repo)
			return content, err
		}

		err = ioutil.WriteFile(defaultPath, gitContent, 0644)
		if err != nil {
			err = errors.Wrapf(err, "failed to write file to %s", defaultPath)
			return content, err
		}

		templatePath = defaultPath

	} else {
		fmt.Printf("Using local config template file %s.\n", templatePath)
	}

	// read template from local file, which might have been written by us, or might have been placed there manually .  Either way we don't really care.  It's just a file at this point.
	tmplBytes, err := ioutil.ReadFile(templatePath)
	if err != nil {
		err = errors.Wrapf(err, "failed reading template file %q", templatePath)
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
		Domain:    s.Config.DNSDomain,
	}

	err = tmpl.Execute(buf, data)
	if err != nil {
		err = errors.Wrapf(err, "failed to execute template")
		return content, err
	}

	content = buf.String()

	return content, err
}

// FetchFileS3 fetches the config template from an s3 url.
func FetchFileS3(s3Meta S3Meta, filePath string) (err error) {
	awsSession, err := DefaultSession()
	if err != nil {
		err = errors.Wrapf(err, "failed to create s3 session")
		return err
	}

	downloader := s3manager.NewDownloader(awsSession)
	downloadOptions := &s3.GetObjectInput{
		Bucket: aws.String(s3Meta.Bucket),
		Key:    aws.String(s3Meta.Key),
	}

	outFile, err := os.Create(filePath)
	if err != nil {
		err = errors.Wrapf(err, "failed creating file %s", filePath)
		return err
	}

	defer outFile.Close()

	_, err = downloader.Download(outFile, downloadOptions)
	if err != nil {
		err = errors.Wrapf(err, "download failed")
		return err
	}

	return err
}

// S3Meta a struct for holding metadata for S3 Objects.  There's probably already a struct that holds this, but this is all I need.
type S3Meta struct {
	Bucket string
	Region string
	Key    string
	Url    string
}

// S3Url returns true, and a metadata struct if the url given appears to be in s3
func S3Url(url string) (ok bool, meta S3Meta) {
	// Check to see if it's an s3 URL.
	s3Url := regexp.MustCompile(`https?://(.*)\.s3\.(.*)\.amazonaws.com/?(.*)?`)

	matches := s3Url.FindAllStringSubmatch(url, -1)

	if len(matches) == 0 {
		return ok, meta
	}

	match := matches[0]

	if len(match) == 3 {
		meta = S3Meta{
			Bucket: match[1],
			Region: match[2],
			Url:    url,
		}

		ok = true
		return ok, meta

	} else if len(match) == 4 {
		meta = S3Meta{
			Bucket: match[1],
			Region: match[2],
			Key:    match[3],
			Url:    url,
		}

		ok = true
		return ok, meta
	}

	return ok, meta
}

// StringInSlice returns true if the given string is in the given slice
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
