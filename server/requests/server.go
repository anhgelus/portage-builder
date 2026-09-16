package requests

import (
	"context"
	"crypto/sha3"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"anhgelus.world/portage-builder/proto"
	"anhgelus.world/portage-builder/server/files"
)

type Session struct {
	mu          sync.Mutex
	user        *x509.Certificate
	fileArg     *proto.UploadFileArg
	lastPart    *uint8
	currentFile []byte
}

func (s *Session) HandleFile(_ context.Context, arg *proto.UploadFileArg) (*proto.MessageResponse[proto.NothingArg], error) {
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

func (s *Session) HandleFilePart(_ context.Context, arg *proto.UploadFilePartArg) (*proto.MessageResponse[proto.NothingArg], error) {
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

func (s *Session) HandleAddPackages(ctx context.Context, arg *proto.ListPackage, chroot *files.Root) (*proto.MessageResponse[proto.NothingArg], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := chroot.AppendPackages(ctx, arg.Packages...)
	if err != nil {
		return nil, err
	}
	return proto.OkResponse, nil
}

func (s *Session) HandleRemovePackages(ctx context.Context, arg *proto.ListPackage, chroot *files.Root) (*proto.MessageResponse[proto.NothingArg], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := chroot.RemovePackages(ctx, arg.Packages...)
	if err != nil {
		return nil, err
	}
	return proto.OkResponse, nil
}

func (s *Session) HandleBuildPackages(ctx context.Context, arg *proto.ListPackage, chroot *files.Root) (*proto.MessageResponse[proto.NothingArg], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := chroot.BuildPackages(ctx, false, arg.Packages)
	if err != nil {
		return nil, err
	}
	return proto.OkResponse, nil
}

func (s *Session) HandleUpdateWorld(ctx context.Context, chroot *files.Root) (*proto.MessageResponse[proto.NothingArg], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := chroot.Update(ctx)
	if err != nil {
		return nil, err
	}
	return proto.OkResponse, nil
}

func (s *Session) HandleListPackages(ctx context.Context, chroot *files.Root) (*proto.MessageResponse[*proto.ListPackage], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := chroot.OpenFile("etc/portage/world", os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	pkgs := strings.Split(string(b), "\n")
	arg := &proto.ListPackage{Packages: make([]*proto.Package, 0, len(pkgs))}
	for _, pkg := range pkgs {
		arg.Packages = append(arg.Packages, proto.NewPackage(pkg))
	}
	return &proto.MessageResponse[*proto.ListPackage]{Kind: proto.KindOk, Arg: arg}, nil
}
