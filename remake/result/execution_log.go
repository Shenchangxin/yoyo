// Copyright 2026 Hypit.AI
// Translated from @hypit/result/src/execution-log.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package result

import "io"

type countingReader struct {
	reader io.Reader
	size   *int64
}

func (c countingReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	*c.size += int64(n)
	return n, err
}

// PreserveExecutionLog stores execution evidence as execution.jsonl.
// A nil stream records nothing.
func PreserveExecutionLog(chunks io.Reader, write func(path string, chunks io.Reader, mediaType string) error) (*BuildResultFileRef, error) {
	if chunks == nil {
		return nil, nil
	}
	var size int64
	const path = "execution.jsonl"
	const mediaType = "application/x-ndjson"
	if err := write(path, countingReader{reader: chunks, size: &size}, mediaType); err != nil {
		return nil, err
	}
	return &BuildResultFileRef{Kind: "build-file", Path: path, Size: size, MediaType: mediaType}, nil
}
