package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

// AzureBlobstore implements the Blob interface and provides the ability
// write files to Azure Blob Storage.
type AzureBlobstore struct {
	containerName string
	serviceURL    *azblob.ServiceURL
}

// NewAzureBlobstore creates a storage client, suitable for use with
// serverenv.ServerEnv.
func NewAzureBlobstore(ctx context.Context, containerName string, accountName, accountKey string) (*AzureBlobstore, error) {

	primaryURLRaw := fmt.Sprintf("https://%s.blob.core.windows.net", accountName)
	primaryURL, err := url.Parse(primaryURLRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL %v: %v", primaryURLRaw, err)
	}

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	serviceURL := azblob.NewServiceURL(*primaryURL, p)

	return &AzureBlobstore{
		serviceURL:    &serviceURL,
		containerName: containerName,
	}, nil
}

func (s *AzureBlobstore) CreateObject(ctx context.Context, name string, contents []byte) error {

	blobURL := s.serviceURL.NewContainerURL(s.containerName).NewBlockBlobURL(name)
	headers := azblob.BlobHTTPHeaders{}

	if _, err := azblob.UploadBufferToBlockBlob(ctx, contents, blobURL, azblob.UploadToBlockBlobOptions{
		BlobHTTPHeaders: headers,
	}); err != nil {
		return fmt.Errorf("storage.CreateObject: %w", err)
	}
	return nil
}

func (s *AzureBlobstore) GetObjectBytes(ctx context.Context, name string) ([]byte, error) {
	blobURL := s.serviceURL.NewContainerURL(s.containerName).NewBlobURL(name)

	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return []byte{}, nil
	}

	// NOTE: automatically retries are performed if the connection fails
	bodyStream := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 5})

	// read the body into a buffer
	downloadedData := bytes.Buffer{}
	_, err = downloadedData.ReadFrom(bodyStream)
	if err != nil {
		return []byte{}, nil
	}

	return downloadedData.Bytes(), nil
}
