// Package vmman provides a VM manager plugin.
//
// This file ports the DistroHopper Go frontend backend into vmsman, allowing
// VMs to be discovered, downloaded and launched from supported OS templates.
package vmman

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// DistroConfig holds DistroHopper-related settings.
type DistroConfig struct {
	VMs       string
	RAMDisk   string
	RAMDir    string
	ISOs      string
	Templates string
	Terminal  string
	SourceDir string
}

// DistroApp is the ported DistroHopper application backend.
type DistroApp struct {
	ctx        context.Context
	source     string
	executable string
	configDir  string
	icons      string
	cfg        DistroConfig
}

// isFile reports whether path is a regular file.
func isFile(path string) bool { s, e := os.Stat(path); return e == nil && s.Mode().IsRegular() }

// NewDistroApp creates a new ported DistroHopper backend.
func NewDistroApp(ctx context.Context, cfg DistroConfig) (*DistroApp, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return nil, err
	}
	source := cfg.SourceDir
	if source == "" {
		source = os.Getenv("DH_SOURCE_DIR")
	}
	if source == "" {
		source = os.Getenv("APPDIR")
	}
	if source == "" {
		source = filepath.Dir(exe)
		if filepath.Base(source) == "build" && isFile(filepath.Join(source, "..", "go.mod")) {
			source = filepath.Dir(source)
		}
	}
	source, err = filepath.Abs(source)
	if err != nil {
		return nil, err
	}
	// Prefer a directory that actually contains the DistroHopper tools/assets.
	for _, candidate := range []string{
		source,
		filepath.Join(source, "src", "vmsman", "assets"),
		filepath.Join(source, "vmsman", "assets"),
		filepath.Join(filepath.Dir(exe), "..", "share", "sysman", "vmsman"),
		"/usr/share/sysman/vmsman",
	} {
		if isFile(filepath.Join(candidate, "qget")) {
			source = candidate
			break
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	a := &DistroApp{ctx: ctx, source: source, executable: exe, configDir: filepath.Join(home, ".config", "sysman", "vmsman")}
	a.icons = filepath.Join(a.configDir, "icons")
	a.cfg = cfg
	if a.cfg.VMs == "" {
		a.cfg.VMs = filepath.Join(home, "vm")
	}
	if a.cfg.RAMDisk == "" {
		a.cfg.RAMDisk = "no"
	}
	if a.cfg.RAMDir == "" {
		a.cfg.RAMDir = "/dev/shm/quickemu"
	}
	if a.cfg.Templates == "" {
		a.cfg.Templates = "no"
	}
	if a.cfg.SourceDir != "" {
		a.source = a.cfg.SourceDir
		if s, err := filepath.Abs(a.cfg.SourceDir); err == nil {
			a.source = s
		}
	}
	// Copy vendored icons and create required config directories. Best-effort:
	// missing icons fall back to the bundled tux.svg placeholder.
	_ = a.prepare()
	return a, nil
}

func (a *DistroApp) env() []string {
	env := os.Environ()
	for k, v := range map[string]string{"DH_CONFIG_DIR": a.configDir, "DH_CONFIG": filepath.Join(a.configDir, "config"), "DH_ICON_DIR": a.icons, "VMS_DIR": a.cfg.VMs, "RAMDISK": a.cfg.RAMDisk, "RAMDISK_DIR": a.cfg.RAMDir, "ISO_DIR": a.cfg.ISOs, "USE_TEMPLATES": a.cfg.Templates, "DH_USE_TEMPLATES": "no"} {
		if k == "DH_USE_TEMPLATES" && a.cfg.Templates == "yes" {
			v = "yes"
		}
		env = append(env, k+"="+v)
	}
	return env
}

func (a *DistroApp) command(name string, args ...string) *exec.Cmd {
	c := exec.CommandContext(a.ctx, name, args...)
	c.Env = append(a.env(), "LC_ALL=C")
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		if c.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	c.WaitDelay = 2 * time.Second
	return c
}

func (a *DistroApp) tool(name string) string {
	for _, p := range []string{filepath.Join(a.source, name), filepath.Join(a.source, "bin", name)} {
		if s, e := os.Stat(p); e == nil && s.Mode()&0111 != 0 && !s.IsDir() {
			return p
		}
	}
	return name
}

func (a *DistroApp) output(name string, args ...string) (string, error) {
	c := a.command(name, args...)
	var stderr strings.Builder
	c.Stderr = &stderr
	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func (a *DistroApp) prepare() error {
	for _, dir := range []string{a.configDir, a.icons, filepath.Join(a.configDir, "ready"), filepath.Join(a.configDir, "supported")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	files, _ := filepath.Glob(filepath.Join(a.source, "icons", "*"))
	for _, f := range files {
		if isFile(f) {
			if err := copyFile(f, filepath.Join(a.icons, filepath.Base(f)), 0644); err != nil {
				return err
			}
		}
	}
	if !isFile(filepath.Join(a.icons, "hop.png")) {
		for _, p := range []string{filepath.Join(a.source, "website-source", "hop220.png")} {
			if isFile(p) {
				if err := copyFile(p, filepath.Join(a.icons, "hop.png"), 0644); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

// OperatingSystem is a supported or ready VM template.
type OperatingSystem struct {
	ID   string
	Name string
	Icon string
}

// Metadata is qget template metadata.
type Metadata struct {
	Name                              string
	Description                       string
	Architectures, Releases, Editions []string
}

func parseMetadata(s string) Metadata {
	var m Metadata
	inEditions := false
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		key, value, hasColon := strings.Cut(line, ":")
		if m.Name == "" && line != "" && !hasColon && !inEditions {
			m.Name = line
		}
		if hasColon {
			inEditions = false
			switch strings.TrimSpace(key) {
			case "Description":
				m.Description = strings.TrimSpace(value)
			case "Archs", "Architectures":
				m.Architectures = strings.Fields(value)
			case "Releases":
				m.Releases = strings.Fields(value)
			case "EditionsList":
				inEditions = true
			}
		} else if inEditions && line != "" {
			m.Editions = append(m.Editions, line)
		}
	}
	return m
}

func validName(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, "/\\\x00\r\n") && !strings.HasPrefix(s, "-")
}

func validOSID(id string) bool {
	if !validName(id) {
		return false
	}
	for _, c := range id {
		allowed := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.'
		if !allowed {
			return false
		}
	}
	return true
}

func (a *DistroApp) systems() ([]OperatingSystem, error) {
	ids, err := a.supported()
	if err != nil {
		return nil, err
	}
	systems := make([]OperatingSystem, 0, len(ids))
	for _, id := range ids {
		details := a.templateDetails(id)
		label := details.Name
		if label == "" {
			label = id
		}
		systems = append(systems, OperatingSystem{ID: id, Name: label, Icon: a.resolveIcon(id, details.Icon)})
	}
	sort.Slice(systems, func(i, j int) bool { return strings.ToLower(systems[i].Name) < strings.ToLower(systems[j].Name) })
	return systems, nil
}

func (a *DistroApp) supported() ([]string, error) {
	out, err := a.output(a.tool("qget"), "--list-os")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !validOSID(line) {
			return nil, fmt.Errorf("qget --list-os returned a display name or invalid template ID: %q", line)
		}
		names = append(names, line)
	}
	if len(names) == 0 {
		return nil, errors.New("qget --list-os produced no entries")
	}
	return names, nil
}

func (a *DistroApp) ready() ([]string, error) {
	entries, err := os.ReadDir(a.cfg.VMs)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".conf") && e.Name() != "distrohopper.conf" {
			names = append(names, strings.TrimSuffix(e.Name(), ".conf"))
		}
	}
	return names, nil
}

type templateDetails struct {
	Name, Description, Icon string
}

func (a *DistroApp) templateInfo(name string) (string, string) {
	d := a.templateDetails(name)
	return d.Description, d.Icon
}

func (a *DistroApp) templateDetails(name string) templateDetails {
	for name != "" {
		path := filepath.Join(a.source, "templates", name)
		if !isFile(path) {
			path = filepath.Join(a.source, "public", name)
		}
		if isFile(path) {
			c := a.command("bash", "-c", `OS="$2"; source "$1" >/dev/null 2>&1; printf '%s\0%s\0%s' "${PRETTY:-}" "${DESCRIPTION:-}" "${ICON:-}"`, "dh-template", path, name)
			out, err := c.Output()
			if err != nil {
				return templateDetails{}
			}
			values := strings.SplitN(string(out), "\x00", 3)
			if len(values) == 3 {
				return templateDetails{values[0], values[1], values[2]}
			}
			return templateDetails{}
		}
		i := strings.LastIndex(name, "-")
		if i < 0 {
			break
		}
		name = name[:i]
	}
	return templateDetails{}
}

func (a *DistroApp) resolveIcon(name, icon string) string {
	if filepath.Base(icon) == icon && isFile(filepath.Join(a.icons, icon)) {
		return filepath.Join(a.icons, icon)
	}
	for candidate := name; candidate != ""; {
		for _, ext := range []string{".svg", ".png"} {
			p := filepath.Join(a.icons, candidate+ext)
			if isFile(p) {
				return p
			}
		}
		i := strings.LastIndex(candidate, "-")
		if i < 0 {
			break
		}
		candidate = candidate[:i]
	}
	return filepath.Join(a.icons, "tux.svg")
}

func (a *DistroApp) iconFor(name string) string {
	_, icon := a.templateInfo(name)
	return a.resolveIcon(name, icon)
}

func (a *DistroApp) query(osID, arch, release string) (Metadata, error) {
	if !validOSID(osID) {
		return Metadata{}, errors.New("use a template ID such as alpine-linux, not a display name")
	}
	args := []string{"--show", "--plain"}
	if arch != "" {
		args = append(args, "--arch", arch)
	}
	if release != "" {
		args = append(args, "--release", release)
	}
	args = append(args, osID)
	out, err := a.output(a.tool("qget"), args...)
	if err != nil {
		return Metadata{}, err
	}
	m := parseMetadata(out)
	if len(m.Architectures) == 0 {
		m.Architectures = []string{"x86_64"}
	}
	return m, nil
}

func desktopQuote(s string) string {
	r := strings.NewReplacer("\\", "\\\\\\\\", "\"", "\\\\\"", "$", "\\\\$", "`", "\\\\`", "%", "%%")
	return "\"" + r.Replace(s) + "\""
}

func keyValue(s string) string {
	return strings.NewReplacer("\\", "\\\\", "\n", "\\n", "\r", "\\r", "\t", "\\t").Replace(s)
}

func (a *DistroApp) desktop(kind, name, label, description, command string) error {
	if !validName(name) {
		return fmt.Errorf("invalid desktop name %q", name)
	}
	text := fmt.Sprintf("[Desktop Entry]\nVersion=1.0\nType=Application\nName=%s\nComment=%s\nExec=%s\nIcon=%s\nTerminal=false\nCategories=System;X-Virtualization;\n", keyValue(label), keyValue(description), command, keyValue(a.iconFor(name)))
	return atomicWrite(filepath.Join(a.configDir, kind, name+".desktop"), []byte(text), 0755)
}

func pruneEntries(dir string, names []string) error {
	keep := map[string]bool{}
	for _, name := range names {
		keep[name+".desktop"] = true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".desktop") && !keep[e.Name()] {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// pickerExecutable resolves the standalone vmsman-create binary used by
// generated supported-OS .desktop files, so they do not depend on vmsman or
// sysman being installed.
func (a *DistroApp) pickerExecutable() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for _, cand := range []string{filepath.Join(dir, "vmsman-create"), filepath.Join(dir, "..", "bin", "vmsman-create")} {
			if st, err := os.Stat(cand); err == nil && !st.IsDir() && st.Mode()&0111 != 0 {
				return cand
			}
		}
	}
	if p, err := exec.LookPath("vmsman-create"); err == nil {
		return p
	}
	for _, cand := range []string{"/usr/local/bin/vmsman-create", "/usr/bin/vmsman-create"} {
		if st, err := os.Stat(cand); err == nil && !st.IsDir() && st.Mode()&0111 != 0 {
			return cand
		}
	}
	return "vmsman-create"
}

func (a *DistroApp) frontendExecutable() string {
	if image := os.Getenv("APPIMAGE"); isFile(image) {
		return image
	}
	return a.executable
}

func (a *DistroApp) renewReady() error {
	names, err := a.ready()
	if err != nil {
		return err
	}
	for _, name := range names {
		desc, _ := a.templateInfo(name)
		conf, err := filepath.Abs(filepath.Join(a.cfg.VMs, name+".conf"))
		if err != nil {
			return err
		}
		command := desktopQuote(a.tool("quickemu")) + " --vm " + desktopQuote(conf)
		if a.frontendExecutable() != a.executable {
			command = desktopQuote(a.frontendExecutable()) + " run " + desktopQuote(name)
		}
		if err := a.desktop("ready", name, name, desc, command); err != nil {
			return err
		}
	}
	return pruneEntries(filepath.Join(a.configDir, "ready"), names)
}

func (a *DistroApp) renewSupported(progress func(int, int, string)) error {
	jobs := 4
	if v := os.Getenv("DH_RENEW_JOBS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return errors.New("DH_RENEW_JOBS must be a positive integer")
		}
		jobs = n
	}
	lock, err := os.OpenFile(filepath.Join(a.configDir, "renew.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("regeneration already running or lock unavailable: %w", err)
	}
	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) }()
	names, err := a.supported()
	if err != nil {
		return err
	}
	if jobs > len(names) {
		jobs = len(names)
	}
	type result struct {
		name string
		err  error
	}
	tasks := make(chan string)
	results := make(chan result)
	var wg sync.WaitGroup
	for i := 0; i < jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range tasks {
				m, err := a.query(name, "", "")
				if err == nil {
					label := m.Name
					if label == "" {
						label = a.templateDetails(name).Name
					}
					if label == "" {
						label = name
					}
					err = a.desktop("supported", name, label, m.Description, desktopQuote(a.pickerExecutable())+" create "+desktopQuote(name))
				}
				results <- result{name, err}
			}
		}()
	}
	go func() {
		defer close(tasks)
		for _, name := range names {
			tasks <- name
		}
	}()
	go func() { wg.Wait(); close(results) }()
	total := len(names) + 1
	completed := 0
	if progress != nil {
		progress(0, total, "Reading metadata")
	}
	var errs []error
	for r := range results {
		completed++
		if r.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.name, r.err))
		}
		if progress != nil {
			progress(completed, total, r.name)
		}
	}
	if len(errs) == 0 {
		if err := pruneEntries(filepath.Join(a.configDir, "supported"), names); err != nil {
			errs = append(errs, err)
		}
	}
	if err := a.renewReady(); err != nil {
		errs = append(errs, err)
	}
	if progress != nil {
		progress(total, total, "Ready VMs refreshed")
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func (a *DistroApp) download(name, arch, release, edition string, log func([]byte)) error {
	available, err := a.query(name, arch, "")
	if err != nil {
		return err
	}
	if !contains(available.Architectures, arch) || !contains(available.Releases, release) {
		return errors.New("this architecture/release combination is no longer supported")
	}
	m, err := a.query(name, arch, release)
	if err != nil {
		return err
	}
	if (len(m.Editions) == 0 && edition != "") || (len(m.Editions) > 0 && !contains(m.Editions, edition)) {
		return errors.New("this architecture/release/edition combination is no longer supported")
	}
	if err := os.MkdirAll(a.cfg.VMs, 0755); err != nil {
		return err
	}
	args := []string{"--arch", arch, name, release}
	if edition != "" {
		args = append(args, edition)
	}
	c := a.command(a.tool("qget"), args...)
	c.Dir = a.cfg.VMs
	w := logWriter{log}
	c.Stdout = w
	c.Stderr = w
	return c.Run()
}

type logWriter struct{ write func([]byte) }

func (w logWriter) Write(p []byte) (int, error) {
	if w.write != nil {
		w.write(append([]byte(nil), p...))
	}
	return len(p), nil
}

func (a *DistroApp) launch(name string) error {
	if !validName(name) {
		return errors.New("invalid VM name")
	}
	conf, err := filepath.Abs(filepath.Join(a.cfg.VMs, name+".conf"))
	if err != nil {
		return err
	}
	qemu := a.tool("quickemu")
	if qemu == "quickemu" {
		qemu = quickemuBin()
	}
	c := a.command(qemu, "--vm", conf)
	c.Dir = a.cfg.VMs
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".dh-entry-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func copyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return atomicWrite(dst, data, mode)
}

// Query returns metadata for the given OS/arch/release.
func (a *DistroApp) Query(osID, arch, release string) (Metadata, error) {
	return a.query(osID, arch, release)
}

// Download downloads the chosen OS/arch/release/edition.
func (a *DistroApp) Download(name, arch, release, edition string, log func([]byte)) error {
	return a.download(name, arch, release, edition, log)
}

// Context returns the app context.
func (a *DistroApp) Context() context.Context {
	return a.ctx
}

// Ready returns ready VM names.
func (a *DistroApp) Ready() ([]string, error) {
	return a.ready()
}

// Systems returns supported operating systems.
func (a *DistroApp) Systems() ([]OperatingSystem, error) {
	return a.systems()
}

// IconFor resolves an icon path for the given name.
func (a *DistroApp) IconFor(name string) string {
	return a.iconFor(name)
}

// Launch launches the named VM.
func (a *DistroApp) Launch(name string) error {
	return a.launch(name)
}

// RenewReady refreshes ready VM entries.
func (a *DistroApp) RenewReady() error {
	return a.renewReady()
}

// RenewSupported regenerates supported VM entries.
func (a *DistroApp) RenewSupported(progress func(int, int, string)) error {
	return a.renewSupported(progress)
}
