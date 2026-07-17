package releaseworkflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/redact"
)

type Action string

const (
	ActionCheck   Action = "check"
	ActionTag     Action = "tag"
	ActionPublish Action = "publish"
)

type Options struct {
	Root             string
	Action           Action
	ExpectedVersion  string
	Branch           string
	Remote           string
	GitHubRepository string
	Title            string
	Confirm          bool
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
}

type Result struct {
	RepositoryRoot string
	Version        string
	Tag            string
	Commit         string
	LogPath        string
	ReleaseURL     string
	MainCommit     string
}

type releaseView struct {
	TagName      string `json:"tagName"`
	Name         string `json:"name"`
	IsDraft      bool   `json:"isDraft"`
	IsPrerelease bool   `json:"isPrerelease"`
	PublishedAt  string `json:"publishedAt"`
	URL          string `json:"url"`
}

type remoteTag struct {
	Exists    bool
	ObjectSHA string
	CommitSHA string
}

type engine struct {
	ctx    context.Context
	opts   Options
	result Result
	log    *redact.Writer
	out    *redact.Writer
	errOut *redact.Writer
	in     io.Reader
	origin string
}

var stableVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

func Run(ctx context.Context, opts Options) (result Result, runErr error) {
	defer func() {
		runErr = censorError(runErr)
	}()

	applyDefaults(&opts)
	if err := validateAction(opts.Action); err != nil {
		return result, err
	}

	root, err := resolveRoot(ctx, opts.Root)
	if err != nil {
		return result, err
	}
	result.RepositoryRoot = root

	logPath, logFile, err := openLog(root, opts.Action)
	if err != nil {
		return result, err
	}
	defer logFile.Close()
	result.LogPath = logPath

	logWriter := redact.NewWriter(logFile)
	e := &engine{
		ctx:    ctx,
		opts:   opts,
		result: result,
		log:    logWriter,
		out:    redact.NewWriter(io.MultiWriter(opts.Stdout, logWriter)),
		errOut: redact.NewWriter(io.MultiWriter(opts.Stderr, logWriter)),
		in:     opts.Stdin,
	}

	defer func() {
		result = e.result
		status := "PASS"
		if runErr != nil {
			status = "FAIL"
		}
		fmt.Fprintf(e.log, "\nFINAL STATUS: %s\n", status)
		if runErr != nil {
			fmt.Fprintf(e.log, "REASON: %v\n", runErr)
		}
		_ = e.flushWriters()
	}()

	e.heading()
	if err := e.preflight(); err != nil {
		return e.result, err
	}

	local, remote, err := e.inspectAndVerifyTagState()
	if err != nil {
		return e.result, err
	}

	if err := e.runReleaseValidation(); err != nil {
		return e.result, err
	}

	local, remote, err = e.inspectAndVerifyTagState()
	if err != nil {
		return e.result, err
	}

	switch opts.Action {
	case ActionCheck:
		e.printTagState(local, remote)
		fmt.Fprintln(e.out, "\nPASS: release candidate checks completed; no tag or remote release was changed.")
		return e.result, nil
	case ActionTag:
		if !opts.Confirm {
			return e.result, errors.New("tag requires --confirm because it creates a signed local Git tag")
		}
		if err := e.createOrVerifyLocalTag(local, remote); err != nil {
			return e.result, err
		}
		fmt.Fprintln(e.out, "\nPASS: signed local release tag is ready; nothing was pushed.")
		return e.result, nil
	case ActionPublish:
		if !opts.Confirm {
			return e.result, errors.New("publish requires --confirm because it pushes refs and creates a GitHub Release")
		}
		if !local.Exists {
			return e.result, fmt.Errorf("signed local tag %s does not exist; run the tag stage first", e.result.Tag)
		}
		if err := e.verifyLocalTag(local); err != nil {
			return e.result, err
		}
		if err := e.publishTag(remote); err != nil {
			return e.result, err
		}
		if err := e.promoteMain(); err != nil {
			return e.result, err
		}
		if err := e.publishGitHubRelease(); err != nil {
			return e.result, err
		}
		fmt.Fprintln(e.out, "\nPASS: release tag, stable main branch, and GitHub Release all identify the tested commit.")
		return e.result, nil
	default:
		return e.result, fmt.Errorf("unsupported action %q", opts.Action)
	}
}

