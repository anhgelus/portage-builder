package files

import (
	"bytes"
	"context"
	"errors"
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
}

// LoadRoot for the specified user.
func LoadRoot(userFolder, user string) (*Root, error) {
	root, err := os.OpenRoot(path.Join(userFolder, user))
	return &Root{root, user}, err
}

// CreateRoot initializes the [Root] for the given user.
//
// stage3 is the path to the xz-compressed tarball containing the Gentoo stage3 to use.
func CreateRoot(ctx context.Context, stage3, userFolder, user string) (*Root, error) {
	p := path.Join(userFolder, user)
	err := os.Mkdir(p, 0o755)
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
	var errs []error
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

// BuildWorld (@world set) in the [Root].
func (r *Root) BuildWorld(ctx context.Context, emptytree bool) error {
	return r.BuildSet(ctx, "@world", emptytree)
}

// BuildSelected (@selected set) in the [Root].
func (r *Root) BuildSelected(ctx context.Context, emptytree bool) error {
	return r.BuildSet(ctx, "@selected", emptytree)
}

// BuildSet in the [Root].
func (r *Root) BuildSet(ctx context.Context, set string, emptytree bool) error {
	flag := "-p"
	if emptytree {
		flag = "--emptytree"
	}
	chroot := r.CommandContext(ctx, "emerge", flag, set)
	return chroot.Run()
}

// Update the [Root] by upgrading the @world.
func (r *Root) Update(ctx context.Context) error {
	cmd := r.CommandContext(ctx, "emaint", "-a", "sync")
	err := cmd.Run()
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
	arg := make([]string, 2+len(args))
	arg[0] = r.Name()
	arg[1] = name
	for i, a := range args {
		arg[i+2] = a
	}
	return exec.CommandContext(ctx, "chroot", arg...)
}

// ChecksumOf a file in the [Root].
func (r *Root) ChecksumOf(path string) ([64]byte, error) {
	return common.ChecksumOf(r.FS(), path)
}

// AppendPackage to the world file (@selected set).
func (r *Root) AppendPackage(pkgs ...*proto.Package) error {
	f, err := r.OpenFile(
		"etc/portage/world",
		os.O_APPEND|os.O_WRONLY|os.O_CREATE,
		0o644)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer f.Close()
	var buf bytes.Buffer
	buf.Grow(len(pkgs))
	for _, pkg := range pkgs {
		buf.WriteRune('\n')
		buf.WriteString(string(*pkg))
	}
	_, err = f.Write(buf.Bytes())
	return err
}
