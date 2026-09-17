package files

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path"

	"anhgelus.world/portage-builder/common"
	"anhgelus.world/portage-builder/proto"
)

// Root represents a Gentoo chroot.
type Root struct {
	*os.Root
	User string
	info *common.RingBuffer
}

// LoadRoot for the specified user.
func LoadRoot(userFolder, user string) (*Root, error) {
	root, err := os.OpenRoot(path.Join(userFolder, user))
	return &Root{root, user, common.NewRingBuffer(512)}, err
}

// CreateRoot initializes the [Root] for the given user.
//
// stage3 is the path to the xz-compressed tarball containing the Gentoo stage3 to use.
func CreateRoot(ctx context.Context, stage3, userFolder, user string) (*Root, error) {
	p := path.Join(userFolder, user)
	_, err := os.Stat(p)
	if err == nil {
		return LoadRoot(userFolder, user)
	}
	err = os.MkdirAll(userFolder, 0o755)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(
		ctx,
		"tar",
		"xpvf", stage3, "--xattrs-include='*.*'", "--numeric-owner", p)
	err = cmd.Run()
	if err != nil {
		return nil, err
	}
	return LoadRoot(userFolder, user)
}

// Mount required folders in the [Root].
func (r *Root) Mount(ctx context.Context) error {
	mnt := exec.CommandContext(ctx, "mount", "--types", "proc", "/proc", r.Path("proc"))
	err := mnt.Run()
	if err != nil {
		return err
	}
	mnt = exec.CommandContext(ctx, "mount", "--rbind", "/dev", r.Path("dev"))
	err = mnt.Run()
	if err != nil {
		return err
	}
	mnt = exec.CommandContext(ctx, "cp", "--dereference", "/etc/resolv.conf", r.Path("etc"))
	return mnt.Run()
}

// Path returns the absolute path of the path inside the [Root].
func (r *Root) Path(inside string) (absolute string) {
	return path.Join(r.Name(), inside)
}

func (r *Root) Close(ctx context.Context) error {
	errs := make([]error, 0, 4)
	mnt := exec.CommandContext(ctx, "umount", r.Path("proc"))
	errs = append(errs, mnt.Run())
	mnt = exec.CommandContext(ctx, "umount", r.Path("dev"))
	errs = append(errs, mnt.Run())
	mnt = exec.CommandContext(ctx, "rm", r.Path("etc/resolv.conf"))
	errs = append(errs, mnt.Run())
	errs = append(errs, r.Root.Close())
	var out error
	for _, err := range errs {
		if err == nil {
			continue
		}
		if out != nil {
			out = errors.Join(out, err)
		} else {
			out = err
		}
	}
	return out
}

// BuildPackages in the [Root].
func (r *Root) BuildPackages(ctx context.Context, emptytree bool, pkgs []*proto.Package) error {
	args := make([]string, 0, 2+len(pkgs))
	// verbose, build binpkg
	args = append(args, "-vb")
	if emptytree {
		args = append(args, "--emptytree")
	}
	for _, pkg := range pkgs {
		args = append(args, pkg.String())
	}
	chroot := r.CommandContext(ctx, "emerge", args...)
	return chroot.Run()
}

// Update the [Root] by upgrading the @world.
func (r *Root) Update(ctx context.Context) error {
	cmd := r.CommandContext(ctx, "emaint", "-a", "sync")
	err := cmd.Run()
	if err != nil {
		return err
	}
	cmd = r.CommandContext(ctx, "emerge", "-1", "portage")
	err = cmd.Run()
	if err != nil {
		return err
	}
	cmd = r.CommandContext(ctx, "emerge", "-vuDN", "@world")
	err = cmd.Run()
	if err != nil {
		return err
	}
	cmd = r.CommandContext(ctx, "emerge", "--depclean")
	return cmd.Run()
}

// CommandContext returns a [exec.Cmd] that runs inside the [Root].
func (r *Root) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	arg := make([]string, 0, 2+len(args))
	arg = append(arg, r.Name(), name)
	arg = append(arg, args...)
	cmd := exec.CommandContext(ctx, "chroot", arg...)
	cmd.Stderr = r.info
	cmd.Stdout = r.info
	return cmd
}

// ChecksumOf a file in the [Root].
func (r *Root) ChecksumOf(path string) ([32]byte, error) {
	return common.ChecksumOf(r.FS(), path)
}

// RemovePackages from the world file (@selected set).
func (r *Root) RemovePackages(ctx context.Context, pkgs ...*proto.Package) error {
	args := make([]string, 0, len(pkgs)+1)
	args = append(args, "-W")
	for _, pkg := range pkgs {
		args = append(args, pkg.String())
	}
	return r.CommandContext(ctx, "emerge", args...).Run()
}

// AppendPackages to the world file (@selected set).
func (r *Root) AppendPackages(ctx context.Context, pkgs ...*proto.Package) error {
	args := make([]string, 0, len(pkgs)+1)
	args = append(args, "-n")
	for _, pkg := range pkgs {
		args = append(args, pkg.String())
	}
	return r.CommandContext(ctx, "emerge", args...).Run()
}

func (r *Root) Info() io.Reader {
	return r.info
}
