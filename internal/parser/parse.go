package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	db "github.com/OptimusePrime/petagpt/internal/db"
	"github.com/OptimusePrime/petagpt/internal/sqlc"
	"github.com/spf13/viper"
)

const LLAMA_INDEX_API_BASE = "https://api.cloud.llamaindex.ai/api/v1"

type LlamaIndexParsingStatus string

const (
	STATUS_PENDING         LlamaIndexParsingStatus = "PENDING"
	STATUS_SUCCESS         LlamaIndexParsingStatus = "SUCCESS"
	STATUS_ERROR           LlamaIndexParsingStatus = "ERROR"
	STATUS_PARTIAL_SUCCESS LlamaIndexParsingStatus = "PARTIAL_SUCCESS"
	STATUS_CANCELLED       LlamaIndexParsingStatus = "CANCELLED"
)

type LlamaIndexParsingStatusResponse struct {
	ID           string                  `json:"id"`
	Status       LlamaIndexParsingStatus `json:"status"`
	ErrorCode    string                  `json:"error_code"`
	ErrorMessage string                  `json:"error_message"`
}

type LlamaIndexJobStatusRequest struct {
	JobID string `json:"job_id"`
}

const PARSING_PAGE_SEPARATOR = "\n@@RieSDIh6U5htthJY@@\n"

func ParseDocument(ctx context.Context, document []byte, dc *DocumentChunker) error {
	resp, err := uploadDocumentLlamaIndex(ctx, document)
	if err != nil {
		return err
	}

	status, err := waitJobDoneLlamaIndex(ctx, resp.ID)
	if err != nil || status != STATUS_SUCCESS {
		return fmt.Errorf("wait job failed: %w", err)
	}

	jobResult, err := getJobMarkdownResultLlamaIndex(resp.ID)
	if err != nil {
		return fmt.Errorf("markdown result failed: %w", err)
	}

	pages := strings.Split(jobResult.Markdown, PARSING_PAGE_SEPARATOR)

	for _, page := range pages {
		sentences, err := dc.SentenceSegmentText(ctx, page)
		if err != nil {
			return fmt.Errorf("segmentation failed: %w", err)
		}
		fmt.Println(sentences)
	}

	return nil
}

func uploadDocumentLlamaIndex(ctx context.Context, document []byte) (*LlamaIndexParsingStatusResponse, error) {
	reqBody := new(bytes.Buffer)

	multipartWriter := multipart.NewWriter(reqBody)
	err := multipartWriter.WriteField("page_separator", PARSING_PAGE_SEPARATOR)
	if err != nil {
		return nil, err
	}

	writer, err := multipartWriter.CreateFormFile("file", "document.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart form file for document: %w", err)
	}

	_, err = writer.Write(document)
	if err != nil {
		return nil, fmt.Errorf("failed to write document to multipart form file: %w", err)
	}
	err = multipartWriter.Close()
	if err != nil {
		return nil, err
	}

	fmt.Println(reqBody.Len())

	client := &http.Client{}

	apiPath := LLAMA_INDEX_API_BASE + "/parsing/upload"

	req, err := http.NewRequest(http.MethodPost, apiPath, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for document parsing: %w", err)
	}
	fmt.Println(viper.GetString("document_parser.api_key"))
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", viper.GetString("document_parser.api_key")))

	resp, err := client.Do(req)

	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to upload file for document parsing: %w (%s)", err, resp.Status)
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(respBytes))

	parsingStatus := new(LlamaIndexParsingStatusResponse)
	err = json.Unmarshal(respBytes, parsingStatus)
	if err != nil {
		return nil, err
	}

	return parsingStatus, nil
}

func checkJobStatusLlamaIndex(jobId string) (*LlamaIndexParsingStatusResponse, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	getJobReq := LlamaIndexJobStatusRequest{
		JobID: jobId,
	}

	jsonBytes, err := json.Marshal(getJobReq)
	if err != nil {
		return nil, err
	}

	apiPath := LLAMA_INDEX_API_BASE + fmt.Sprintf("/parsing/job/%s", jobId)

	req, err := http.NewRequest(http.MethodGet, apiPath, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for checking parsing job status: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", viper.Get("document_parser.api_key")))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	parsingStatus := new(LlamaIndexParsingStatusResponse)
	err = json.Unmarshal(respBytes, parsingStatus)
	if err != nil {
		return nil, err
	}

	return parsingStatus, nil
}

func waitJobDoneLlamaIndex(ctx context.Context, jobId string) (LlamaIndexParsingStatus, error) {
	for {
		status, err := checkJobStatusLlamaIndex(jobId)
		if err != nil {
			return "", err
		}

		if status.Status == STATUS_SUCCESS {
			return STATUS_SUCCESS, nil
		}

		if deadline, ok := ctx.Deadline(); ok && time.Now().After(deadline) {
			return status.Status, context.DeadlineExceeded
		}

		time.Sleep(3 * time.Second)
	}
}

type JobMarkdownResultLlamaIndex struct {
	Markdown    string `json:"markdown"`
	JobMetadata struct {
		CreditsUsed               int  `json:"credits_used"`
		JobCreditsUsage           int  `json:"job_credits_usage"`
		JobPages                  int  `json:"job_pages"`
		JobAutoModeTriggeredPages int  `json:"job_auto_mode_triggered_pages"`
		JobIsCacheHit             bool `json:"job_is_cache_hit"`
	} `json:"job_metadata"`
}

func getJobMarkdownResultLlamaIndex(jobId string) (*JobMarkdownResultLlamaIndex, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	getJobReq := LlamaIndexJobStatusRequest{
		JobID: jobId,
	}

	jsonBytes, err := json.Marshal(getJobReq)
	if err != nil {
		return nil, err
	}

	apiPath := LLAMA_INDEX_API_BASE + fmt.Sprintf("/parsing/job/%s/result/markdown", jobId)

	req, err := http.NewRequest(http.MethodGet, apiPath, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for getting parsing job result: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", viper.Get("document_parser.api_key")))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	jobResult := new(JobMarkdownResultLlamaIndex)
	err = json.Unmarshal(respBytes, jobResult)
	if err != nil {
		return nil, err
	}

	return jobResult, nil
}

func saveJobResultLlamaIndex(ctx context.Context, jobId string, indexId int, documentId int) error {
	_, err := getJobMarkdownResultLlamaIndex(jobId)
	if err != nil {
		return err
	}

	queries := sqlc.New(db.MainDB)
	queries.CreateChunk(ctx, sqlc.CreateChunkParams{})

	return nil
}