type censoredError struct {
	err error
}

func (e censoredError) Error() string {
	return redact.Sanitize(e.err.Error())
}

func (e censoredError) Unwrap() error {
	return e.err
}

func censorError(err error) error {
	if err == nil {
		return nil
	}
	return censoredError{err: err}
}

func (e *engine) flushWriters() error {
	var flushErrors []error
	for _, writer := range []*redact.Writer{e.out, e.errOut, e.log} {
		if writer == nil {
			continue
		}
		if err := writer.Flush(); err != nil {
			flushErrors = append(flushErrors, err)
		}
	}
	return errors.Join(flushErrors...)
}

func applyDefaults(opts *Options) {
	if opts.Branch == "" {
		opts.Branch = "dev"
	}
	if opts.Remote == "" {
		opts.Remote = "origin"
	}
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
}

func validateAction(action Action) error {
	switch action {
	case ActionCheck, ActionTag, ActionPublish:
		return nil
	default:
		return fmt.Errorf("unsupported action %q", action)
	}
}

func resolveRoot(ctx context.Context, requested string) (string, error) {
	dir := strings.TrimSpace(requested)
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", errors.New("current directory is not inside a Git repository")
	}
	root, err := filepath.Abs(strings.TrimSpace(string(output)))
	if err != nil {
		return "", err
	}
	return root, nil
}

func openLog(root string, action Action) (string, *os.File, error) {
	dir := filepath.Join(root, ".local", "validation", "releases")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", nil, fmt.Errorf("create release workflow log directory: %w", err)
	}
	name := fmt.Sprintf("release-workflow-%s-%s.log", time.Now().UTC().Format("20060102T150405Z"), action)
	path := filepath.Join(dir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return "", nil, fmt.Errorf("create release workflow log: %w", err)
	}
	return path, file, nil
}

func (e *engine) heading() {
	fmt.Fprintln(e.out, "IRON SIGNAL · RELEASE WORKFLOW")
	fmt.Fprintf(e.out, "Action: %s\n", e.opts.Action)
	fmt.Fprintf(e.out, "Repository: %s\n", e.result.RepositoryRoot)
	fmt.Fprintf(e.out, "Branch: %s\n", e.opts.Branch)
	fmt.Fprintf(e.out, "Remote: %s\n", e.opts.Remote)
	fmt.Fprintln(e.out, "────────────────────────────────────────────────────────────")
}

func (e *engine) stage(name, detail string) {
	fmt.Fprintf(e.out, "\n[%s] %s\n", name, detail)
}

