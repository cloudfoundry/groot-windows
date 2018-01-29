package registry

import (
	"fmt"

	digest "github.com/opencontainers/go-digest"
)

type SHAMismatchError struct {
	expected string
	actual   string
}

func (e *SHAMismatchError) Error() string {
	return fmt.Sprintf("sha256 mismatch: expected %s, got %s", e.expected, e.actual)
}

type DownloadError struct {
	Cause   error
	blobSHA string
}

func (e *DownloadError) Error() string {
	return fmt.Sprintf("failed downloading blob %.8s: %s", e.blobSHA, e.Cause.Error())
}

type DigestAlgorithmError struct {
	expected digest.Algorithm
	actual   digest.Algorithm
}

func (e *DigestAlgorithmError) Error() string {
	return fmt.Sprintf("invalid digest algorithm: expected %s, got %s", e.expected, e.actual)
}

type HTTPNotOKError struct {
	statusCode int
}

func (e *HTTPNotOKError) Error() string {
	return fmt.Sprintf("unsuccessful response from server: %d", e.statusCode)
}

type InvalidMediaTypeError struct {
	mediaType string
}

func (e *InvalidMediaTypeError) Error() string {
	return fmt.Sprintf("invalid media type: %s", e.mediaType)
}
