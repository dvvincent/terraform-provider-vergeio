package file

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"

	"terraform-provider-vergeio/internal/provider/vergeio"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	FileEndpoint = vergeio.APIEndpoint + "/files"
	blockSize    = 262144 // 256KB chunks, matching verge-cli
)

type FileApi struct {
	client *vergeio.Client
}

func NewFileApi(c *vergeio.Client) *FileApi {
	return &FileApi{client: c}
}

type fileAPIModel struct {
	Id             interface{} `json:"$key,omitempty"`
	Name           string      `json:"name"`
	Description    string      `json:"description,omitempty"`
	Type           string      `json:"type"`
	Filesize       int64       `json:"filesize,omitempty"`
	AllocatedBytes string      `json:"allocated_bytes,omitempty"`
}

// getIdString extracts the ID as a string from various possible types
func getIdString(id interface{}) string {
	switch v := id.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (fa *FileApi) createFile(ctx context.Context, data *FileResourceModel) error {
	// First, download the file to get its size (needed for allocated_bytes)
	var fileSize int64
	var tmpFile *os.File
	var err error

	if !data.SourceUrl.IsNull() {
		tmpFile, fileSize, err = fa.downloadToTemp(ctx, data.SourceUrl.ValueString())
		if err != nil {
			return err
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()
	}

	// Create file record with allocated_bytes (key insight from verge-cli!)
	apiData := map[string]interface{}{
		"name": data.Name.ValueString(),
		"type": data.Type.ValueString(),
	}
	if data.Description.ValueString() != "" {
		apiData["description"] = data.Description.ValueString()
	}
	if fileSize > 0 {
		apiData["allocated_bytes"] = strconv.FormatInt(fileSize, 10)
	}

	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return err
	}

	apiResp, err := fa.client.Post(FileEndpoint, encodedBuffer)
	if err != nil {
		return err
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != 201 {
		body, _ := io.ReadAll(apiResp.Body)
		return fmt.Errorf("failed to create file record: status %d, body: %s", apiResp.StatusCode, string(body))
	}

	var vergeResp vergeio.VergeResponse
	if err := json.NewDecoder(apiResp.Body).Decode(&vergeResp); err != nil {
		return err
	}

	data.Id = types.StringValue(vergeResp.Key)
	tflog.Info(ctx, fmt.Sprintf("Created file record with ID %s", data.Id.ValueString()))

	// Now upload the file content in chunks
	if tmpFile != nil {
		threads := int(data.UploadThreads.ValueInt64())
		if threads < 1 {
			threads = 8 // Default
		}
		if err := fa.uploadChunked(ctx, data.Id.ValueString(), tmpFile, fileSize, threads); err != nil {
			return err
		}
	}

	// Read back to populate computed fields (description, filesize)
	return fa.readFile(ctx, data)
}

func (fa *FileApi) downloadToTemp(ctx context.Context, sourceUrl string) (*os.File, int64, error) {
	tflog.Info(ctx, fmt.Sprintf("Downloading file from %s", sourceUrl))

	tmpFile, err := os.CreateTemp("", "vergeio-upload-*")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create temp file: %v", err)
	}

	resp, err := http.Get(sourceUrl)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, 0, fmt.Errorf("failed to download from URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, 0, fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	tflog.Info(ctx, fmt.Sprintf("Downloading %d bytes...", resp.ContentLength))
	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, 0, fmt.Errorf("failed to save to temp file: %v", err)
	}

	// Seek back to start for upload
	if _, err := tmpFile.Seek(0, 0); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, 0, err
	}

	tflog.Info(ctx, fmt.Sprintf("Downloaded %d bytes to temp file", written))
	return tmpFile, written, nil
}

