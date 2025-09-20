package parser

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
)

func parseChunkLlamaIndex(chunk []byte, apiPath string) error {
	reqBody := new(bytes.Buffer)

	multipartWriter := multipart.NewWriter(reqBody)
	writer, err := multipartWriter.CreateFormFile("file", "")
	if err != nil {
		return fmt.Errorf("failed to create multipart form file for chunk: %w", err)
	}

	_, err = writer.Write(chunk)
	if err != nil {
		return fmt.Errorf("failed to write chunk to multipart form file: %w", err)
	}

	client := &http.Client{}

	req, err := http.NewRequest(http.MethodPost, apiPath, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for chunk parsing: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to upload file for chunk parsing: %w", err)
	}

	return nil
}