func (e *engine) preflight() error {
	e.stage("Preflight", "checking repository, release metadata, and authoritative refs")

	branch, err := e.capture("git", "branch", "--show-current")
	if err != nil {
		return fmt.Errorf("determine current branch: %w", err)
	}
	if branch != e.opts.Branch {
		return fmt.Errorf("current branch is %q; release workflow requires %q", branch, e.opts.Branch)
	}

	status, err := e.capture("git", "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return fmt.Errorf("inspect working tree: %w", err)
	}
	if status != "" {
		return errors.New("repository must be completely clean")
	}

	versionBytes, err := os.ReadFile(filepath.Join(e.result.RepositoryRoot, "VERSION"))
	if err != nil {
		return fmt.Errorf("read VERSION: %w", err)
	}
	version := strings.TrimSpace(string(versionBytes))
	if !stableVersionPattern.MatchString(version) {
		return fmt.Errorf("VERSION %q is not a stable MAJOR.MINOR.PATCH release", version)
	}
	if expected := strings.TrimSpace(e.opts.ExpectedVersion); expected != "" && expected != version {
		return fmt.Errorf("VERSION is %s, but --version requested %s", version, expected)
	}
	e.result.Version = version
	e.result.Tag = "isras-v" + version
	e.result.Commit, err = e.capture("git", "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("resolve HEAD: %w", err)
	}

	notes := filepath.Join(e.result.RepositoryRoot, "docs", "releases", version+".md")
	info, err := os.Stat(notes)
	if err != nil {
		return fmt.Errorf("release notes %s are unavailable: %w", relative(e.result.RepositoryRoot, notes), err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("release notes %s are empty", relative(e.result.RepositoryRoot, notes))
	}
	changelog, err := os.ReadFile(filepath.Join(e.result.RepositoryRoot, "CHANGELOG.md"))
	if err != nil {
		return fmt.Errorf("read CHANGELOG.md: %w", err)
	}
	if !strings.Contains(string(changelog), "## "+version+" —") {
		return fmt.Errorf("CHANGELOG.md does not contain the release heading for %s", version)
	}

	e.origin, err = e.capturePrivate("git", "remote", "get-url", e.opts.Remote)
	if err != nil {
		return fmt.Errorf("read remote URL: %w", err)
	}
	if e.origin == "" {
		return fmt.Errorf("remote %s has an empty URL", e.opts.Remote)
	}
	fmt.Fprintf(e.log, "REMOTE URL: %s\n", sanitizeOrigin(e.origin))

	var remoteBranch string
	if err := e.retryRead("read authoritative release branch", func() error {
		output, err := e.capture("git", "ls-remote", "--heads", e.opts.Remote, "refs/heads/"+e.opts.Branch)
		if err != nil {
			return err
		}
		remoteBranch, err = parseSingleRemoteRef(output, "refs/heads/"+e.opts.Branch)
		return err
	}); err != nil {
		return err
	}
	if remoteBranch != e.result.Commit {
		return fmt.Errorf("local HEAD %s does not equal %s/%s %s", short(e.result.Commit), e.opts.Remote, e.opts.Branch, short(remoteBranch))
	}

	if err := e.run("git", "verify-commit", e.result.Commit); err != nil {
		return fmt.Errorf("verify exact release commit: %w", err)
	}

	fmt.Fprintf(e.out, "Version: %s\n", e.result.Version)
	fmt.Fprintf(e.out, "Tag: %s\n", e.result.Tag)
	fmt.Fprintf(e.out, "Commit: %s\n", e.result.Commit)
	return nil
}

func (e *engine) runReleaseValidation() error {
	e.stage("Validation", "building repository-owned tooling and validating the exact pushed commit")
	commands := [][]string{
		{"./tools/build-validator.sh"},
		{"./tools/build-release-validator.sh"},
		{"./.local/bin/isras-validate", "all", "--mode", "commit"},
		{"./.local/bin/isras-release-validate", "--ref", e.opts.Branch},
	}
	for _, command := range commands {
		if err := e.run(command[0], command[1:]...); err != nil {
			return fmt.Errorf("release validation command %s failed: %w", strings.Join(command, " "), err)
		}
	}
	return nil
}

func (e *engine) inspectAndVerifyTagState() (remoteTag, remoteTag, error) {
	local, remote, err := e.inspectTag()
	if err != nil {
		return local, remote, err
	}
	if err := validateTagIdentity(local, remote, e.result.Commit, e.result.Tag); err != nil {
		return local, remote, err
	}
	if local.Exists {
		if err := e.verifyLocalTag(local); err != nil {
			return local, remote, err
		}
	}
	return local, remote, nil
}

func validateTagIdentity(local, remote remoteTag, expectedCommit, tag string) error {
	if local.Exists && local.CommitSHA != expectedCommit {
		return fmt.Errorf(
			"local tag %s identifies %s, but the current release candidate is %s; release tags are immutable, so advance VERSION and release metadata instead of reusing the existing tag",
			tag,
			short(local.CommitSHA),
			short(expectedCommit),
		)
	}
	if remote.Exists && remote.CommitSHA != expectedCommit {
		return fmt.Errorf(
			"remote tag %s identifies %s, but the current release candidate is %s; release tags are immutable, so advance VERSION and release metadata instead of reusing the existing tag",
			tag,
			short(remote.CommitSHA),
			short(expectedCommit),
		)
	}
	if remote.Exists && !local.Exists {
		return fmt.Errorf(
			"remote tag %s exists without a corresponding local tag; fetch and verify the immutable tag before continuing",
			tag,
		)
	}
	if local.Exists && remote.Exists && local.ObjectSHA != remote.ObjectSHA {
		return fmt.Errorf(
			"remote tag object %s does not match local tag object %s",
			short(remote.ObjectSHA),
			short(local.ObjectSHA),
		)
	}
	return nil
}

func (e *engine) inspectTag() (remoteTag, remoteTag, error) {
	local := remoteTag{}
	object, err := e.captureAllowExit(1, "git", "show-ref", "--verify", "--hash", "refs/tags/"+e.result.Tag)
	if err != nil {
		return local, remoteTag{}, fmt.Errorf("inspect local release tag: %w", err)
	}
	if object != "" {
		local.Exists = true
		local.ObjectSHA = object
		target, err := e.capture("git", "rev-list", "-n", "1", e.result.Tag)
		if err != nil {
			return local, remoteTag{}, fmt.Errorf("resolve local release tag target: %w", err)
		}
		local.CommitSHA = target
	}

	var remote remoteTag
	err = e.retryRead("inspect remote release tag", func() error {
		output, err := e.capture("git", "ls-remote", "--tags", e.opts.Remote,
			"refs/tags/"+e.result.Tag,
			"refs/tags/"+e.result.Tag+"^{}")
		if err != nil {
			return err
		}
		remote, err = parseRemoteTag(output, e.result.Tag)
		return err
	})
	if err != nil {
		return local, remoteTag{}, err
	}
	return local, remote, nil
}

func (e *engine) printTagState(local, remote remoteTag) {
	switch {
	case !local.Exists && !remote.Exists:
		fmt.Fprintln(e.out, "Tag state: absent locally and remotely; candidate is ready for the tag stage.")
	case local.Exists && !remote.Exists:
		fmt.Fprintln(e.out, "Tag state: signed local tag exists; candidate is ready for the publish stage after verification.")
	case local.Exists && remote.Exists:
		fmt.Fprintln(e.out, "Tag state: local and remote tags exist; both must identify the tested commit.")
	default:
		fmt.Fprintln(e.out, "Tag state: remote tag exists without a corresponding local tag.")
	}
}

func (e *engine) createOrVerifyLocalTag(local, remote remoteTag) error {
	e.stage("Tag", "creating or verifying the signed annotated local release tag")
	if remote.Exists && !local.Exists {
		return fmt.Errorf("remote tag %s already exists without a local tag; fetch and investigate before continuing", e.result.Tag)
	}
	if local.Exists {
		if err := e.verifyLocalTag(local); err != nil {
			return err
		}
		if remote.Exists {
			if err := compareRemoteTag(local, remote, e.result.Commit); err != nil {
				return err
			}
		}
		fmt.Fprintf(e.out, "Local tag already exists and is valid: %s\n", e.result.Tag)
		return nil
	}

	title := e.releaseTitle()
	if err := e.runInteractive("git", "tag", "-s", "-a", e.result.Tag, e.result.Commit, "-m", title); err != nil {
		return fmt.Errorf("create signed annotated tag: %w", err)
	}
	object, err := e.capture("git", "rev-parse", "refs/tags/"+e.result.Tag)
	if err != nil {
		return fmt.Errorf("resolve created tag object: %w", err)
	}
	created := remoteTag{Exists: true, ObjectSHA: object, CommitSHA: e.result.Commit}
	return e.verifyLocalTag(created)
}

func (e *engine) verifyLocalTag(local remoteTag) error {
	if !local.Exists {
		return fmt.Errorf("local tag %s does not exist", e.result.Tag)
	}
	typeName, err := e.capture("git", "cat-file", "-t", e.result.Tag)
	if err != nil {
		return fmt.Errorf("inspect local tag object type: %w", err)
	}
	if typeName != "tag" {
		return fmt.Errorf("%s is %q, not an annotated tag object", e.result.Tag, typeName)
	}
	if err := e.run("git", "verify-tag", e.result.Tag); err != nil {
		return fmt.Errorf("verify signed local release tag: %w", err)
	}
	target, err := e.capture("git", "rev-list", "-n", "1", e.result.Tag)
	if err != nil {
		return fmt.Errorf("resolve local release tag target: %w", err)
	}
	if target != e.result.Commit {
		return fmt.Errorf("local tag %s identifies %s, expected tested commit %s", e.result.Tag, short(target), short(e.result.Commit))
	}
	return nil
}

func (e *engine) publishTag(remote remoteTag) error {
	e.stage("Publish tag", "pushing and verifying the signed annotated release tag")
	localObject, err := e.capture("git", "rev-parse", "refs/tags/"+e.result.Tag)
	if err != nil {
		return fmt.Errorf("resolve local tag object: %w", err)
	}
	local := remoteTag{Exists: true, ObjectSHA: localObject, CommitSHA: e.result.Commit}

	if remote.Exists {
		return compareRemoteTag(local, remote, e.result.Commit)
	}

	pushErr := e.runInteractive("git", "push", e.opts.Remote, "refs/tags/"+e.result.Tag)
	var observed remoteTag
	inspectErr := e.retryRead("verify pushed remote tag", func() error {
		output, err := e.capture("git", "ls-remote", "--tags", e.opts.Remote,
			"refs/tags/"+e.result.Tag,
			"refs/tags/"+e.result.Tag+"^{}")
		if err != nil {
			return err
		}
		observed, err = parseRemoteTag(output, e.result.Tag)
		return err
	})
	if inspectErr != nil {
		if pushErr != nil {
			return fmt.Errorf("push tag failed and remote state could not be verified: push: %v; verify: %w", pushErr, inspectErr)
		}
		return inspectErr
	}
	if err := compareRemoteTag(local, observed, e.result.Commit); err != nil {
		if pushErr != nil {
			return fmt.Errorf("push tag reported failure and remote verification did not prove success: push: %v; verify: %w", pushErr, err)
		}
		return err
	}
	if pushErr != nil {
		fmt.Fprintln(e.out, "Push reported an error, but remote verification proved the exact tag was published.")
	}
	return nil
}

func (e *engine) promoteMain() error {
	e.stage("Promote main", "fast-forwarding the stable branch to the tested release commit")
	var remoteMain string
	if err := e.retryRead("read remote main", func() error {
		output, err := e.capture("git", "ls-remote", "--heads", e.opts.Remote, "refs/heads/main")
		if err != nil {
			return err
		}
		remoteMain, err = parseSingleRemoteRef(output, "refs/heads/main")
		return err
	}); err != nil {
		return err
	}
	if remoteMain == e.result.Commit {
		e.result.MainCommit = remoteMain
		fmt.Fprintln(e.out, "main already identifies the tested release commit.")
		return nil
	}

	presentExit, presentErr := e.commandStatus("git", "cat-file", "-e", remoteMain+"^{commit}")
	if presentErr != nil {
		return fmt.Errorf("inspect remote main commit locally: %w", presentErr)
	}
	if presentExit != 0 {
		if err := e.retryRead("fetch remote main commit", func() error {
			return e.run("git", "fetch", "--no-tags", e.opts.Remote, remoteMain)
		}); err != nil {
			return err
		}
	}

	ancestorExit, ancestorErr := e.commandStatus("git", "merge-base", "--is-ancestor", remoteMain, e.result.Commit)
	if ancestorErr != nil {
		return fmt.Errorf("determine whether main can fast-forward: %w", ancestorErr)
	}
	if ancestorExit != 0 {
		return fmt.Errorf("main cannot be fast-forwarded safely from %s to %s", short(remoteMain), short(e.result.Commit))
	}

	pushErr := e.runInteractive("git", "push", e.opts.Remote, e.result.Commit+":refs/heads/main")
	var observed string
	verifyErr := e.retryRead("verify remote main", func() error {
		output, err := e.capture("git", "ls-remote", "--heads", e.opts.Remote, "refs/heads/main")
		if err != nil {
			return err
		}
		observed, err = parseSingleRemoteRef(output, "refs/heads/main")
		return err
	})
	if verifyErr != nil {
		if pushErr != nil {
			return fmt.Errorf("main push failed and remote state could not be verified: push: %v; verify: %w", pushErr, verifyErr)
		}
		return verifyErr
	}
	if observed != e.result.Commit {
		return fmt.Errorf("remote main is %s after publication, expected %s", short(observed), short(e.result.Commit))
	}
	if pushErr != nil {
		fmt.Fprintln(e.out, "Main push reported an error, but remote verification proved the exact fast-forward completed.")
	}
	e.result.MainCommit = observed
	return nil
}

func (e *engine) publishGitHubRelease() error {
	e.stage("GitHub Release", "creating or verifying the published non-prerelease release")
	if _, err := exec.LookPath("gh"); err != nil {
		return errors.New("GitHub CLI gh is not installed")
	}
	if err := e.run("gh", "auth", "status", "--active", "--hostname", "github.com"); err != nil {
		return fmt.Errorf("GitHub CLI authentication is unavailable: %w", err)
	}

	repository := strings.TrimSpace(e.opts.GitHubRepository)
	if repository == "" {
		var err error
		repository, err = parseGitHubRepository(e.origin)
		if err != nil {
			return err
		}
	}
	title := e.releaseTitle()
	notes := filepath.Join(e.result.RepositoryRoot, "docs", "releases", e.result.Version+".md")

	view, exists, err := e.viewRelease(repository)
	if err != nil {
		return err
	}
	if !exists {
		createErr := e.runInteractive("gh", "release", "create", e.result.Tag,
			"--repo", repository,
			"--verify-tag",
			"--title", title,
			"--notes-file", notes,
			"--latest")
		view, exists, err = e.viewRelease(repository)
		if err != nil {
			if createErr != nil {
				return fmt.Errorf("GitHub Release creation failed and the resulting state could not be verified: create: %v; verify: %w", createErr, err)
			}
			return err
		}
		if !exists {
			if createErr != nil {
				return fmt.Errorf("GitHub Release creation failed: %w", createErr)
			}
			return errors.New("GitHub Release was not found after creation")
		}
		if createErr != nil {
			fmt.Fprintln(e.out, "Release creation reported an error, but GitHub verification found the expected release.")
		}
	}
	if err := verifyReleaseView(view, e.result.Tag, title); err != nil {
		return err
	}
	e.result.ReleaseURL = view.URL
	fmt.Fprintf(e.out, "Release URL: %s\n", view.URL)
	return nil
}

func (e *engine) viewRelease(repository string) (releaseView, bool, error) {
	args := []string{"release", "view", e.result.Tag,
		"--repo", repository,
		"--json", "tagName,name,isDraft,isPrerelease,publishedAt,url"}
	output, err := e.captureAllowAnyExit("gh", args...)
	if err == nil {
		var view releaseView
		if err := json.Unmarshal([]byte(output), &view); err != nil {
			return releaseView{}, false, fmt.Errorf("parse GitHub Release response: %w", err)
		}
		return view, true, nil
	}
	fmt.Fprintf(e.log, "INFO: gh release view did not find a usable release: %v\n", err)
	return releaseView{}, false, nil
}

func (e *engine) releaseTitle() string {
	if title := strings.TrimSpace(e.opts.Title); title != "" {
		return title
	}
	return "ISRAS " + e.result.Version + " — Solo Developer Baseline"
}

func (e *engine) retryRead(label string, fn func() error) error {
	var last error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			last = err
			fmt.Fprintf(e.log, "READ RETRY %d/3 (%s): %v\n", attempt, label, err)
		}
		if attempt < 3 {
			select {
			case <-e.ctx.Done():
				return e.ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
	}
	return fmt.Errorf("%s failed after 3 attempts: %w", label, last)
}

func (e *engine) run(name string, args ...string) error {
	_, err := e.execute(false, true, name, args...)
	return err
}

func (e *engine) runInteractive(name string, args ...string) error {
	_, err := e.execute(true, true, name, args...)
	return err
}

func (e *engine) capture(name string, args ...string) (string, error) {
	return e.execute(false, false, name, args...)
}

func (e *engine) capturePrivate(name string, args ...string) (string, error) {
	cmd := exec.CommandContext(e.ctx, name, args...)
	cmd.Dir = e.result.RepositoryRoot
	var buffer boundedBuffer
	cmd.Stdout = &buffer
	cmd.Stderr = &buffer
	err := cmd.Run()
	output := strings.TrimSpace(buffer.String())
	if err == nil {
		return output, nil
	}
	exitCode := -1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}
	return "", &commandError{Command: safeCommand(name, args), ExitCode: exitCode, Output: "output withheld"}
}