func (fa *FileApi) uploadChunked(ctx context.Context, fileId string, file *os.File, fileSize int64, threads int) error {
	tflog.Info(ctx, fmt.Sprintf("Starting parallel chunked upload of %d bytes to file ID %s with %d threads", fileSize, fileId, threads))

	totalChunks := fileSize / blockSize
	if fileSize%blockSize > 0 {
		totalChunks++
	}

	tflog.Info(ctx, fmt.Sprintf("Will upload %d chunks of %d bytes using %d parallel threads", totalChunks, blockSize, threads))

	// TLS config matching the client
	tlsConfig := &tls.Config{InsecureSkipVerify: fa.client.Insecure}

	// Setup concurrency control (matching verge-cli's pattern)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, threads)
	errChan := make(chan error, 1)
	doneChan := make(chan struct{})

	var uploadedChunks int64
	var progressMu sync.Mutex

	// Process all chunks
	for chunkNum := int64(0); chunkNum < totalChunks; chunkNum++ {
		select {
		case <-doneChan:
			// Error occurred, stop launching new goroutines
			break
		case err := <-errChan:
			close(doneChan)
			wg.Wait()
			return err
		default:
			// Acquire semaphore slot
			semaphore <- struct{}{}
			wg.Add(1)

			go func(chunk int64) {
				defer wg.Done()
				defer func() { <-semaphore }()

				offset := chunk * blockSize

				// Read chunk
				chunkSize := blockSize
				if offset+int64(chunkSize) > fileSize {
					chunkSize = int(fileSize - offset)
				}

				buffer := make([]byte, chunkSize)
				_, err := file.ReadAt(buffer, offset)
				if err != nil && err != io.EOF {
					select {
					case errChan <- fmt.Errorf("error reading file at position %d: %v", offset, err):
					default:
					}
					return
				}

				// Build chunk URL with filepos parameter
				chunkURL := fmt.Sprintf("https://%s/%s/%s?filepos=%d",
					fa.client.Host,
					FileEndpoint,
					url.PathEscape(fileId),
					offset)

				// Create request
				req, err := http.NewRequest("PUT", chunkURL, bytes.NewBuffer(buffer))
				if err != nil {
					select {
					case errChan <- fmt.Errorf("error creating chunk request: %v", err):
					default:
					}
					return
				}

				req.Header.Set("Content-Type", "application/octet-stream")
				req.Header.Set("Expect", "")
				req.SetBasicAuth(fa.client.Username, fa.client.Password)

				// Use a fresh connection for each chunk (matching verge-cli's approach)
				chunkClient := &http.Client{
					Transport: &http.Transport{
						TLSClientConfig:   tlsConfig,
						DisableKeepAlives: true,
					},
				}

				resp, err := chunkClient.Do(req)
				if err != nil {
					select {
					case errChan <- fmt.Errorf("error uploading chunk at position %d: %v", offset, err):
					default:
					}
					return
				}

				if resp.StatusCode >= 400 {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					select {
					case errChan <- fmt.Errorf("error uploading chunk at position %d: status %d, body: %s", offset, resp.StatusCode, string(body)):
					default:
					}
					return
				}
				resp.Body.Close()

				// Update progress
				progressMu.Lock()
				uploadedChunks++
				currentUploaded := uploadedChunks
				progressMu.Unlock()

				if currentUploaded%50 == 0 || currentUploaded == totalChunks {
					percentage := currentUploaded * 100 / totalChunks
					tflog.Info(ctx, fmt.Sprintf("Upload progress: %d%% (%d/%d chunks)", percentage, currentUploaded, totalChunks))
				}
			}(chunkNum)
		}
	}

	// Wait for all uploads to complete
	wg.Wait()

	// Check for any remaining errors
	select {
	case err := <-errChan:
		return err
	default:
	}

	tflog.Info(ctx, fmt.Sprintf("Parallel chunked upload complete! Uploaded %d chunks using %d threads", totalChunks, threads))
	return nil
}

func (fa *FileApi) readFile(ctx context.Context, data *FileResourceModel) error {
	endpoint := fmt.Sprintf("%s/%s", FileEndpoint, url.PathEscape(data.Id.ValueString()))
	apiResp, err := fa.client.Get(endpoint, &vergeio.Options{Fields: "$key,name,description,type,filesize"})

	if err != nil {
		if apiErr, ok := err.(vergeio.Error); ok && apiErr.StatusCode == 404 {
			data.Id = types.StringNull()
			return nil
		}
		return err
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == 404 {
		data.Id = types.StringNull()
		return nil
	}

	if apiResp.StatusCode != 200 {
		return fmt.Errorf("failed to read file: status %d", apiResp.StatusCode)
	}

	var apiModel fileAPIModel
	if err := json.NewDecoder(apiResp.Body).Decode(&apiModel); err != nil {
		return err
	}

	data.Name = types.StringValue(apiModel.Name)
	data.Description = types.StringValue(apiModel.Description)
	data.Type = types.StringValue(apiModel.Type)
	data.Filesize = types.Int64Value(apiModel.Filesize)

	return nil
}

func (fa *FileApi) deleteFile(ctx context.Context, data *FileResourceModel) error {
	endpoint := fmt.Sprintf("%s/%s", FileEndpoint, url.PathEscape(data.Id.ValueString()))
	apiResp, err := fa.client.Delete(endpoint)
	if err != nil {
		return err
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != 200 && apiResp.StatusCode != 404 {
		return fmt.Errorf("failed to delete file: status %d", apiResp.StatusCode)
	}

	return nil
}
