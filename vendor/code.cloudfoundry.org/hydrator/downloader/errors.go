package downloader

import "fmt"

type MaxLayerDownloadRetriesError struct {
	DiffID string
	SHA    string
}

func (e *MaxLayerDownloadRetriesError) Error() string {
	return fmt.Sprintf("Exceeded maximum download attempts for blob with diffID: %.8s, sha256: %.8s", e.DiffID, e.SHA)
}
