// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package app

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

type Info struct {
	logger       *plog.Logger
	hostname     string
	processInfo  ProcessInfo
	dependencies []*debug.Module
	signature    string
}

func NewInfo(logger *plog.Logger) *Info {
	return &Info{
		logger: logger,
		processInfo: ProcessInfo{
			name: executable(logger),
			time: time.Now(),
			pid:  uint32(os.Getpid()),
			ppid: uint32(os.Getppid()),
			euid: uint32(os.Geteuid()),
			egid: uint32(os.Getegid()),
			uid:  uint32(os.Getuid()),
			gid:  uint32(os.Getgid()),
		},
	}
}

type ProcessInfo struct {
	name                            string
	time                            time.Time
	pid, ppid, euid, egid, uid, gid uint32
}

func (i *Info) GetProcessInfo() *ProcessInfo {
	return &i.processInfo
}

func (p *ProcessInfo) Name() string {
	return p.name
}

func (p *ProcessInfo) StartTime() time.Time {
	return p.time
}

func (p *ProcessInfo) Pid() uint32 {
	return p.pid
}

func (p *ProcessInfo) Ppid() uint32 {
	return p.ppid
}

func (p *ProcessInfo) Euid() uint32 {
	return p.euid
}

func (p *ProcessInfo) Egid() uint32 {
	return p.egid
}

func (p *ProcessInfo) Uid() uint32 {
	return p.uid
}

func (p *ProcessInfo) Gid() uint32 {
	return p.gid
}

func GoVersion() string {
	return runtime.Version()
}

var goBuildTarget = runtime.GOARCH + "-" + runtime.GOOS

func GoBuildTarget() string {
	return goBuildTarget
}

func (a *Info) Hostname() string {
	if a.hostname == "" {
		var err error
		a.hostname, err = os.Hostname()
		if err != nil {
			a.logger.Error(err)
			a.hostname = ""
		}
	}
	return a.hostname
}

func (i *Info) Dependencies() (deps []*debug.Module, sig string, err error) {
	if i.dependencies != nil {
		return i.dependencies, i.signature, nil
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		// Return an error but also the signature of the empty dependency bundle
		return nil, bundleSignature(deps), sqerrors.New("could not read the go build information - only available with go modules")
	}

	i.dependencies = info.Deps
	i.signature = bundleSignature(info.Deps)
	return i.dependencies, i.signature, nil
}

func bundleSignature(deps []*debug.Module) string {
	set := make([]string, len(deps))
	var strBuilder strings.Builder
	for i, dep := range deps {
		strBuilder.Reset()
		strBuilder.Grow(len(dep.Path) + len(dep.Version) + 1)
		strBuilder.WriteString(dep.Path)
		strBuilder.WriteByte('-')
		strBuilder.WriteString(dep.Version)
		set[i] = strBuilder.String()
	}
	sort.Strings(set)
	str := strings.Join(set, "|")
	sum := sha1.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}

func executable(logger *plog.Logger) string {
	name, err := os.Executable()
	if err != nil {
		// Log it and continue without it
		logger.Error(errors.Wrap(err, "could not read the executable name"))
		return ""
	}
	return name
}

func VendorPrefix() string {
	type t struct{}
	pkgPath := reflect.TypeOf(t{}).PkgPath()
	return vendorPrefix(pkgPath)
}

func vendorPrefix(pkgPath string) (prefix string) {
	vendor := "/vendor/"
	i := strings.Index(pkgPath, vendor)
	if i == -1 {
		return ""
	}
	return pkgPath[:i+len(vendor)]
}
