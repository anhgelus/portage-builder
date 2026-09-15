package requests

import (
	"crypto/sha3"
	"errors"
	"fmt"
	"sync"

	"anhgelus.world/portage-builder/proto"
)

type Session struct {
	mu          sync.Mutex
	fileArg     *proto.UploadFileArg
	lastPart    *uint8
	currentFile []byte
}

func (s *Session) HandleFile(arg *proto.UploadFileArg) (*proto.MessageResponse[proto.NothingArg], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fileArg != nil {
		return nil, errors.New("file upload not finished")
	}
	s.fileArg = arg
	s.currentFile = make([]byte, 0, arg.Size)
	s.lastPart = nil
	return proto.OkResponse, nil
}

func (s *Session) HandleFilePart(arg *proto.UploadFilePartArg) (*proto.MessageResponse[proto.NothingArg], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fileArg == nil {
		return nil, errors.New("file upload not started")
	}
	if s.lastPart == nil {
		if arg.Part != 0 {
			return nil, errors.New("missing first part")
		}
	} else if *s.lastPart+1 != arg.Part {
		return nil, fmt.Errorf("missing part %d", *s.lastPart+1)
	}
	s.lastPart = &arg.Part
	if len(s.currentFile)+len(arg.Content) > int(s.fileArg.Size) {
		return nil, errors.New("invalid file size: not matching sum of parts")
	}
	s.currentFile = append(s.currentFile, arg.Content...)
	if arg.Part == s.fileArg.Parts-1 {
		if sha3.Sum256(s.currentFile) != s.fileArg.Checksum {
			return nil, errors.New("mismatching checksum")
		}
		//TODO: write file
	}
	return proto.OkResponse, nil
}