func (e *engine) commandStatus(name string, args ...string) (int, error) {
	cmd := exec.CommandContext(e.ctx, name, args...)
	cmd.Dir = e.result.RepositoryRoot
	var buffer boundedBuffer
	cmd.Stdout = &buffer
	cmd.Stderr = &buffer
	fmt.Fprintf(e.log, "\nCOMMAND: %s\n", safeCommand(name, args))
	err := cmd.Run()
	if buffer.Len() > 0 {
		fmt.Fprintf(e.log, "OUTPUT:\n%s\n", strings.TrimSpace(buffer.String()))
	}
	if err == nil {
		fmt.Fprintln(e.log, "EXIT CODE: 0")
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		fmt.Fprintf(e.log, "EXIT CODE: %d\n", exitErr.ExitCode())
		return exitErr.ExitCode(), nil
	}
	return -1, err
}

func (e *engine) captureAllowExit(allowed int, name string, args ...string) (string, error) {
	output, err := e.execute(false, false, name, args...)
	if err == nil {
		return output, nil
	}
	var commandErr *commandError
	if errors.As(err, &commandErr) && commandErr.ExitCode == allowed {
		return "", nil
	}
	return "", err
}

func (e *engine) captureAllowAnyExit(name string, args ...string) (string, error) {
	return e.execute(false, false, name, args...)
}

