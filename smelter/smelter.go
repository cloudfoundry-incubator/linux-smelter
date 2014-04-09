package smelter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry-incubator/runtime-schema/models"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/cloudfoundry/gunk/command_runner"

	"github.com/cloudfoundry-incubator/linux-smelter/droplet"
)

type Smelter struct {
	config *models.LinuxSmeltingConfig

	runner command_runner.CommandRunner
}

type NoneDetectedError struct {
	AppDir string
}

func (e NoneDetectedError) Error() string {
	return fmt.Sprintf("no buildpack detected for %s", e.AppDir)
}

type MalformedReleaseYAMLError struct {
	ParseError error
}

func (e MalformedReleaseYAMLError) Error() string {
	return fmt.Sprintf(
		"buildpack's release output invalid: %s",
		e.ParseError,
	)
}

type Release struct {
	DefaultProcessTypes struct {
		Web string `yaml:"web"`
	} `yaml:"default_process_types"`
}

func New(
	config *models.LinuxSmeltingConfig,
	runner command_runner.CommandRunner,
) *Smelter {
	return &Smelter{
		config: config,
		runner: runner,
	}
}

func (s *Smelter) Smelt() error {
	if err := os.MkdirAll(s.config.OutputDir(), 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(s.config.ResultJsonDir(), 0755); err != nil {
		return err
	}

	detectedBuildpackDir, detectedName, err := s.detect()
	if err != nil {
		return err
	}

	err = s.compile(detectedBuildpackDir)
	if err != nil {
		return err
	}

	releaseInfo, err := s.release(detectedBuildpackDir)
	if err != nil {
		return err
	}

	err = s.saveInfo(detectedName, releaseInfo)
	if err != nil {
		return err
	}

	dropletFS := droplet.NewFileSystem(s.runner)

	err = dropletFS.GenerateFiles(s.config.AppDir(), s.config.OutputDir())
	if err != nil {
		return err
	}

	return nil
}

func (s *Smelter) detect() (string, string, error) {
	for _, buildpack := range s.config.BuildpackOrder() {
		output := new(bytes.Buffer)

		err := s.runner.Run(&exec.Cmd{
			Path:   path.Join(s.config.BuildpackPath(buildpack), "bin", "detect"),
			Args:   []string{s.config.AppDir()},
			Stdout: output,
			Stderr: os.Stderr,
		})

		if err == nil {
			return s.config.BuildpackPath(buildpack), strings.TrimRight(output.String(), "\n"), nil
		}
	}

	return "", "", NoneDetectedError{AppDir: s.config.AppDir()}
}

func (s *Smelter) compile(buildpackDir string) error {
	return s.runner.Run(&exec.Cmd{
		Path:   path.Join(buildpackDir, "bin", "compile"),
		Args:   []string{s.config.AppDir(), s.config.CacheDir()},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}

func (s *Smelter) release(buildpackDir string) (Release, error) {
	releaseOut := new(bytes.Buffer)

	release := &exec.Cmd{
		Path:   path.Join(buildpackDir, "bin", "release"),
		Args:   []string{s.config.AppDir()},
		Stderr: os.Stderr,
		Stdout: releaseOut,
	}

	err := s.runner.Run(release)
	if err != nil {
		return Release{}, err
	}

	decoder := candiedyaml.NewDecoder(releaseOut)

	var parsedRelease Release

	err = decoder.Decode(&parsedRelease)
	if err != nil {
		return Release{}, MalformedReleaseYAMLError{err}
	}

	return parsedRelease, nil
}

func (s *Smelter) saveInfo(detectedName string, releaseInfo Release) error {
	infoFile, err := os.Create(filepath.Join(s.config.OutputDir(), "staging_info.yml"))
	if err != nil {
		return err
	}

	defer infoFile.Close()

	resultFile, err := os.Create(s.config.ResultJsonPath())
	if err != nil {
		return err
	}

	defer resultFile.Close()

	info := models.StagingInfo{
		DetectedBuildpack: detectedName,
		StartCommand:      releaseInfo.DefaultProcessTypes.Web,
	}

	err = candiedyaml.NewEncoder(infoFile).Encode(info)
	if err != nil {
		return err
	}

	err = json.NewEncoder(resultFile).Encode(info)
	if err != nil {
		return err
	}

	return nil
}
