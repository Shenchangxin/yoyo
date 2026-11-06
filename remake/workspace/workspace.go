// Copyright 2026 Hypit.AI
// Translated from @hypit/workspace/src/index.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

// Package workspace is the replaceable definition environment for one compilation.
package workspace

import (
	"io"

	"github.com/Shenchangxin/yoyo/remake/protocol"
	"github.com/Shenchangxin/yoyo/remake/source"
)

// BlobAttachment is a repeatably openable transfer admitted by the selected Workspace.
type BlobAttachment struct {
	Artifact protocol.BlobRef
	// Location is the stable external address when the Workspace supplies a file by reference.
	// A nil Location means the property is absent.
	Location *string
	Open     func() (io.ReadCloser, error)
}

// WorkspaceSession is one isolated, read-once definition session for exactly one compilation.
type WorkspaceSession interface {
	Entry() source.ResolvedSource
	ResolveSource(importer source.SourceUnit, request source.SourceImportRequest) (source.ResolvedSource, error)
	ResolveAsset(importer source.SourceUnit, request source.SourceAssetRequest) (source.ResolvedSourceAsset, error)
	Attachments() []BlobAttachment
}

// Workspace is a replaceable definition environment.
type Workspace interface {
	Open(entryLocator string) (WorkspaceSession, error)
}

// WorkspaceError is the coded failure raised by a Workspace.
type WorkspaceError struct {
	Code    string
	Message string
	Subject *string
	Name    string
}

func (e *WorkspaceError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func workspaceError(code, message, subject string) *WorkspaceError {
	value := subject
	return &WorkspaceError{Code: code, Message: message, Subject: &value, Name: "WorkspaceError"}
}
