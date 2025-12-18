// Package main provides tests for the npm plugin.
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

func TestGetInfo(t *testing.T) {
	p := &NpmPlugin{}
	info := p.GetInfo()

	t.Run("name", func(t *testing.T) {
		if info.Name != "npm" {
			t.Errorf("expected name 'npm', got %q", info.Name)
		}
	})

	t.Run("version", func(t *testing.T) {
		if info.Version != "2.0.0" {
			t.Errorf("expected version '2.0.0', got %q", info.Version)
		}
	})

	t.Run("description", func(t *testing.T) {
		if info.Description != "Publish packages to npm registry" {
			t.Errorf("unexpected description: %q", info.Description)
		}
	})

	t.Run("author", func(t *testing.T) {
		if info.Author != "Relicta Team" {
			t.Errorf("expected author 'Relicta Team', got %q", info.Author)
		}
	})

	t.Run("hooks", func(t *testing.T) {
		expectedHooks := []plugin.Hook{plugin.HookPrePublish, plugin.HookPostPublish}
		if len(info.Hooks) != len(expectedHooks) {
			t.Errorf("expected %d hooks, got %d", len(expectedHooks), len(info.Hooks))
			return
		}
		for i, hook := range expectedHooks {
			if info.Hooks[i] != hook {
				t.Errorf("expected hook %q at index %d, got %q", hook, i, info.Hooks[i])
			}
		}
	})

	t.Run("config_schema_valid_json", func(t *testing.T) {
		var schema map[string]any
		if err := json.Unmarshal([]byte(info.ConfigSchema), &schema); err != nil {
			t.Errorf("config schema is not valid JSON: %v", err)
		}
		if schema["type"] != "object" {
			t.Errorf("expected schema type 'object', got %v", schema["type"])
		}
	})
}

