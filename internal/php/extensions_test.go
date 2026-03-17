package php

import (
	"testing"

	"qoliber/magebox/internal/platform"
)

func TestResolvePackageName_Ubuntu(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux, LinuxDistro: platform.DistroDebian}
	mgr := NewExtensionManager(p)

	tests := []struct {
		ext      string
		version  string
		wantPkg  string
		wantPecl bool
	}{
		{"redis", "8.3", "php8.3-redis", false},
		{"xdebug", "8.2", "php8.2-xdebug", false},
		{"imagick", "8.4", "php8.4-imagick", false},
		{"pgsql", "8.3", "php8.3-pgsql", false},
		// Unknown extension falls back to convention
		{"foobar", "8.3", "php8.3-foobar", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext+"_"+tt.version, func(t *testing.T) {
			pkg, usePecl := mgr.ResolvePackageName(tt.ext, tt.version)
			if pkg != tt.wantPkg {
				t.Errorf("ResolvePackageName(%q, %q) pkg = %q, want %q", tt.ext, tt.version, pkg, tt.wantPkg)
			}
			if usePecl != tt.wantPecl {
				t.Errorf("ResolvePackageName(%q, %q) usePecl = %v, want %v", tt.ext, tt.version, usePecl, tt.wantPecl)
			}
		})
	}
}

func TestResolvePackageName_Fedora(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux, LinuxDistro: platform.DistroFedora}
	mgr := NewExtensionManager(p)

	tests := []struct {
		ext      string
		version  string
		wantPkg  string
		wantPecl bool
	}{
		{"redis", "8.3", "php83-php-pecl-redis", false},
		{"xdebug", "8.2", "php82-php-xdebug", false},
		{"imagick", "8.4", "php84-php-pecl-imagick-im7", false},
		{"mysql", "8.3", "php83-php-mysqlnd", false},
		// Unknown extension falls back to convention
		{"foobar", "8.3", "php83-php-foobar", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext+"_"+tt.version, func(t *testing.T) {
			pkg, usePecl := mgr.ResolvePackageName(tt.ext, tt.version)
			if pkg != tt.wantPkg {
				t.Errorf("ResolvePackageName(%q, %q) pkg = %q, want %q", tt.ext, tt.version, pkg, tt.wantPkg)
			}
			if usePecl != tt.wantPecl {
				t.Errorf("ResolvePackageName(%q, %q) usePecl = %v, want %v", tt.ext, tt.version, usePecl, tt.wantPecl)
			}
		})
	}
}

func TestResolvePackageName_Darwin(t *testing.T) {
	p := &platform.Platform{Type: platform.Darwin, IsAppleSilicon: true}
	mgr := NewExtensionManager(p)

	tests := []struct {
		ext      string
		version  string
		wantPkg  string
		wantPecl bool
	}{
		{"redis", "8.3", "redis", true},
		{"xdebug", "8.2", "xdebug", true},
		// Bundled extensions don't use pecl
		{"gd", "8.3", "gd", false},
		{"intl", "8.3", "intl", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext+"_"+tt.version, func(t *testing.T) {
			pkg, usePecl := mgr.ResolvePackageName(tt.ext, tt.version)
			if pkg != tt.wantPkg {
				t.Errorf("ResolvePackageName(%q, %q) pkg = %q, want %q", tt.ext, tt.version, pkg, tt.wantPkg)
			}
			if usePecl != tt.wantPecl {
				t.Errorf("ResolvePackageName(%q, %q) usePecl = %v, want %v", tt.ext, tt.version, usePecl, tt.wantPecl)
			}
		})
	}
}

func TestResolvePackageName_Arch(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux, LinuxDistro: platform.DistroArch}
	mgr := NewExtensionManager(p)

	tests := []struct {
		ext      string
		version  string
		wantPkg  string
		wantPecl bool
	}{
		{"redis", "8.3", "php-redis", false},
		{"imagick", "8.3", "php-imagick", false},
		// No arch package, falls back to pecl
		{"mongodb", "8.3", "mongodb", true},
		// Unknown extension falls back to convention
		{"foobar", "8.3", "php-foobar", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext+"_"+tt.version, func(t *testing.T) {
			pkg, usePecl := mgr.ResolvePackageName(tt.ext, tt.version)
			if pkg != tt.wantPkg {
				t.Errorf("ResolvePackageName(%q, %q) pkg = %q, want %q", tt.ext, tt.version, pkg, tt.wantPkg)
			}
			if usePecl != tt.wantPecl {
				t.Errorf("ResolvePackageName(%q, %q) usePecl = %v, want %v", tt.ext, tt.version, usePecl, tt.wantPecl)
			}
		})
	}
}

