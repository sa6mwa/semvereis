package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/sa6mwa/blox"
	"github.com/spf13/cobra"
)

//go:embed semvereis.go
var code string

//go:embed VERSION
var version string

var (
	gitDir               = ".git"
	gitCommand           = "git"
	gitRetrieveLatestTag = []string{"describe", "--tags", "--abbrev=0"}
	gitRetrieveFullTag   = []string{"describe", "--tags", "--long"}
	gitLatestShortHash   = []string{"rev-parse", "--short", "HEAD"}
)

var (
	fromFile                           string
	preserveV                          bool
	addV                               bool
	longTags                           bool
	addLatestGitCommitHashAsPreRelease bool
	prerelease                         string
	fallbackSemver                     string
)

var (
	ErrUnableToRetrieveSemverString = errors.New("unable to retrieve semantic version string")
)

var rootCmd = &cobra.Command{
	Use:          "semvereis",
	Short:        "A tool to retrieve and increment semantic versions",
	Version:      version,
	SilenceUsage: true,
}

var nextCmd = &cobra.Command{
	Use:     "next",
	Aliases: []string{"n", "bump", "increment"},
	Short:   "Retrieve the next semantic major, minor or patch version",
}

var nextMajorCmd = &cobra.Command{
	Use:                   "major [flags] [semverString]",
	DisableFlagsInUseLine: true,
	Short:                 "Increment major version",
	Long:                  blox.WrapString("Increment the major version of optional semverString. If semverString is left out and there is a .git directory in current directory, tool will execute git to attempt retrieving the latest tag and try incrementing it in case it passes as a semantic version.", 80),
	RunE:                  nextVersionFunc,
}

var nextMinorCmd = &cobra.Command{
	Use:                   "minor [flags] [semverString]",
	DisableFlagsInUseLine: true,
	Short:                 "Increment minor version",
	Long:                  blox.WrapString("Increment the minor version of optional semverString. If semverString is left out and there is a .git directory in current directory, tool will execute git to attempt retrieving the latest tag and try incrementing it in case it passes as a semantic version.", 80),
	RunE:                  nextVersionFunc,
}

var nextPatchCmd = &cobra.Command{
	Use:                   "patch [flags] [semverString]",
	DisableFlagsInUseLine: true,
	Short:                 "Increment patch version",
	Long:                  blox.WrapString("Increment the patch version of optional semverString. If semverString is left out and there is a .git directory in current directory, tool will execute git to attempt retrieving the latest tag and try incrementing it in case it passes as a semantic version.", 80),
	RunE:                  nextVersionFunc,
}

var dumpCodeCmd = &cobra.Command{
	Use:                   "code",
	DisableFlagsInUseLine: true,
	Short:                 "Dump the code of this program to stdout",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(code)
	},
}

func nextVersionFunc(cmd *cobra.Command, args []string) error {
	var inr io.Reader = nil
	if len(args) > 0 {
		if len(args) > 1 {
			return cmd.Help()
		}
		inr = strings.NewReader(args[0])
	}
	// --file will override argument
	if f, err := cmd.Flags().GetString("file"); err != nil {
		return err
	} else if f == "-" {
		inr = os.Stdin
	} else if f != "" {
		fd, err := os.Open(f)
		if err != nil {
			return err
		}
		defer fd.Close()
		inr = fd
	}
	svstr, err := getSemVer(inr, longTags, fallbackSemver)
	if err != nil {
		return err
	}
	if addV {
		preserveV = true
		if !strings.HasPrefix(svstr, "v") {
			svstr = "v" + svstr
		}
	}
	v, err := semver.NewVersion(svstr)
	if err != nil {
		return err
	}
	var sv semver.Version
	switch cmd.Name() {
	case "major":
		sv = v.IncMajor()
	case "minor":
		sv = v.IncMinor()
	case "patch":
		sv = v.IncPatch()
	}
	pr := strings.TrimSpace(prerelease)
	if addLatestGitCommitHashAsPreRelease {
		gh, err := getLatestGitHash()
		if err != nil {
			return err
		}
		if pr != "" && gh != "" {
			pr += "-" + gh
		} else if gh != "" {
			pr = gh
		}
	}
	if pr != "" {
		sv, err = sv.SetPrerelease(pr)
		if err != nil {
			return err
		}
	}
	if preserveV {
		fmt.Println(sv.Original())
	} else {
		fmt.Println(sv.String())
	}
	return nil
}

