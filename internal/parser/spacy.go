package parser

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/OptimusePrime/petagpt/configs"
	"github.com/spf13/viper"
)

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
}

func NewDocumentChunker(ctx context.Context, numWorkers int) (*DocumentChunker, error) {
	chunker := new(DocumentChunker)

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

func (dc *DocumentChunker) SentenceSegmentText(ctx context.Context, text string) ([]string, error) {
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