func TestInstallCommand(t *testing.T) {
	tests := []struct {
		name     string
		platform *platform.Platform
		ext      string
		version  string
		want     string
	}{
		{
			name:     "ubuntu redis",
			platform: &platform.Platform{Type: platform.Linux, LinuxDistro: platform.DistroDebian},
			ext:      "redis",
			version:  "8.3",
			want:     "sudo apt install -y php8.3-redis",
		},
		{
			name:     "fedora redis",
			platform: &platform.Platform{Type: platform.Linux, LinuxDistro: platform.DistroFedora},
			ext:      "redis",
			version:  "8.3",
			want:     "sudo dnf install -y php83-php-pecl-redis",
		},
		{
			name:     "arch redis",
			platform: &platform.Platform{Type: platform.Linux, LinuxDistro: platform.DistroArch},
			ext:      "redis",
			version:  "8.3",
			want:     "sudo pacman -S --noconfirm php-redis",
		},
		{
			name:     "darwin redis",
			platform: &platform.Platform{Type: platform.Darwin, IsAppleSilicon: true},
			ext:      "redis",
			version:  "8.3",
			want:     "/opt/homebrew/opt/php@8.3/bin/pecl install redis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewExtensionManager(tt.platform)
			got := mgr.InstallCommand(tt.ext, tt.version)
			if got != tt.want {
				t.Errorf("InstallCommand(%q, %q) = %q, want %q", tt.ext, tt.version, got, tt.want)
			}
		})
	}
}

func TestIsPIEPackage(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"redis", false},
		{"xdebug", false},
		{"noisebynorthwest/php-spx", true},
		{"vendor/package", true},
		{"some-ext", false},
		{"org/ext-name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPIEPackage(tt.name)
			if got != tt.want {
				t.Errorf("IsPIEPackage(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestPIEInstallCommand(t *testing.T) {
	tests := []struct {
		name     string
		platform *platform.Platform
		pkg      string
		version  string
		want     string
	}{
		{
			name:     "ubuntu pie",
			platform: &platform.Platform{Type: platform.Linux, LinuxDistro: platform.DistroDebian},
			pkg:      "noisebynorthwest/php-spx",
			version:  "8.3",
			want:     "sudo pie install --with-php-config=/usr/bin/php-config8.3 noisebynorthwest/php-spx",
		},
		{
			name:     "fedora pie",
			platform: &platform.Platform{Type: platform.Linux, LinuxDistro: platform.DistroFedora},
			pkg:      "noisebynorthwest/php-spx",
			version:  "8.3",
			want:     "sudo pie install --with-php-config=/opt/remi/php83/root/usr/bin/php-config noisebynorthwest/php-spx",
		},
		{
			name:     "darwin pie",
			platform: &platform.Platform{Type: platform.Darwin, IsAppleSilicon: true},
			pkg:      "noisebynorthwest/php-spx",
			version:  "8.3",
			want:     "sudo pie install --with-php-config=/opt/homebrew/opt/php@8.3/bin/php-config noisebynorthwest/php-spx",
		},
		{
			name:     "arch pie",
			platform: &platform.Platform{Type: platform.Linux, LinuxDistro: platform.DistroArch},
			pkg:      "noisebynorthwest/php-spx",
			version:  "8.3",
			want:     "sudo pie install --with-php-config=/usr/bin/php-config noisebynorthwest/php-spx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewExtensionManager(tt.platform)
			got := mgr.PIEInstallCommand(tt.pkg, tt.version)
			if got != tt.want {
				t.Errorf("PIEInstallCommand(%q, %q) = %q, want %q", tt.pkg, tt.version, got, tt.want)
			}
		})
	}
}

func TestPieExtensionName(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux, LinuxDistro: platform.DistroDebian}
	mgr := NewExtensionManager(p)

	tests := []struct {
		pkg  string
		want string
	}{
		{"noisebynorthwest/php-spx", "spx"},
		{"openswoole/openswoole", "openswoole"},
		{"vendor/php-extension", "extension"},
		{"org/myext", "myext"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			got := mgr.pieExtensionName(tt.pkg)
			if got != tt.want {
				t.Errorf("pieExtensionName(%q) = %q, want %q", tt.pkg, got, tt.want)
			}
		})
	}
}