const maxCapturedCommandOutput = 1 << 20

type boundedBuffer struct {
	mu        sync.Mutex
	buffer    bytes.Buffer
	truncated bool
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	originalLength := len(data)
	remaining := maxCapturedCommandOutput - b.buffer.Len()
	if remaining > 0 {
		if remaining > len(data) {
			remaining = len(data)
		}
		_, _ = b.buffer.Write(data[:remaining])
	}
	if remaining < len(data) {
		b.truncated = true
	}
	return originalLength, nil
}

func (b *boundedBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Len()
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	output := b.buffer.String()
	if b.truncated {
		output = strings.TrimRight(output, "\r\n") + "\n[OUTPUT TRUNCATED AT 1 MiB]"
	}
	return output
}

type commandError struct {
	Command  string
	ExitCode int
	Output   string
}

func (e *commandError) Error() string {
	if e.Output == "" {
		return fmt.Sprintf("%s exited %d", e.Command, e.ExitCode)
	}
	return fmt.Sprintf("%s exited %d: %s", e.Command, e.ExitCode, e.Output)
}

func (e *engine) execute(interactive, stream bool, name string, args ...string) (string, error) {
	fmt.Fprintf(e.log, "\nCOMMAND: %s\n", safeCommand(name, args))
	cmd := exec.CommandContext(e.ctx, name, args...)
	cmd.Dir = e.result.RepositoryRoot
	if interactive {
		cmd.Stdin = e.in
	}

	var buffer boundedBuffer
	if stream {
		cmd.Stdout = io.MultiWriter(e.out, &buffer)
		cmd.Stderr = io.MultiWriter(e.errOut, &buffer)
	} else {
		cmd.Stdout = &buffer
		cmd.Stderr = &buffer
	}

	err := cmd.Run()
	if stream {
		if flushErr := errors.Join(e.out.Flush(), e.errOut.Flush()); flushErr != nil {
			err = errors.Join(err, fmt.Errorf("flush censored command output: %w", flushErr))
		}
	}
	output := strings.TrimSpace(buffer.String())
	if !stream && output != "" {
		fmt.Fprintf(e.log, "OUTPUT:\n%s\n", output)
	}
	if err == nil {
		fmt.Fprintln(e.log, "EXIT CODE: 0")
		if stream {
			return redact.Sanitize(output), nil
		}
		return output, nil
	}
	exitCode := -1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}
	fmt.Fprintf(e.log, "EXIT CODE: %d\n", exitCode)
	return redact.Sanitize(output), &commandError{Command: safeCommand(name, args), ExitCode: exitCode, Output: concise(output)}
}