func getSemVer(semverInput io.Reader, useLongTag bool, gitFallbackSemver string) (string, error) {
	semverString := ""
	if semverInput == nil {
		// attempt to get semver string from git
		fi, err := os.Stat(gitDir)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			fb := strings.TrimSpace(gitFallbackSemver)
			if fb != "" {
				return fb, nil
			}
			return "", fmt.Errorf("no %s directory found and no default semver specified", gitDir)
		} else if err != nil {
			return "", err
		}
		if !fi.IsDir() {
			return "", nil
		}
		// resolve gitPath
		gitPath := which(gitCommand)
		if gitPath == "" {
			return "", fmt.Errorf("%s directory exists, but could not locate %s command in PATH", gitDir, gitCommand)
		}
		var cmd *exec.Cmd
		if useLongTag {
			cmd = exec.Command(gitPath, gitRetrieveFullTag...)
		} else {
			cmd = exec.Command(gitPath, gitRetrieveLatestTag...)
		}
		var cmdOut bytes.Buffer
		var cmdErr bytes.Buffer
		cmd.Stdout = &cmdOut
		cmd.Stderr = &cmdErr
		if err := cmd.Run(); err != nil {
			fb := strings.TrimSpace(gitFallbackSemver)
			if fb != "" {
				return fb, nil
			}
			fmt.Fprintf(os.Stderr, "%s %s: %v: %s", gitPath, strings.Join(gitRetrieveLatestTag, " "), err, cmdErr.String())
			return "", err
		}
		s := bufio.NewScanner(bytes.NewReader(cmdOut.Bytes()))
		for s.Scan() {
			semverString = strings.TrimSpace(s.Text())
			break
		}
		if err := s.Err(); err != nil {
			return "", err
		}
	} else {
		// use scanner and stop processing after reading the first line
		s := bufio.NewScanner(semverInput)
		for s.Scan() {
			semverString = s.Text()
			break
		}
		if err := s.Err(); err != nil {
			return "", err
		}
	}
	if semverString == "" {
		return "", errors.New("unable to retrieve a semantic version string")
	}
	return semverString, nil
}

func getLatestGitHash() (shortHash string, err error) {
	gitPath := which(gitCommand)
	if gitPath == "" {
		return "", fmt.Errorf("%s command not found in PATH", gitCommand)
	}
	cmd := exec.Command(gitPath, gitLatestShortHash...)
	var cmdOut bytes.Buffer
	var cmdErr bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: %v: %s", gitPath, strings.Join(gitLatestShortHash, " "), err, cmdErr.String())
		return "", err
	}
	s := bufio.NewScanner(bytes.NewReader(cmdOut.Bytes()))
	for s.Scan() {
		shortHash = strings.TrimSpace(s.Text())
		break
	}
	if shortHash != "" {
		return shortHash, nil
	}
	return "", errors.New("unable to get latest git hash")
}

func which(prog string) string {
	for _, p := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		progPath := filepath.Join(p, prog)
		fi, err := os.Stat(progPath)
		if err != nil {
			continue
		}
		switch {
		case fi.IsDir():
			continue
		case fi.Mode()&0o111 != 0:
			return progPath
		}
	}
	return ""
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&fromFile, "file", "f", "", "read semantic version from `file`name, use - to read from stdin")
	rootCmd.PersistentFlags().BoolVarP(&preserveV, "preserve-v", "v", false, "if semver is prefixes with a \"v\", preserve it in the output")
	rootCmd.PersistentFlags().BoolVarP(&addV, "add-v", "V", false, "if semver is not prefixed with a \"v\", add it")
	rootCmd.PersistentFlags().BoolVarP(&longTags, "long-git-tags", "l", false, "when using git tags as semver, use long tags")
	rootCmd.PersistentFlags().BoolVarP(&addLatestGitCommitHashAsPreRelease, "add-git-hash", "g", false, "add latest git hash as prerelease to semver")
	rootCmd.PersistentFlags().StringVarP(&prerelease, "prerelease", "p", "", "add `string` as prerelease to semver")
	rootCmd.PersistentFlags().StringVarP(&fallbackSemver, "default", "d", "", "default or fallback semver if git attempt fails")

	rootCmd.AddCommand(nextCmd)
	nextCmd.AddCommand(nextMajorCmd)
	nextCmd.AddCommand(nextMinorCmd)
	nextCmd.AddCommand(nextPatchCmd)

	rootCmd.AddCommand(dumpCodeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
