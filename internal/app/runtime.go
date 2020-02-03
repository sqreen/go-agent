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
	"github.com/sqreen/go-libsqreen/waf"
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

func (p *ProcessInfo) GetName() string {
	return p.name
}

func (p *ProcessInfo) GetTime() time.Time {
	return p.time
}

func (p *ProcessInfo) GetPid() uint32 {
	return p.pid
}

func (p *ProcessInfo) GetPpid() uint32 {
	return p.ppid
}

func (p *ProcessInfo) GetEuid() uint32 {
	return p.euid
}

func (p *ProcessInfo) GetEgid() uint32 {
	return p.egid
}

func (p *ProcessInfo) GetUid() uint32 {
	return p.uid
}

func (p *ProcessInfo) GetGid() uint32 {
	return p.gid
}

func (p *ProcessInfo) GetLibSqreenVersion() *string {
	return waf.Version()
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
		return nil, "", sqerrors.New("could not read the build information")
	}

	i.dependencies = info.Deps
	i.signature = bundleSignature(info.Deps)
	return i.dependencies, i.signature, nil
}

func bundleSignature(deps []*debug.Module) string {
	set := make([]string, len(deps))
	for i, dep := range deps {
		set[i] = fmt.Sprintf("%s-%s", dep.Path, dep.Version)
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
	pkg := reflect.TypeOf(t{}).PkgPath()
	vendor := "vendor/"
	i := strings.Index(pkg, vendor)
	if i == -1 {
		return ""
	}
	return pkg[:i+len(vendor)]
}