func parseRemoteTag(output, tag string) (remoteTag, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return remoteTag{}, nil
	}
	var result remoteTag
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return remoteTag{}, errors.New("remote tag lookup returned malformed output")
		}
		switch fields[1] {
		case "refs/tags/" + tag:
			result.Exists = true
			result.ObjectSHA = fields[0]
		case "refs/tags/" + tag + "^{}":
			result.CommitSHA = fields[0]
		default:
			return remoteTag{}, fmt.Errorf("remote tag lookup returned unexpected ref %q", fields[1])
		}
	}
	if result.Exists && result.CommitSHA == "" {
		return remoteTag{}, fmt.Errorf("remote tag %s is not an annotated tag", tag)
	}
	if !result.Exists && result.CommitSHA != "" {
		return remoteTag{}, fmt.Errorf("remote tag %s returned a peeled commit without a tag object", tag)
	}
	return result, nil
}

func compareRemoteTag(local, remote remoteTag, expectedCommit string) error {
	if !remote.Exists {
		return errors.New("remote release tag does not exist")
	}
	if remote.ObjectSHA != local.ObjectSHA {
		return fmt.Errorf("remote tag object %s does not match local tag object %s", short(remote.ObjectSHA), short(local.ObjectSHA))
	}
	if remote.CommitSHA != expectedCommit {
		return fmt.Errorf("remote tag identifies %s, expected tested commit %s", short(remote.CommitSHA), short(expectedCommit))
	}
	return nil
}