func TestValidate(t *testing.T) {
	p := &NpmPlugin{}
	ctx := context.Background()

	tests := []struct {
		name       string
		config     map[string]any
		wantValid  bool
		wantErrors []string
	}{
		{
			name:      "empty_config_valid",
			config:    map[string]any{},
			wantValid: true,
		},
		{
			name: "valid_access_public",
			config: map[string]any{
				"access": "public",
			},
			wantValid: true,
		},
		{
			name: "valid_access_restricted",
			config: map[string]any{
				"access": "restricted",
			},
			wantValid: true,
		},
		{
			name: "invalid_access_level",
			config: map[string]any{
				"access": "private",
			},
			wantValid:  false,
			wantErrors: []string{"access"},
		},
		{
			name: "valid_full_config",
			config: map[string]any{
				"registry":       "https://registry.npmjs.org",
				"tag":            "latest",
				"access":         "public",
				"dry_run":        true,
				"update_version": true,
			},
			wantValid: true,
		},
		{
			name: "package_dir_not_found",
			config: map[string]any{
				"package_dir": "/nonexistent/path/to/package",
			},
			wantValid:  false,
			wantErrors: []string{"package_dir"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := p.Validate(ctx, tt.config)
			if err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}

			if resp.Valid != tt.wantValid {
				t.Errorf("Validate().Valid = %v, want %v", resp.Valid, tt.wantValid)
			}

			if !tt.wantValid && len(tt.wantErrors) > 0 {
				for _, wantField := range tt.wantErrors {
					found := false
					for _, e := range resp.Errors {
						if e.Field == wantField {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error for field %q, but not found in errors: %v", wantField, resp.Errors)
					}
				}
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	p := &NpmPlugin{}

	tests := []struct {
		name   string
		raw    map[string]any
		expect Config
	}{
		{
			name: "defaults",
			raw:  map[string]any{},
			expect: Config{
				Tag:           "latest",
				UpdateVersion: true,
			},
		},
		{
			name: "all_fields",
			raw: map[string]any{
				"registry":       "https://custom.registry.com",
				"tag":            "beta",
				"access":         "restricted",
				"otp":            "123456",
				"dry_run":        true,
				"package_dir":    "./packages/core",
				"update_version": false,
			},
			expect: Config{
				Registry:      "https://custom.registry.com",
				Tag:           "beta",
				Access:        "restricted",
				OTP:           "123456",
				DryRun:        true,
				PackageDir:    "./packages/core",
				UpdateVersion: false,
			},
		},
		{
			name: "empty_tag_defaults_to_latest",
			raw: map[string]any{
				"tag": "",
			},
			expect: Config{
				Tag:           "latest",
				UpdateVersion: true,
			},
		},
		{
			name: "partial_config",
			raw: map[string]any{
				"registry": "https://npm.example.com",
				"dry_run":  true,
			},
			expect: Config{
				Registry:      "https://npm.example.com",
				Tag:           "latest",
				DryRun:        true,
				UpdateVersion: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.parseConfig(tt.raw)

			if got.Registry != tt.expect.Registry {
				t.Errorf("Registry = %q, want %q", got.Registry, tt.expect.Registry)
			}
			if got.Tag != tt.expect.Tag {
				t.Errorf("Tag = %q, want %q", got.Tag, tt.expect.Tag)
			}
			if got.Access != tt.expect.Access {
				t.Errorf("Access = %q, want %q", got.Access, tt.expect.Access)
			}
			if got.OTP != tt.expect.OTP {
				t.Errorf("OTP = %q, want %q", got.OTP, tt.expect.OTP)
			}
			if got.DryRun != tt.expect.DryRun {
				t.Errorf("DryRun = %v, want %v", got.DryRun, tt.expect.DryRun)
			}
			if got.PackageDir != tt.expect.PackageDir {
				t.Errorf("PackageDir = %q, want %q", got.PackageDir, tt.expect.PackageDir)
			}
			if got.UpdateVersion != tt.expect.UpdateVersion {
				t.Errorf("UpdateVersion = %v, want %v", got.UpdateVersion, tt.expect.UpdateVersion)
			}
		})
	}
}

func TestExecute(t *testing.T) {
	p := &NpmPlugin{}
	ctx := context.Background()

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "npm-plugin-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a valid package.json in the temp directory
	packageJSON := map[string]any{
		"name":    "test-package",
		"version": "1.0.0",
		"private": false,
	}
	packageData, _ := json.MarshalIndent(packageJSON, "", "  ")
	packagePath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(packagePath, packageData, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Change to the temp directory for tests that need package_dir validation
	origWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()

	releaseCtx := plugin.ReleaseContext{
		Version:         "1.2.3",
		PreviousVersion: "1.2.2",
		TagName:         "v1.2.3",
		ReleaseType:     "patch",
		Branch:          "main",
		CommitSHA:       "abc123",
	}

	t.Run("pre_publish_update_version_dry_run", func(t *testing.T) {
		req := plugin.ExecuteRequest{
			Hook: plugin.HookPrePublish,
			Config: map[string]any{
				"update_version": true,
				"package_dir":    ".",
			},
			Context: releaseCtx,
			DryRun:  true,
		}

		resp, err := p.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}

		if !resp.Success {
			t.Errorf("Execute failed: %s", resp.Error)
		}
		if resp.Message == "" {
			t.Error("Expected message in response")
		}
	})

	t.Run("pre_publish_update_version_disabled", func(t *testing.T) {
		req := plugin.ExecuteRequest{
			Hook: plugin.HookPrePublish,
			Config: map[string]any{
				"update_version": false,
			},
			Context: releaseCtx,
			DryRun:  true,
		}

		resp, err := p.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}

		if !resp.Success {
			t.Errorf("Execute failed: %s", resp.Error)
		}
		if resp.Message != "Version update disabled" {
			t.Errorf("unexpected message: %q", resp.Message)
		}
	})

	t.Run("post_publish_dry_run", func(t *testing.T) {
		req := plugin.ExecuteRequest{
			Hook: plugin.HookPostPublish,
			Config: map[string]any{
				"package_dir": ".",
				"tag":         "latest",
				"access":      "public",
			},
			Context: releaseCtx,
			DryRun:  true,
		}

		resp, err := p.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}

		if !resp.Success {
			t.Errorf("Execute failed: %s", resp.Error)
		}

		// Check outputs
		if resp.Outputs == nil {
			t.Error("Expected outputs in response")
		} else {
			if resp.Outputs["package"] != "test-package" {
				t.Errorf("expected package 'test-package', got %v", resp.Outputs["package"])
			}
			if resp.Outputs["version"] != "1.2.3" {
				t.Errorf("expected version '1.2.3', got %v", resp.Outputs["version"])
			}
		}
	})

	t.Run("post_publish_config_dry_run", func(t *testing.T) {
		req := plugin.ExecuteRequest{
			Hook: plugin.HookPostPublish,
			Config: map[string]any{
				"package_dir": ".",
				"dry_run":     true, // Config-level dry run
			},
			Context: releaseCtx,
			DryRun:  false, // Request-level not dry run
		}

		resp, err := p.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}

		if !resp.Success {
			t.Errorf("Execute failed: %s", resp.Error)
		}
	})

	t.Run("post_publish_private_package", func(t *testing.T) {
		// Create a private package.json
		privateDir := filepath.Join(tmpDir, "private-pkg")
		if err := os.Mkdir(privateDir, 0755); err != nil {
			t.Fatalf("failed to create private pkg dir: %v", err)
		}
		privatePkg := map[string]any{
			"name":    "private-package",
			"version": "1.0.0",
			"private": true,
		}
		privatePkgData, _ := json.MarshalIndent(privatePkg, "", "  ")
		if err := os.WriteFile(filepath.Join(privateDir, "package.json"), privatePkgData, 0644); err != nil {
			t.Fatalf("failed to write private package.json: %v", err)
		}

		req := plugin.ExecuteRequest{
			Hook: plugin.HookPostPublish,
			Config: map[string]any{
				"package_dir": "private-pkg",
			},
			Context: releaseCtx,
			DryRun:  true,
		}

		resp, err := p.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}

		if !resp.Success {
			t.Errorf("Execute failed: %s", resp.Error)
		}
		if resp.Message != "Package is private, skipping npm publish" {
			t.Errorf("unexpected message: %q", resp.Message)
		}
	})

	t.Run("unhandled_hook", func(t *testing.T) {
		req := plugin.ExecuteRequest{
			Hook:    plugin.HookPostNotes,
			Config:  map[string]any{},
			Context: releaseCtx,
			DryRun:  true,
		}

		resp, err := p.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}

		if !resp.Success {
			t.Errorf("Execute failed: %s", resp.Error)
		}
		if resp.Message != "Hook post-notes not handled" {
			t.Errorf("unexpected message: %q", resp.Message)
		}
	})

	t.Run("post_publish_missing_package_json", func(t *testing.T) {
		// Create empty directory without package.json
		emptyDir := filepath.Join(tmpDir, "empty-pkg")
		if err := os.Mkdir(emptyDir, 0755); err != nil {
			t.Fatalf("failed to create empty pkg dir: %v", err)
		}

		req := plugin.ExecuteRequest{
			Hook: plugin.HookPostPublish,
			Config: map[string]any{
				"package_dir": "empty-pkg",
			},
			Context: releaseCtx,
			DryRun:  true,
		}

		resp, err := p.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}

		if resp.Success {
			t.Error("Execute should have failed for missing package.json")
		}
		if resp.Error == "" {
			t.Error("Expected error message")
		}
	})
}

