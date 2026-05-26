package routes

import "testing"

func TestPathToPattern(t *testing.T) {
	registered := map[string]bool{
		"":              true, // root routes package
		"users":         true,
		"users/posts":   true,
		"posts":         true,
		"auth/login":    true,
		"auth/register": true,
	}

	cases := []struct {
		pkgPath string
		want    string
	}{
		{"myapp/routes", "/"},
		{"myapp/routes/users", "/users"},
		{"myapp/routes/posts", "/posts"},
		{"myapp/routes/auth/login", "/auth/login"},
		{"myapp/routes/auth/register", "/auth/register"},
		// Nested resource: users has a route, so users/posts gets :id inserted
		{"myapp/routes/users/posts", "/users/:id/posts"},
	}

	for _, tc := range cases {
		got := pathToPattern(tc.pkgPath, registered, nil)
		if got != tc.want {
			t.Errorf("pathToPattern(%q) = %q, want %q", tc.pkgPath, got, tc.want)
		}
	}
}

func TestPathToPattern_DeepNesting(t *testing.T) {
	registered := map[string]bool{
		"users":                true,
		"users/posts":          true,
		"users/posts/comments": true,
	}

	cases := []struct {
		pkgPath string
		want    string
	}{
		{"myapp/routes/users", "/users"},
		{"myapp/routes/users/posts", "/users/:id/posts"},
		{"myapp/routes/users/posts/comments", "/users/:id/posts/:post_id/comments"},
	}

	for _, tc := range cases {
		got := pathToPattern(tc.pkgPath, registered, nil)
		if got != tc.want {
			t.Errorf("pathToPattern(%q) = %q, want %q", tc.pkgPath, got, tc.want)
		}
	}
}

func TestPathToPattern_MemberOverride(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	registered := map[string]bool{
		"users":       true,
		"users/posts": true,
	}

	cases := []struct {
		name     string
		pkgPath  string
		reg      map[string]bool
		override *bool
		want     string
	}{
		{
			name:     "force member when parent not registered",
			pkgPath:  "myapp/routes/kampala/uganda",
			reg:      map[string]bool{},
			override: boolPtr(true),
			want:     "/kampala/:id/uganda",
		},
		{
			name:     "suppress member when parent is registered",
			pkgPath:  "myapp/routes/users/posts",
			reg:      registered,
			override: boolPtr(false),
			want:     "/users/posts",
		},
		{
			name:     "nil override uses auto-detection (parent registered)",
			pkgPath:  "myapp/routes/users/posts",
			reg:      registered,
			override: nil,
			want:     "/users/:id/posts",
		},
		{
			name:     "nil override uses auto-detection (parent not registered)",
			pkgPath:  "myapp/routes/auth/login",
			reg:      map[string]bool{"auth/login": true},
			override: nil,
			want:     "/auth/login",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pathToPattern(tc.pkgPath, tc.reg, tc.override)
			if got != tc.want {
				t.Errorf("pathToPattern(%q) = %q, want %q", tc.pkgPath, got, tc.want)
			}
		})
	}
}

func TestSingularize(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"users", "user"},
		{"posts", "post"},
		{"comments", "comment"},
		{"categories", "category"},
		{"buses", "bus"},
		{"boxes", "box"},
		{"addresses", "address"},
		{"class", "class"}, // doesn't strip "ss"
	}

	for _, tc := range cases {
		got := singularize(tc.input)
		if got != tc.want {
			t.Errorf("singularize(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
