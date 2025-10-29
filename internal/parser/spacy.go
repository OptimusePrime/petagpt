package parser

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/OptimusePrime/petagpt/configs"
	"github.com/spf13/viper"
	"golang.org/x/sync/semaphore"
)

type Chunk struct {
	Content string
	Context string
}

func (c Chunk) String() string {
	return fmt.Sprintf("%s\n\n\n%s", c.Context, c.Content)
}

func (c Chunk) SHA256() string {
	sha := sha256.Sum256([]byte(c.String()))
	
	return base64.StdEncoding.EncodeToString(sha[:])
}

type spacyMethod string

const (
	SENTENCE_SEGMENTATION spacyMethod = "senter"
)

type spacyRequest struct {
	ID     string          `json:"id"`
	Method spacyMethod     `json:"method"`
	Data   json.RawMessage `json:"data"`
}

type spacyResponse struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type SpacyWorker struct {
	cmd      *exec.Cmd
	writer   *bufio.Writer
	reader   *bufio.Scanner
	mu       sync.Mutex
	inflight sync.Map
}

type spacySegmentationRequest struct {
	Text string `json:"text"`
}

type spacySegmentationResponse struct {
	Sentences []string `json:"sentences"`
}

func StartSpacyWorker(ctx context.Context, pythonPath string) (*SpacyWorker, error) {
	scriptPath := filepath.Join(viper.GetString("data_dir"), "bin/spacy_worker.py")
	cmd := exec.CommandContext(ctx, pythonPath, scriptPath)

	cmd.Stderr = cmd.Stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	worker := &SpacyWorker{
		cmd:    cmd,
		writer: bufio.NewWriter(stdin),
		reader: bufio.NewScanner(stdout),
	}

	go func() {
		for worker.reader.Scan() {
			var msg spacyResponse

			if err := json.Unmarshal(worker.reader.Bytes(), &msg); err != nil {
				continue
			}

			if chAny, ok := worker.inflight.Load(msg.ID); ok {
				ch := chAny.(chan spacyResponse)
				ch <- msg
				close(ch)
				worker.inflight.Delete(msg.ID)
			}
		}
	}()

	return worker, nil
}

func (w *SpacyWorker) Call(ctx context.Context, method spacyMethod, payload any, out any) error {
	id := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int())
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg, err := json.Marshal(spacyRequest{
		ID:     id,
		Method: method,
		Data:   jsonPayload,
	})
	fmt.Println("marshalled json")
	if err != nil {
		return err
	}

	ch := make(chan spacyResponse, 1)
	w.inflight.Store(id, ch)

	w.mu.Lock()
	_, err = w.writer.Write(append(msg, '\n'))
	if err == nil {
		err = w.writer.Flush()
	}
	w.mu.Unlock()
	fmt.Println("wrote data to worker")

	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	select {
	case m := <-ch:
		if m.Error != "" {
			return errors.New(m.Error)
		}
		if out != nil {
			return json.Unmarshal(m.Result, out)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *SpacyWorker) Shutdown() error {
	return w.cmd.Process.Kill()
}

type DocumentChunker struct {
	workers []*SpacyWorker
	sem     *semaphore.Weighted
}

func NewDocumentChunker(ctx context.Context, numWorkers int) (*DocumentChunker, error) {
	chunker := new(DocumentChunker)
	chunker.sem = semaphore.NewWeighted(int64(numWorkers))

	for range numWorkers {
		w, err := StartSpacyWorker(ctx, configs.GetPythonPath())
		if err != nil {
			return nil, err
		}

		chunker.workers = append(chunker.workers, w)
	}

	return chunker, nil
}

func (dc *DocumentChunker) Shutdown() error {
	for _, w := range dc.workers {
		return w.Shutdown()
	}

	return nil
}

func (dc *DocumentChunker) NumWorkers() int {
	return len(dc.workers)
}

func (dc *DocumentChunker) sentenceSegmentText(ctx context.Context, text string) ([]string, error) {
	workerIdx := rand.IntN(dc.NumWorkers())
	fmt.Println("randomized")

	resp := new(spacySegmentationResponse)

	err := dc.workers[workerIdx].Call(ctx, SENTENCE_SEGMENTATION, spacySegmentationRequest{
		Text: text,
	}, resp)
	fmt.Println("called worker")
	if err != nil {
		return []string{}, err
	}

	return resp.Sentences, nil
}

func (dc *DocumentChunker) extractTablesFromDocument(ctx context.Context, parsedDocument string) ([]string, error) {
	tableRegexStr := "<table>.*?<\\/table>"
	tableRegex, err := regexp.Compile(tableRegexStr)
	if err != nil {
		return nil, err
	}

	tables := tableRegex.FindAllString(parsedDocument, -1)

	ch := make(chan string, len(tables))
	errCh := make(chan error, len(tables))

	for _, table := range tables {
		err = dc.sem.Acquire(ctx, 1)
		if err != nil {
			return nil, err
		}

		go func() {
			defer dc.sem.Release(1)

			tableSummary, err := TransformTable(ctx, table)
			if err != nil {
				errCh <- err
			}

			ch <- tableSummary
		}()
	}

	var tableSummaries []string

	for range len(tables) {
		select {
		case summary := <-ch:
			tableSummaries = append(tableSummaries, summary)
		case err := <-errCh:
			return nil, err
		}
	}

	return tableSummaries, nil
}

func (dc *DocumentChunker) Chunk(ctx context.Context, document string, chunkSize int) ([]Chunk, error) {
	var chunkContents []string

	tableSummaries, err := dc.extractTablesFromDocument(ctx, document)
	if err != nil {
		return nil, err
	}

	chunkContents = append(chunkContents, tableSummaries...)

	tableRegexStr := "<table>.*?<\\/table>"
	tableRegex, err := regexp.Compile(tableRegexStr)
	if err != nil {
		return nil, err
	}

	documentNoTables := tableRegex.ReplaceAllString(document, "")

	pages := strings.Split(documentNoTables, PARSING_PAGE_SEPARATOR)

	var sentences []string

	for _, page := range pages {
		pageSentences, err := dc.sentenceSegmentText(ctx, page)
		if err != nil {
			return nil, fmt.Errorf("sentence segmentation failed: %w", err)
		}

		sentences = append(sentences, pageSentences...)
	}

	currentSentence := 0
	for {
		cutoff := min(currentSentence+chunkSize, len(sentences))

		chunk := strings.TrimSpace(strings.Join(sentences[currentSentence:cutoff], " "))
		if len(chunk) > 0 {
			chunkContents = append(chunkContents, chunk)
		}

		if cutoff == len(sentences) {
			break
		}

		currentSentence += chunkSize
	}

	ch := make(chan Chunk, len(chunkContents))
	errCh := make(chan error, len(chunkContents))

	for _, content := range chunkContents {
		if err = dc.sem.Acquire(ctx, 1); err != nil {
			return nil, err
		}

		go func() {
			defer dc.sem.Release(1)

			chunkContext, err := CreateChunkContext(ctx, document, content)
			if err != nil {
				errCh <- err
				return
			}

			chunk := Chunk{
				Content: content,
				Context: chunkContext,
			}

			ch <- chunk
		}()
	}

	var chunks []Chunk

	for range len(chunkContents) {
		select {
		case chunk := <-ch:
			chunks = append(chunks, chunk)
		case err = <-errCh:
			return nil, err
		}
	}

	return chunks, nil
}