func TestValidateRegistry(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		wantErr  bool
	}{
		{"empty_valid", "", false},
		{"https_valid", "https://registry.npmjs.org", false},
		{"https_custom", "https://npm.example.com/", false},
		{"http_localhost", "http://localhost:4873", false},
		{"http_127_0_0_1", "http://127.0.0.1:4873", false},
		{"http_external_invalid", "http://registry.npmjs.org", true},
		{"invalid_url", "not-a-url", true},
		{"newline_injection", "https://example.com\n--otp=123456", true},
		{"carriage_return_injection", "https://example.com\r--otp=123456", true},
		{"tab_injection", "https://example.com\t--otp=123456", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegistry(tt.registry)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegistry(%q) error = %v, wantErr %v", tt.registry, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTag(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		wantErr bool
	}{
		{"empty_valid", "", false},
		{"latest", "latest", false},
		{"beta", "beta", false},
		{"alpha_1", "alpha.1", false},
		{"next_major", "next-major", false},
		{"underscore", "my_tag", false},
		{"numeric_prefix", "1.0.0", false},
		{"too_long", string(make([]byte, 129)), true},
		{"special_chars", "tag@123", true},
		{"spaces", "my tag", true},
		{"starts_with_dot", ".hidden", true},
		{"starts_with_hyphen", "-invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTag(tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTag(%q) error = %v, wantErr %v", tt.tag, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAccess(t *testing.T) {
	tests := []struct {
		name    string
		access  string
		wantErr bool
	}{
		{"empty_valid", "", false},
		{"public", "public", false},
		{"restricted", "restricted", false},
		{"private_invalid", "private", true},
		{"unknown_invalid", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAccess(tt.access)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAccess(%q) error = %v, wantErr %v", tt.access, err, tt.wantErr)
			}
		})
	}
}

func TestValidateOTP(t *testing.T) {
	tests := []struct {
		name    string
		otp     string
		wantErr bool
	}{
		{"empty_valid", "", false},
		{"six_digits", "123456", false},
		{"seven_digits", "1234567", false},
		{"eight_digits", "12345678", false},
		{"five_digits_invalid", "12345", true},
		{"nine_digits_invalid", "123456789", true},
		{"letters_invalid", "abcdef", true},
		{"mixed_invalid", "123abc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOTP(tt.otp)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOTP(%q) error = %v, wantErr %v", tt.otp, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePackageDir(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "npm-plugin-dir-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Create a file (not a directory)
	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Change to the temp directory
	origWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()

	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{"empty_uses_cwd", "", false},
		{"current_dir", ".", false},
		{"valid_subdir", "subdir", false},
		{"path_traversal_blocked", "../..", true},
		{"path_with_traversal", "subdir/../../../etc", true},
		{"file_not_dir", "file.txt", true},
		{"nonexistent", "nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validatePackageDir(tt.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePackageDir(%q) error = %v, wantErr %v", tt.dir, err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	p := &NpmPlugin{}

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "empty_config_valid",
			config:  Config{},
			wantErr: false,
		},
		{
			name: "full_valid_config",
			config: Config{
				Registry: "https://registry.npmjs.org",
				Tag:      "latest",
				Access:   "public",
				OTP:      "123456",
			},
			wantErr: false,
		},
		{
			name: "invalid_registry",
			config: Config{
				Registry: "http://external.registry.com",
			},
			wantErr: true,
		},
		{
			name: "invalid_tag",
			config: Config{
				Tag: "invalid@tag",
			},
			wantErr: true,
		},
		{
			name: "invalid_access",
			config: Config{
				Access: "private",
			},
			wantErr: true,
		},
		{
			name: "invalid_otp",
			config: Config{
				OTP: "abc",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.validateConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdatePackageVersion(t *testing.T) {
	p := &NpmPlugin{}
	ctx := context.Background()

	t.Run("dry_run_shows_would_update", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "npm-version-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		packageJSON := map[string]any{
			"name":    "test-package",
			"version": "1.0.0",
		}
		packageData, _ := json.MarshalIndent(packageJSON, "", "  ")
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), packageData, 0644); err != nil {
			t.Fatalf("failed to write package.json: %v", err)
		}

		// Change to temp dir for validation
		origWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer func() { _ = os.Chdir(origWd) }()

		cfg := &Config{PackageDir: "."}
		releaseCtx := plugin.ReleaseContext{Version: "2.0.0"}

		resp, err := p.updatePackageVersion(ctx, cfg, releaseCtx, true)
		if err != nil {
			t.Fatalf("updatePackageVersion returned error: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success, got error: %s", resp.Error)
		}

		// Verify package.json was NOT modified (dry run)
		data, err := os.ReadFile(filepath.Join(tmpDir, "package.json"))
		if err != nil {
			t.Fatalf("failed to read package.json: %v", err)
		}
		var pkg map[string]any
		if err := json.Unmarshal(data, &pkg); err != nil {
			t.Fatalf("failed to unmarshal package.json: %v", err)
		}
		if pkg["version"] != "1.0.0" {
			t.Errorf("package.json was modified during dry run")
		}
	})

	t.Run("actual_update_modifies_file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "npm-version-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		packageJSON := map[string]any{
			"name":    "test-package",
			"version": "1.0.0",
		}
		packageData, _ := json.MarshalIndent(packageJSON, "", "  ")
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), packageData, 0644); err != nil {
			t.Fatalf("failed to write package.json: %v", err)
		}

		// Change to temp dir for validation
		origWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer func() { _ = os.Chdir(origWd) }()

		cfg := &Config{PackageDir: "."}
		releaseCtx := plugin.ReleaseContext{Version: "2.0.0"}

		resp, err := p.updatePackageVersion(ctx, cfg, releaseCtx, false)
		if err != nil {
			t.Fatalf("updatePackageVersion returned error: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success, got error: %s", resp.Error)
		}

		// Verify package.json WAS modified
		data, err := os.ReadFile(filepath.Join(tmpDir, "package.json"))
		if err != nil {
			t.Fatalf("failed to read package.json: %v", err)
		}
		var pkg map[string]any
		if err := json.Unmarshal(data, &pkg); err != nil {
			t.Fatalf("failed to unmarshal package.json: %v", err)
		}
		if pkg["version"] != "2.0.0" {
			t.Errorf("expected version '2.0.0', got %v", pkg["version"])
		}

		// Check outputs
		if resp.Outputs == nil {
			t.Error("expected outputs")
		} else {
			if resp.Outputs["old_version"] != "1.0.0" {
				t.Errorf("expected old_version '1.0.0', got %v", resp.Outputs["old_version"])
			}
			if resp.Outputs["new_version"] != "2.0.0" {
				t.Errorf("expected new_version '2.0.0', got %v", resp.Outputs["new_version"])
			}
		}
	})

	t.Run("missing_package_json", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "npm-version-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		// Change to temp dir for validation
		origWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer func() { _ = os.Chdir(origWd) }()

		cfg := &Config{PackageDir: "."}
		releaseCtx := plugin.ReleaseContext{Version: "2.0.0"}

		resp, err := p.updatePackageVersion(ctx, cfg, releaseCtx, false)
		if err != nil {
			t.Fatalf("updatePackageVersion returned error: %v", err)
		}

		if resp.Success {
			t.Error("expected failure for missing package.json")
		}
	})

	t.Run("invalid_package_json", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "npm-version-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		// Write invalid JSON
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("not json"), 0644); err != nil {
			t.Fatalf("failed to write package.json: %v", err)
		}

		// Change to temp dir for validation
		origWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer func() { _ = os.Chdir(origWd) }()

		cfg := &Config{PackageDir: "."}
		releaseCtx := plugin.ReleaseContext{Version: "2.0.0"}

		resp, err := p.updatePackageVersion(ctx, cfg, releaseCtx, false)
		if err != nil {
			t.Fatalf("updatePackageVersion returned error: %v", err)
		}

		if resp.Success {
			t.Error("expected failure for invalid JSON")
		}
	})
}

