package utils

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/dropbox/godropbox/errors"
	"github.com/ozean12/pritunl-zero/errortypes"
	"github.com/sirupsen/logrus"
)

func Exec(dir, name string, arg ...string) (err error) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if dir != "" {
		cmd.Dir = dir
	}

	err = cmd.Run()
	if err != nil {
		err = &errortypes.ExecError{
			errors.Wrapf(err, "utils: Failed to exec '%s'", name),
		}
		return
	}

	return
}

func ExecInput(dir, input, name string, arg ...string) (err error) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		err = &errortypes.ExecError{
			errors.Wrapf(err,
				"utils: Failed to get stdin in exec '%s'", name),
		}
		return
	}

	if dir != "" {
		cmd.Dir = dir
	}

	err = cmd.Start()
	if err != nil {
		err = &errortypes.ExecError{
			errors.Wrapf(err, "utils: Failed to exec '%s'", name),
		}
		return
	}

	_, err = io.WriteString(stdin, input)
	if err != nil {
		err = &errortypes.ExecError{
			errors.Wrapf(err, "utils: Failed to write stdin in exec '%s'",
				name),
		}
		return
	}

	err = stdin.Close()
	if err != nil {
		err = &errortypes.ExecError{
			errors.Wrapf(err, "utils: Failed to close stdin in exec '%s'",
				name),
		}
		return
	}

	err = cmd.Wait()
	if err != nil {
		err = &errortypes.ExecError{
			errors.Wrapf(err, "utils: Failed to exec '%s'", name),
		}
		return
	}

	return
}

func ExecOutput(dir, name string, arg ...string) (output string, err error) {
	cmd := exec.Command(name, arg...)
	cmd.Stderr = os.Stderr

	if dir != "" {
		cmd.Dir = dir
	}

	outputByt, err := cmd.Output()
	if outputByt != nil {
		output = string(outputByt)
	}
	if err != nil {
		err = &errortypes.ExecError{
			errors.Wrapf(err, "utils: Failed to exec '%s'", name),
		}
		return
	}

	return
}

func ExecCombinedOutput(dir, name string, arg ...string) (
	output string, err error) {

	cmd := exec.Command(name, arg...)

	if dir != "" {
		cmd.Dir = dir
	}

	outputByt, err := cmd.CombinedOutput()
	if outputByt != nil {
		output = string(outputByt)
	}
	if err != nil {
		err = &errortypes.ExecError{
			errors.Wrapf(err, "utils: Failed to exec '%s'", name),
		}
		return
	}

	return
}

func ExecCombinedOutputLogged(ignores []string, name string, arg ...string) (
	output string, err error) {

	cmd := exec.Command(name, arg...)

	outputByt, err := cmd.CombinedOutput()
	if outputByt != nil {
		output = string(outputByt)
	}

	if err != nil && ignores != nil {
		for _, ignore := range ignores {
			if strings.Contains(output, ignore) {
				err = nil
				break
			}
		}
	}
	if err != nil {
		err = &errortypes.ExecError{
			errors.Wrapf(err, "utils: Failed to exec '%s'", name),
		}

		logrus.WithFields(logrus.Fields{
			"output": output,
			"cmd":    name,
			"arg":    arg,
			"error":  err,
		}).Error("utils: Process exec error")
		return
	}

	return
}

func ExecCombinedOutputLoggedDir(ignores []string,
	dir, name string, arg ...string) (
	output string, err error) {

	cmd := exec.Command(name, arg...)
	if dir != "" {
		cmd.Dir = dir
	}

	outputByt, err := cmd.CombinedOutput()
	if outputByt != nil {
		output = string(outputByt)
	}

	if err != nil && ignores != nil {
		for _, ignore := range ignores {
			if strings.Contains(output, ignore) {
				err = nil
				break
			}
		}
	}
	if err != nil {
		err = &errortypes.ExecError{
			errors.Wrapf(err, "utils: Failed to exec '%s'", name),
		}

		logrus.WithFields(logrus.Fields{
			"output": output,
			"cmd":    name,
			"arg":    arg,
			"error":  err,
		}).Error("utils: Process exec error")
		return
	}

	return
}

func ExecOutputLogged(ignores []string, name string, arg ...string) (
	output string, err error) {

	cmd := exec.Command(name, arg...)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Run()
	output = stdout.String()
	errOutput := stderr.String()

	if err != nil && ignores != nil {
		for _, ignore := range ignores {
			if strings.Contains(output, ignore) ||
				strings.Contains(errOutput, ignore) {

				err = nil
				break
			}
		}
	}
	if err != nil {
		err = &errortypes.ExecError{
			errors.Wrapf(err, "utils: Failed to exec '%s'", name),
		}

		logrus.WithFields(logrus.Fields{
			"output":       output,
			"error_output": errOutput,
			"cmd":          name,
			"arg":          arg,
			"error":        err,
		}).Error("utils: Process exec error")
		return
	}

	return
}