func parseSingleRemoteRef(output, expectedRef string) (string, error) {
	lines := nonEmptyLines(output)
	if len(lines) != 1 {
		return "", fmt.Errorf("remote ref lookup returned %d entries, expected one", len(lines))
	}
	fields := strings.Fields(lines[0])
	if len(fields) != 2 || fields[1] != expectedRef {
		return "", fmt.Errorf("remote ref lookup did not return %s", expectedRef)
	}
	return fields[0], nil
}

func parseGitHubRepository(origin string) (string, error) {
	origin = strings.TrimSpace(origin)
	var path string
	switch {
	case strings.HasPrefix(origin, "git@github.com:"):
		path = strings.TrimPrefix(origin, "git@github.com:")
	case strings.HasPrefix(origin, "ssh://git@github.com/"):
		path = strings.TrimPrefix(origin, "ssh://git@github.com/")
	case strings.HasPrefix(origin, "https://github.com/"):
		path = strings.TrimPrefix(origin, "https://github.com/")
	case strings.HasPrefix(origin, "http://github.com/"):
		path = strings.TrimPrefix(origin, "http://github.com/")
	default:
		return "", fmt.Errorf("cannot derive GitHub repository from remote URL %q; supply --github-repo owner/name", origin)
	}
	path = strings.TrimSuffix(path, ".git")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("remote URL %q does not contain an owner/name repository", origin)
	}
	return parts[0] + "/" + parts[1], nil
}