func TestPublishPackage(t *testing.T) {
	p := &NpmPlugin{}
	ctx := context.Background()

	t.Run("dry_run_returns_command_info", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "npm-publish-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		packageJSON := map[string]any{
			"name":    "test-publish-package",
			"version": "1.0.0",
			"private": false,
		}
		packageData, _ := json.MarshalIndent(packageJSON, "", "  ")
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), packageData, 0644); err != nil {
			t.Fatalf("failed to write package.json: %v", err)
		}

		// Change to temp dir for validation
		origWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer func() { _ = os.Chdir(origWd) }()

		cfg := &Config{
			PackageDir: ".",
			Tag:        "beta",
			Access:     "public",
			Registry:   "https://registry.npmjs.org",
		}
		releaseCtx := plugin.ReleaseContext{Version: "1.0.0"}

		resp, err := p.publishPackage(ctx, cfg, releaseCtx, true)
		if err != nil {
			t.Fatalf("publishPackage returned error: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success, got error: %s", resp.Error)
		}

		// Check outputs contain expected information
		if resp.Outputs == nil {
			t.Error("expected outputs")
		} else {
			if resp.Outputs["package"] != "test-publish-package" {
				t.Errorf("expected package 'test-publish-package', got %v", resp.Outputs["package"])
			}
			if resp.Outputs["command"] == nil {
				t.Error("expected command in outputs")
			}
		}
	})

	t.Run("otp_redacted_in_dry_run_message", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "npm-publish-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		packageJSON := map[string]any{
			"name":    "test-otp-package",
			"version": "1.0.0",
			"private": false,
		}
		packageData, _ := json.MarshalIndent(packageJSON, "", "  ")
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), packageData, 0644); err != nil {
			t.Fatalf("failed to write package.json: %v", err)
		}

		// Change to temp dir for validation
		origWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer func() { _ = os.Chdir(origWd) }()

		cfg := &Config{
			PackageDir: ".",
			OTP:        "123456",
		}
		releaseCtx := plugin.ReleaseContext{Version: "1.0.0"}

		resp, err := p.publishPackage(ctx, cfg, releaseCtx, true)
		if err != nil {
			t.Fatalf("publishPackage returned error: %v", err)
		}

		// OTP should be redacted in the message
		if resp.Outputs != nil {
			cmd, ok := resp.Outputs["command"].(string)
			if ok && !contains(cmd, "[REDACTED]") {
				t.Error("OTP should be redacted in command output")
			}
		}
	})

	t.Run("private_package_skips_publish", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "npm-publish-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		packageJSON := map[string]any{
			"name":    "private-package",
			"version": "1.0.0",
			"private": true,
		}
		packageData, _ := json.MarshalIndent(packageJSON, "", "  ")
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), packageData, 0644); err != nil {
			t.Fatalf("failed to write package.json: %v", err)
		}

		// Change to temp dir for validation
		origWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer func() { _ = os.Chdir(origWd) }()

		cfg := &Config{PackageDir: "."}
		releaseCtx := plugin.ReleaseContext{Version: "1.0.0"}

		resp, err := p.publishPackage(ctx, cfg, releaseCtx, true)
		if err != nil {
			t.Fatalf("publishPackage returned error: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success, got error: %s", resp.Error)
		}
		if resp.Message != "Package is private, skipping npm publish" {
			t.Errorf("unexpected message: %q", resp.Message)
		}
	})

	t.Run("invalid_config_fails_validation", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "npm-publish-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		packageJSON := map[string]any{
			"name":    "test-package",
			"version": "1.0.0",
			"private": false,
		}
		packageData, _ := json.MarshalIndent(packageJSON, "", "  ")
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), packageData, 0644); err != nil {
			t.Fatalf("failed to write package.json: %v", err)
		}

		// Change to temp dir for validation
		origWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer func() { _ = os.Chdir(origWd) }()

		cfg := &Config{
			PackageDir: ".",
			Access:     "invalid-access", // Invalid access level
		}
		releaseCtx := plugin.ReleaseContext{Version: "1.0.0"}

		resp, err := p.publishPackage(ctx, cfg, releaseCtx, true)
		if err != nil {
			t.Fatalf("publishPackage returned error: %v", err)
		}

		if resp.Success {
			t.Error("expected failure for invalid access level")
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
