package requests

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"anhgelus.world/portage-builder/common"
	"anhgelus.world/portage-builder/proto"
)

type state struct {
	mu       sync.RWMutex
	inConfig bool
	nextFile uint8
	files    uint8
	inPart   bool
	nextPart uint
	parts    uint
}

func (s *state) CanBuild() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.inConfig
}

func (s *state) CanConfig() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.inConfig
}

func (s *state) DoConfig(n uint8) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inConfig = true
	s.inPart = false
	s.nextPart = 0
	s.parts = 0
	s.nextFile = 0
	s.files = n
}

func (s *state) CanSend() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inConfig && !s.inPart
}

func (s *state) DoSend(p uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inPart = true
	s.parts = p
	s.nextPart = 0
}

func (s *state) CanPart(n uint) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextPart == n && s.inPart
}

func (s *state) DoPart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextPart++
	if s.nextPart == s.parts {
		s.inPart = false
		s.nextFile++
	}
	if s.files == s.nextFile {
		s.inConfig = false
	}
}

type File struct {
	Path    string
	Content []byte
}

type UserHandler struct {
	mu    sync.Mutex
	state state
	User  string
	// build requests
	builds chan []*proto.Package
	// config, send and part requests
	updatedFiles    chan File
	currentUpload   bytes.Buffer
	currentChecksum [64]byte
	currentPath     string
}

func NewUserHandler(user string) *UserHandler {
	return &UserHandler{
		User:         user,
		builds:       make(chan []*proto.Package, 1),
		updatedFiles: make(chan File, 1),
	}
}

func (h *UserHandler) HandleBuildRequest(ctx context.Context, arg proto.BuildArg) proto.Response {
	if !h.state.CanBuild() {
		return proto.NewErrorResponse(
			"invalid request",
			fmt.Errorf("cannot use build request in this state"))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.builds <- arg.Packages
	return proto.NewOKResponse()
}

func (h *UserHandler) HandleConfigRequest(ctx context.Context, arg proto.CfgArg) proto.Response {
	if !h.state.CanConfig() {
		return proto.NewErrorResponse(
			"invalid request",
			fmt.Errorf("cannot use config request in this state"))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.state.DoConfig(arg.Files)
	return proto.NewOKResponse()
}

func (h *UserHandler) HandleSendRequest(ctx context.Context, arg proto.SendArg) proto.Response {
	if !h.state.CanSend() {
		return proto.NewErrorResponse(
			"invalid request",
			fmt.Errorf("cannot use send request in this state"))
	}
	if arg.Parts == 0 {
		return proto.NewErrorResponse(
			"invalid request",
			fmt.Errorf("cannot send no parts"))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.state.DoSend(arg.Parts)
	// prepare uploading
	h.currentPath = arg.Path
	h.currentChecksum = arg.Checksum
	return proto.NewOKResponse()
}

func (h *UserHandler) HandlePartRequest(ctx context.Context, arg proto.PartArg) proto.Response {
	if !h.state.CanPart(arg.Part) {
		return proto.NewErrorResponse(
			"invalid request",
			fmt.Errorf("cannot use part request for %d in this state", arg.Part))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.state.DoPart()
	h.currentUpload.Write(arg.Content)
	// finish uploading
	if h.state.nextPart == h.state.parts {
		b := h.currentUpload.Bytes()
		sum := common.ChecksumOfBytes(b)
		if sum != h.currentChecksum {
			return proto.NewErrorResponse(
				"cannot verify file",
				fmt.Errorf(
					"checksum doesn't match: %x (computed), %x (expected)",
					sum, h.currentChecksum))
		}
		h.updatedFiles <- File{h.currentPath, b}
		h.currentUpload = bytes.Buffer{}
	}
	return proto.NewOKResponse()
}

func (h *UserHandler) UpdatedBuilds() <-chan []*proto.Package {
	return h.builds
}

func (h *UserHandler) UploadedFiles() <-chan File {
	return h.updatedFiles
}

func (h *UserHandler) Close() {
	close(h.builds)
	close(h.updatedFiles)
}
