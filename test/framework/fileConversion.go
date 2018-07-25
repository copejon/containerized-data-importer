package framework

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"kubevirt.io/containerized-data-importer/pkg/image"
	"path/filepath"
)

var formatTable = map[string]func(string, string) (string, error){
	image.ExtGz:    gzCmd,
	image.ExtXz:    xzCmd,
	image.ExtTar:   tarCmd,
	image.ExtQcow2: qcow2Cmd,
	"":             noopCmd,
}

// create file based on targetFormat extensions and return created file's name.
// note: intermediate files are removed.
func FormatTestData(srcFile, tgtDir string, targetFormats ...string) (string, error) {
	var err error
	for _, tf := range targetFormats {
		f, ok := formatTable[tf]
		if !ok {
			return "", errors.Errorf("format extension %q not recognized", tf)
		}
		// invoke conversion func
		srcFile, err = f(srcFile, tgtDir)
		if err != nil {
			return "", errors.Wrap(err, "could not format test data")
		}
	}
	return srcFile, nil
}

func tarCmd(src, tgtDir string) (string, error) {
	base := filepath.Base(src)
	tgt := filepath.Join(tgtDir, base+image.ExtTar)
	args := []string{"-cf", tgt, src}

	if err := doCmdAndVerifyFile(tgt, "tar", args...); err != nil {
		return "", err
	}
	return tgt, nil
}

func gzCmd(src, tgtDir string) (string, error) {
	src, err := copyIfNotPresent(src, tgtDir)
	if err != nil {
		return "", err
	}
	base := filepath.Base(src)
	tgt := filepath.Join(tgtDir, base+image.ExtGz)
	if err := doCmdAndVerifyFile(tgt, "gzip", src); err != nil {
		return "", err
	}
	return tgt, nil
}

func xzCmd(src, tgtDir string) (string, error) {
	src, err := copyIfNotPresent(src, tgtDir)
	if err != nil {
		return "", err
	}
	base := filepath.Base(src)
	tgt := filepath.Join(tgtDir, base+image.ExtXz)
	if err := doCmdAndVerifyFile(tgt, "xz", src); err != nil {
		return "", err
	}
	return tgt, nil
}

func qcow2Cmd(srcfile, tgtDir string) (string, error) {
	tgt := strings.Replace(filepath.Base(srcfile), ".iso", image.ExtQcow2, 1)
	tgt = filepath.Join(tgtDir, tgt)
	args := []string{"convert", "-f", "raw", "-O", "qcow2", srcfile, tgt}

	if err := doCmdAndVerifyFile(tgt, "qemu-img", args...); err != nil {
		return "", err
	}
	return tgt, nil
}

func noopCmd(src, tgtDir string) (string, error) {
	newSrc, err := copyIfNotPresent(src, tgtDir)
	if err != nil {
		return "", err
	}
	return newSrc, nil
}

func doCmdAndVerifyFile(tgt, cmd string, args ...string) error {
	if err := doCmd(cmd, args...); err != nil {
		return err
	}
	if _, err := os.Stat(tgt); err != nil {
		return errors.Wrapf(err, "Failed to stat file %q", tgt)
	}
	return nil
}

func doCmd(osCmd string, osArgs ...string) error {
	cmd := exec.Command(osCmd, osArgs...)
	cout, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "OS command `%s %v` errored: %v\nStdout/Stderr: %s", osCmd, strings.Join(osArgs, " "), err, string(cout))
	}
	return nil
}

// copyIfNotPresent checks for the src file in the tgtDir.  If it is not there, it attempts to copy if from src to tgtdir.
// If a copy is performed, the path to the copy is returned.
// If the file already exists, no copy is done and the path to the existing file is returned
func copyIfNotPresent(src, tgtDir string) (string, error) {
	base := filepath.Base(src)
	// Only copy the source image if it does not exist in the temp directory
	if _, err := os.Stat(filepath.Join(tgtDir, base)); err != nil {
		if err := doCmd("cp", "-f", src, tgtDir); err != nil {
			return "", err
		}
	}
	return filepath.Join(tgtDir, base), nil
}
