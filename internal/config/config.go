// Package config reads reqbox.yaml, an optional file beside the binary that
// supplies defaults for the flags.
//
// Why a file at all: a double-clicked binary has no convenient way to pass
// flags. On Windows that is the normal way to start Reqbox, and the only
// alternatives were editing a shortcut or setting an environment variable
// system-wide. A file next to the executable is something the operator can find
// and edit.
//
// Precedence is deliberately boring and matches what most tools do:
//
//	command-line flag  >  environment variable  >  config file  >  built-in default
//
// A flag beats the file because someone typing it is making a more specific
// request than a file they wrote once. The file beats nothing else: it is a
// fallback, never an override, so an existing deployment that passes -addr
// keeps working untouched after someone drops the file in.
package config

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileName is the name looked for beside the binary.
const FileName = "reqbox.yaml"

// File is the on-disk shape.
//
// Field names are short and mirrored by the flags, so the mapping is obvious
// without documentation: `port` here is `-addr`'s port, and so on.
type File struct {
	// Server holds the network settings.
	Server struct {
		// Addr is the full listen address, e.g. "127.0.0.1:5178".
		Addr string `yaml:"addr"`
		// Port is a shorthand for the port part of Addr, so a file that only
		// wants to move the port does not have to repeat the host.
		Port int `yaml:"port"`
		// AutoPort lets the server move to the next free port when the
		// requested one is taken. Pointer so "not set" (default true) is
		// distinguishable from an explicit false.
		AutoPort *bool `yaml:"autoPort"`
		// OpenBrowser controls whether a browser is launched at startup.
		OpenBrowser *bool `yaml:"openBrowser"`
	} `yaml:"server"`

	// Storage holds the database location.
	Storage struct {
		DB string `yaml:"db"`
	} `yaml:"storage"`

	// Auth controls the token check. Auth.Token is the operator-supplied
	// token; leaving it empty keeps the generated one stored in the database.
	Auth struct {
		Enabled *bool  `yaml:"enabled"`
		Token   string `yaml:"token"`
	} `yaml:"auth"`

	// Logging holds verbosity.
	Logging struct {
		Verbose bool `yaml:"verbose"`
	} `yaml:"logging"`
}

// DefaultAddr is the built-in listen address, used when neither the file nor a
// flag says anything.
const DefaultAddr = "127.0.0.1:5178"

// Default returns the values used when nothing else says otherwise.
//
// Server.Addr is deliberately left empty rather than pre-filled with
// DefaultAddr: filling it would make a file that sets only `port` look like it
// set both, and validate() would reject the ordinary case of "move the port,
// keep the host".
func Default() File {
	var f File
	autoPort := true
	f.Server.AutoPort = &autoPort
	openBrowser := true
	f.Server.OpenBrowser = &openBrowser
	enabled := true
	f.Auth.Enabled = &enabled
	return f
}

// Load reads the file beside the binary, if one exists.
//
// A missing file is not an error: the file is optional by design, and most
// installs will never have one. It returns the path it considered even when it
// found nothing, so the caller can log it — "I looked here" is the single most
// useful line when someone's config "is not being picked up".
func Load() (File, string, error) {
	return loadFile(Path())
}

// loadFile reads a specific path. Split from Load so tests can point at a temp
// directory instead of the folder holding the test binary.
func loadFile(path string) (File, string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), path, nil
	}
	if err != nil {
		return Default(), path, fmt.Errorf("read %s: %w", path, err)
	}

	f := Default()
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	// Reject unknown keys: a typo like `auto_port:` would otherwise be silently
	// ignored, and the user would be left believing the setting took effect.
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		// An empty file decodes to io.EOF; that is "all defaults", not a typo.
		if !errors.Is(err, io.EOF) {
			return Default(), path, fmt.Errorf("parse %s: %w", path, err)
		}
	}
	if err := f.validate(); err != nil {
		return Default(), path, fmt.Errorf("%s: %w", path, err)
	}
	return f, path, nil
}

// Path is the location Load reads: beside the binary, so a folder copied to a
// server carries its settings with it.
func Path() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), FileName)
	}
	return FileName
}

func (f File) validate() error {
	if f.Server.Port != 0 && (f.Server.Port < 1 || f.Server.Port > 65535) {
		return fmt.Errorf("server.port %d is out of range 1-65535", f.Server.Port)
	}
	if f.Server.Addr != "" && f.Server.Port != 0 {
		// Both spellings set: one of them is going to be ignored, and guessing
		// which would be the kind of silent surprise this file is meant to
		// avoid. Say so instead.
		if _, port, err := splitHostPort(f.Server.Addr); err == nil && port != "" {
			return errors.New("server.addr and server.port are both set; use one or the other")
		}
	}
	return nil
}

// Addr resolves the final listen address from the file and a fallback.
//
// The file may express the address three ways, in order of specificity:
// server.addr, server.port, or neither (in which case fallback wins).
func (f File) Addr(fallback string) string {
	if f.Server.Addr != "" {
		return f.Server.Addr
	}
	if f.Server.Port != 0 {
		host := "127.0.0.1"
		if h, _, err := splitHostPort(fallback); err == nil && h != "" {
			host = h
		}
		return host + ":" + strconv.Itoa(f.Server.Port)
	}
	return fallback
}

// AutoPortEnabled reports whether the server may move to another port.
func (f File) AutoPortEnabled() bool {
	return f.Server.AutoPort == nil || *f.Server.AutoPort
}

// OpenBrowserEnabled reports whether a browser should be launched.
func (f File) OpenBrowserEnabled() bool {
	return f.Server.OpenBrowser == nil || *f.Server.OpenBrowser
}

// AuthEnabled reports whether the token check is on.
func (f File) AuthEnabled() bool {
	return f.Auth.Enabled == nil || *f.Auth.Enabled
}

// splitHostPort splits "host:port". A thin wrapper over net.SplitHostPort so
// this package and main.go agree on what a malformed address is, including the
// bracketed IPv6 form.
func splitHostPort(addr string) (host, port string, err error) {
	return net.SplitHostPort(addr)
}