func verifyReleaseView(view releaseView, tag, title string) error {
	if view.TagName != tag {
		return fmt.Errorf("GitHub Release tag is %q, expected %q", view.TagName, tag)
	}
	if view.Name != title {
		return fmt.Errorf("GitHub Release title is %q, expected %q", view.Name, title)
	}
	if view.IsDraft {
		return errors.New("GitHub Release remains a draft")
	}
	if view.IsPrerelease {
		return errors.New("GitHub Release is marked as a prerelease")
	}
	if strings.TrimSpace(view.PublishedAt) == "" {
		return errors.New("GitHub Release has no publication timestamp")
	}
	if strings.TrimSpace(view.URL) == "" {
		return errors.New("GitHub Release has no URL")
	}
	return nil
}

func nonEmptyLines(value string) []string {
	var lines []string
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func safeCommand(name string, args []string) string {
	parts := []string{shellQuote(name)}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return redact.Sanitize(strings.Join(parts, " "))
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z') &&
			!(r >= 'A' && r <= 'Z') &&
			!(r >= '0' && r <= '9') &&
			r != '/' && r != '.' && r != '_' && r != '-' && r != ':' && r != '@' && r != '+' && r != '=' {
			return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
		}
	}
	return value
}

func sanitizeOrigin(origin string) string {
	if !strings.Contains(origin, "://") {
		return redact.Sanitize(origin)
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.User == nil {
		return redact.Sanitize(origin)
	}
	parsed.User = url.User("REDACTED")
	return redact.Sanitize(parsed.String())
}

func concise(value string) string {
	value = redact.Sanitize(value)
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 500 {
		return value[:500] + "…"
	}
	return value
}

func short(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

func relative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return filepath.ToSlash(rel)
}
