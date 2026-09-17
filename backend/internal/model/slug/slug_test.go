package slug

import "testing"

func TestMake(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Git Commands", "git-commands"},
		{"  Hello, World!  ", "hello-world"},
		{"Frontend_Guide v2", "frontend-guide-v2"},
		{"C++ & Rust", "c-rust"},
		{"Já Vou", "ja-vou"},
		{"", "untitled"},
		{"---", "untitled"},
		{"###", "untitled"},
		{"Multiple   Spaces", "multiple-spaces"},
	}
	for _, tt := range tests {
		if got := Make(tt.in); got != tt.want {
			t.Errorf("Make(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMakeTruncates(t *testing.T) {
	long := ""
	for i := 0; i < 300; i++ {
		long += "a"
	}
	if got := Make(long); len(got) > MaxLength {
		t.Errorf("Make(long) length = %d, want <= %d", len(got), MaxLength)
	}
}
