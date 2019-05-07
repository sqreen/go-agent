// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package app

import (
	"debug/elf"
	"debug/gosym"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/sqreen/go-agent/agent/internal/plog"
)

type Info struct {
	logger      *plog.Logger
	hostname    string
	processInfo ProcessInfo
}

func NewInfo(logger *plog.Logger) *Info {
	logger = plog.NewLogger("app/info", logger)
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

type Dependency struct {
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

func (i *Info) Dependencies() ([]*Dependency, error) {
	executable := i.processInfo.GetName()

	i.logger.Debug("reading ELF file ", executable)
	exe, err := elf.Open(executable)
	if err != nil {
		return nil, err
	}
	defer exe.Close()

	var pclndat []byte
	if sec := exe.Section(".gopclntab"); sec != nil {
		pclndat, err = sec.Data()
		if err != nil {
			i.logger.Error("cannot read .gopclntab section: ", err)
			return nil, err
		}
	}

	sec := exe.Section(".gosymtab")
	symTabRaw, err := sec.Data()
	if err != nil {
		return nil, err
	}
	pcln := gosym.NewLineTable(pclndat, exe.Section(".text").Addr)
	symTab, err := gosym.NewTable(symTabRaw, pcln)
	if err != nil {
		i.logger.Error("cannot create the Go synbol table: ", err)
		return nil, err
	}

	dependencies := make(map[string]struct{})

	for _, f := range symTab.Funcs {
		if strings.HasPrefix(f.Name, "type.") ||
			strings.HasPrefix(f.Name, "go.") ||
			strings.Index(f.Name, "(") != -1 ||
			strings.Contains(f.Name, "cgo_") {
			continue
		}

		pkg := f.PackageName()
		if i := strings.LastIndex(pkg, "vendor/"); i != -1 {
			pkg = pkg[i+len("vendor/"):]
		}

		if _, exists := dependencies[pkg]; exists {
			continue
		}
		fmt.Println(pkg)
		dependencies[pkg] = struct{}{}
	}

	importedLibs, err := exe.ImportedLibraries()
	if err != nil {
		return nil, err
	}
	fmt.Println(importedLibs)
	//
	//	for _, symbol := range importedSymbols {
	//
	//	}

	return nil, nil
}

func packageName(symbol string) string {

	pathend := strings.LastIndex(symbol, "/")
	if pathend < 0 {
		pathend = 0
	}

	i := strings.Index(symbol[pathend:], ".")
	if i == -1 {
		return ""
	}

	return symbol[:pathend+i]
}

func executable(logger *plog.Logger) string {
	name, err := os.Executable()
	if err != nil {
		logger.Error("could not read the executable name ", err)
		return ""
	}
	return name
}
