package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var anydeskBinaryName = "anydesk"
var anydeskBinaryDirectoryPaths = []string{}

var ErrServiceNotRunning = errors.New("service not running")

func convertError(exitCode int) error {
	// See https://support.anydesk.com/Exit_Codes
	switch exitCode {
	case 1000:
		return errors.New("AnyDesk could not run at all because ntdll was not found.")
	case 1001:
		return errors.New("AnyDesk could not run because kernel32 was not found.")
	case 7000:
		return errors.New("Path initialization failed. See windows event log for details.")
	case 7001:
		return errors.New("Signature check failed.")
	case 7002:
		return errors.New("Unrecognized command line parameter.")
	case 7003:
		return errors.New("Could not start process (already started).")
	case 8000:
		return errors.New("The requested operation requires elevation (start as admin).")
	case 9000:
		return errors.New("Generic exception occured in application. See trace.")
	case 9001:
		return errors.New("The process terminated itself because of a severe error condition. See trace.")
	case 9002:
		return errors.New("The process encountered a system exception. Please contact support.")
	case 9004:
		return errors.New("Error while writing the requested information to stdout.")
	case 9005:
		return errors.New("Error while reading required information from stdin.")
	case 9006:
		return errors.New("The password to be set is too short.")
	case 9007:
		return errors.New("Error while registering licence. See trace for for information.")
	case 9010:
		return errors.New("Could not perform the requested operation because the AnyDesk service was not running.")
	case 0xad1000:
		return errors.New("Could not remove an older client's executable.")
	case 0xad1001:
		return errors.New("Could not stop an older client's service.")
	case 0xad1002:
		return errors.New("Could not terminate an older client's processes.")
	case 0xad1003:
		return errors.New("Could not install the service. May happen in case a Windows Control Panel is open.")
	case 0xad1004:
		return errors.New("An unexpected error occurred.")
	case 0xad1005:
		return errors.New("Received invalid installation parameters.")
	case 0xad1006:
		return errors.New("Could not install custom client (installation set to disallowed). ")
	}
	return nil
}

func convertInfoError(output io.Reader) error {
	// This method is meant for --get-alias, --get-id, --get-status and --version.
	// Iff service isn't running, SERVICE_NOT_RUNNING is output along with non-zero status code.
	bytes, err := ioutil.ReadAll(output)
	if err == nil {
		if strings.TrimSpace(string(bytes)) == "SERVICE_NOT_RUNNING" {
			return ErrServiceNotRunning
		}
	}
	return nil
}

type AnydeskConnectOptions struct {
	Password     string
	Fullscreen   bool
	FileTransfer bool
	Plain        bool
}

func (opts *AnydeskConnectOptions) applyToCmd(cmd *exec.Cmd) {
	if cmd.Args == nil {
		cmd.Args = []string{}
	}
	if len(opts.Password) > 0 {
		cmd.Args = append(cmd.Args, "--with-password")
		cmd.Stdin = strings.NewReader(opts.Password)
	}
	if opts.Fullscreen {
		cmd.Args = append(cmd.Args, "--fullscreen")
	}
	if opts.FileTransfer {
		cmd.Args = append(cmd.Args, "--file-transfer")
	}
	if opts.Plain {
		cmd.Args = append(cmd.Args, "--plain")
	}
}

func findAnyDesk() (result string, err error) {
	result = anydeskBinaryName
	for _, dir := range append(
		filepath.SplitList(os.Getenv("PATH")),
		anydeskBinaryDirectoryPaths...,
	) {
		if dir == "" {
			// Unix shell semantics: path element "" means "."
			dir = "."
		}
		path := filepath.Join(dir, anydeskBinaryName)
		if path, err = exec.LookPath(path); err == nil {
			result = path
			return
		}
	}
	err = &exec.Error{
		Name: anydeskBinaryName,
		Err:  exec.ErrNotFound,
	}
	return
}

type AnydeskCommandLineInterface struct {
	path string
}

func NewAnydeskCommandLineInterface(path string) (cli *AnydeskCommandLineInterface) {
	cli = new(AnydeskCommandLineInterface)
	cli.path = path
	return
}

func (cli *AnydeskCommandLineInterface) run(args ...string) (r *bytes.Buffer, err error) {
	stdoutBuf := new(bytes.Buffer)
	r = stdoutBuf

	cmd, err := cli.cmd(args...)
	if err != nil {
		return
	}

	cmd.Stdin = nil
	cmd.Stdout = stdoutBuf

	err = cmd.Run()
	if err != nil {
		if adErr := convertError(cmd.ProcessState.ExitCode()); adErr != nil {
			err = adErr
		}
		return
	}

	return
}

func (cli *AnydeskCommandLineInterface) cmd(args ...string) (cmd *exec.Cmd, err error) {
	var adPath string
	if len(cli.path) > 0 {
		adPath = cli.path
	} else {
		adPath, err = findAnyDesk()
		if err != nil {
			return
		}
	}

	cmd = exec.Command(adPath, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = nil
	cmd.Stdin = nil

	makeProcessInvisible(cmd)

	return
}

func (cli *AnydeskCommandLineInterface) SetPassword(pw string) (err error) {
	cmd, err := cli.cmd("--set-password")
	if err != nil {
		return
	}

	cmd.Stdin = strings.NewReader(pw)

	err = cmd.Run()
	if err != nil {
		if adErr := convertError(cmd.ProcessState.ExitCode()); adErr != nil {
			err = adErr
		}
		return
	}

	return
}

func (cli *AnydeskCommandLineInterface) Connect(ref string, opt *AnydeskConnectOptions) {
	cmd, err := cli.cmd(ref)
	if err != nil {
		return
	}

	if opt != nil {
		opt.applyToCmd(cmd)
	}

	err = cmd.Run()
	if err != nil {
		if adErr := convertError(cmd.ProcessState.ExitCode()); adErr != nil {
			err = adErr
		}
		return
	}

	return
}

func (cli *AnydeskCommandLineInterface) RemovePassword() (err error) {
	cmd, err := cli.cmd("--remove-password")
	if err != nil {
		return
	}
	err = cmd.Run()
	if err != nil {
		return
	}
	return
}

func (cli *AnydeskCommandLineInterface) GetAlias() (alias string, err error) {
	output, err := cli.run("--get-alias")
	if err != nil {
		if adErr := convertInfoError(output); adErr != nil {
			err = adErr
		}
		return
	}
	alias = strings.TrimSpace(output.String())
	return
}

func (cli *AnydeskCommandLineInterface) GetID() (id string, err error) {
	output, err := cli.run("--get-id")
	if err != nil {
		if adErr := convertInfoError(output); adErr != nil {
			err = adErr
		}
		return
	}
	id = strings.TrimSpace(output.String())
	return
}

func (cli *AnydeskCommandLineInterface) GetStatus() (status string, err error) {
	output, err := cli.run("--get-status")
	if err != nil {
		if adErr := convertInfoError(output); adErr != nil {
			err = adErr
		}
		return
	}
	status = strings.TrimSpace(output.String())
	return
}

func (cli *AnydeskCommandLineInterface) Version() (version string, err error) {
	output, err := cli.run("--version")
	if err != nil {
		if adErr := convertInfoError(output); adErr != nil {
			err = adErr
		}
		return
	}
	version = strings.TrimSpace(output.String())
	return
}
