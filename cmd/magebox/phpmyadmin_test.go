package main

import "testing"

func TestPhpMyAdminServerIndex(t *testing.T) {
	env := []string{
		"PATH=/usr/local/bin",
		"PMA_HOSTS=magebox-mysql-8.0, magebox-mariadb-12.3",
		"PMA_USER=root",
	}

	tests := []struct {
		name string
		env  []string
		host string
		want int
	}{
		{name: "first server", env: env, host: "magebox-mysql-8.0", want: 1},
		{name: "second server, spaces trimmed", env: env, host: "magebox-mariadb-12.3", want: 2},
		{name: "not listed", env: env, host: "magebox-mysql-8.4", want: 0},
		{name: "container from before PMA_HOSTS", env: []string{"PMA_HOST=magebox-mysql-8.0"}, host: "magebox-mysql-8.0", want: 0},
		{name: "no environment", env: nil, host: "magebox-mysql-8.0", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := phpMyAdminServerIndex(tt.env, tt.host); got != tt.want {
				t.Errorf("phpMyAdminServerIndex() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPhpMyAdminDatabaseURL(t *testing.T) {
	got := phpMyAdminDatabaseURL("http://localhost:8036", 2, "my_shop")
	want := "http://localhost:8036/index.php?db=my_shop&route=%2Fdatabase%2Fstructure&server=2"
	if got != want {
		t.Errorf("phpMyAdminDatabaseURL() = %q, want %q", got, want)
	}
}
