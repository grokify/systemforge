package scim

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "req-12345"

	ctx = WithRequestID(ctx, requestID)

	got := RequestIDFromContext(ctx)
	if got != requestID {
		t.Errorf("RequestIDFromContext() = %q, want %q", got, requestID)
	}
}

func TestRequestIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()

	got := RequestIDFromContext(ctx)
	if got != "" {
		t.Errorf("RequestIDFromContext() = %q, want empty string", got)
	}
}

func TestWithAuthSubject(t *testing.T) {
	ctx := context.Background()
	subject := "user-123"

	ctx = WithAuthSubject(ctx, subject)

	got := AuthSubjectFromContext(ctx)
	if got != subject {
		t.Errorf("AuthSubjectFromContext() = %q, want %q", got, subject)
	}
}

func TestAuthSubjectFromContext_Empty(t *testing.T) {
	ctx := context.Background()

	got := AuthSubjectFromContext(ctx)
	if got != "" {
		t.Errorf("AuthSubjectFromContext() = %q, want empty string", got)
	}
}

func TestWithAuthScopes(t *testing.T) {
	ctx := context.Background()
	scopes := []string{"scim:read", "scim:write", "admin"}

	ctx = WithAuthScopes(ctx, scopes)

	got := AuthScopesFromContext(ctx)
	if len(got) != len(scopes) {
		t.Errorf("AuthScopesFromContext() returned %d scopes, want %d", len(got), len(scopes))
	}

	for i, scope := range scopes {
		if got[i] != scope {
			t.Errorf("AuthScopesFromContext()[%d] = %q, want %q", i, got[i], scope)
		}
	}
}

func TestAuthScopesFromContext_Empty(t *testing.T) {
	ctx := context.Background()

	got := AuthScopesFromContext(ctx)
	if got != nil {
		t.Errorf("AuthScopesFromContext() = %v, want nil", got)
	}
}

func TestHasAuthScope(t *testing.T) {
	ctx := context.Background()
	scopes := []string{"scim:read", "scim:write"}
	ctx = WithAuthScopes(ctx, scopes)

	tests := []struct {
		scope string
		want  bool
	}{
		{"scim:read", true},
		{"scim:write", true},
		{"admin", false},
		{"", false},
	}

	for _, tc := range tests {
		got := HasAuthScope(ctx, tc.scope)
		if got != tc.want {
			t.Errorf("HasAuthScope(ctx, %q) = %v, want %v", tc.scope, got, tc.want)
		}
	}
}

func TestHasAuthScope_EmptyContext(t *testing.T) {
	ctx := context.Background()

	if HasAuthScope(ctx, "any") {
		t.Error("HasAuthScope() = true for empty context, want false")
	}
}

func TestExtractRequestMetadata(t *testing.T) {
	tests := []struct {
		name        string
		queryString string
		headers     map[string]string
		want        RequestMetadata
	}{
		{
			name:        "empty request",
			queryString: "",
			want: RequestMetadata{
				StartIndex: 1,
				SortOrder:  "ascending",
			},
		},
		{
			name:        "with filter",
			queryString: "filter=userName%20eq%20%22john%22",
			want: RequestMetadata{
				Filter:     "userName eq \"john\"",
				StartIndex: 1,
				SortOrder:  "ascending",
			},
		},
		{
			name:        "with pagination",
			queryString: "startIndex=10&count=25",
			want: RequestMetadata{
				StartIndex: 10,
				Count:      25,
				SortOrder:  "ascending",
			},
		},
		{
			name:        "with sorting",
			queryString: "sortBy=userName&sortOrder=descending",
			want: RequestMetadata{
				StartIndex: 1,
				SortBy:     "userName",
				SortOrder:  "descending",
			},
		},
		{
			name:        "with attributes",
			queryString: "attributes=userName,name,emails",
			want: RequestMetadata{
				StartIndex: 1,
				SortOrder:  "ascending",
				Attributes: []string{"userName", "name", "emails"},
			},
		},
		{
			name:        "with excluded attributes",
			queryString: "excludedAttributes=password,groups",
			want: RequestMetadata{
				StartIndex:         1,
				SortOrder:          "ascending",
				ExcludedAttributes: []string{"password", "groups"},
			},
		},
		{
			name:        "with ETag headers",
			queryString: "",
			headers: map[string]string{
				"If-Match":      "\"abc123\"",
				"If-None-Match": "\"xyz789\"",
			},
			want: RequestMetadata{
				StartIndex:  1,
				SortOrder:   "ascending",
				IfMatch:     "\"abc123\"",
				IfNoneMatch: "\"xyz789\"",
			},
		},
		{
			name:        "invalid startIndex uses default",
			queryString: "startIndex=-1",
			want: RequestMetadata{
				StartIndex: 1,
				SortOrder:  "ascending",
			},
		},
		{
			name:        "invalid count uses default",
			queryString: "count=-5",
			want: RequestMetadata{
				StartIndex: 1,
				Count:      0,
				SortOrder:  "ascending",
			},
		},
		{
			name:        "case insensitive sortOrder",
			queryString: "sortOrder=DESCENDING",
			want: RequestMetadata{
				StartIndex: 1,
				SortOrder:  "descending",
			},
		},
		{
			name:        "attributes with whitespace",
			queryString: "attributes=%20userName%20,%20name%20,%20emails%20",
			want: RequestMetadata{
				StartIndex: 1,
				SortOrder:  "ascending",
				Attributes: []string{"userName", "name", "emails"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			url := "/scim/v2/Users"
			if tc.queryString != "" {
				url += "?" + tc.queryString
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			got := ExtractRequestMetadata(req)

			if got.Filter != tc.want.Filter {
				t.Errorf("Filter = %q, want %q", got.Filter, tc.want.Filter)
			}
			if got.StartIndex != tc.want.StartIndex {
				t.Errorf("StartIndex = %d, want %d", got.StartIndex, tc.want.StartIndex)
			}
			if got.Count != tc.want.Count {
				t.Errorf("Count = %d, want %d", got.Count, tc.want.Count)
			}
			if got.SortBy != tc.want.SortBy {
				t.Errorf("SortBy = %q, want %q", got.SortBy, tc.want.SortBy)
			}
			if got.SortOrder != tc.want.SortOrder {
				t.Errorf("SortOrder = %q, want %q", got.SortOrder, tc.want.SortOrder)
			}
			if got.IfMatch != tc.want.IfMatch {
				t.Errorf("IfMatch = %q, want %q", got.IfMatch, tc.want.IfMatch)
			}
			if got.IfNoneMatch != tc.want.IfNoneMatch {
				t.Errorf("IfNoneMatch = %q, want %q", got.IfNoneMatch, tc.want.IfNoneMatch)
			}
			if len(got.Attributes) != len(tc.want.Attributes) {
				t.Errorf("Attributes = %v, want %v", got.Attributes, tc.want.Attributes)
			}
			if len(got.ExcludedAttributes) != len(tc.want.ExcludedAttributes) {
				t.Errorf("ExcludedAttributes = %v, want %v", got.ExcludedAttributes, tc.want.ExcludedAttributes)
			}
		})
	}
}

func TestRequestMetadata_ToListOptions(t *testing.T) {
	tests := []struct {
		name         string
		metadata     RequestMetadata
		defaultCount int
		wantCount    int
	}{
		{
			name: "uses metadata count when set",
			metadata: RequestMetadata{
				Filter:     "active eq true",
				StartIndex: 5,
				Count:      50,
				SortBy:     "email",
				SortOrder:  "ascending",
			},
			defaultCount: 100,
			wantCount:    50,
		},
		{
			name: "uses default count when zero",
			metadata: RequestMetadata{
				StartIndex: 1,
				Count:      0,
			},
			defaultCount: 100,
			wantCount:    100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.metadata.ToListOptions(tc.defaultCount)

			if opts.Count != tc.wantCount {
				t.Errorf("Count = %d, want %d", opts.Count, tc.wantCount)
			}
			if opts.Filter != tc.metadata.Filter {
				t.Errorf("Filter = %q, want %q", opts.Filter, tc.metadata.Filter)
			}
			if opts.StartIndex != tc.metadata.StartIndex {
				t.Errorf("StartIndex = %d, want %d", opts.StartIndex, tc.metadata.StartIndex)
			}
			if opts.SortBy != tc.metadata.SortBy {
				t.Errorf("SortBy = %q, want %q", opts.SortBy, tc.metadata.SortBy)
			}
			if opts.SortOrder != tc.metadata.SortOrder {
				t.Errorf("SortOrder = %q, want %q", opts.SortOrder, tc.metadata.SortOrder)
			}
		})
	}
}

func TestGenerateETag(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"1", `"1"`},
		{"abc123", `"abc123"`},
		{"W/abc", `"W/abc"`},
		{"", `""`},
	}

	for _, tc := range tests {
		got := GenerateETag(tc.version)
		if got != tc.want {
			t.Errorf("GenerateETag(%q) = %q, want %q", tc.version, got, tc.want)
		}
	}
}

func TestParseETag(t *testing.T) {
	tests := []struct {
		etag string
		want string
	}{
		{`"abc123"`, "abc123"},
		{`"1"`, "1"},
		{`W/"abc123"`, "abc123"},
		{`abc123`, "abc123"},
		{`""`, ""},
		{"", ""},
	}

	for _, tc := range tests {
		got := ParseETag(tc.etag)
		if got != tc.want {
			t.Errorf("ParseETag(%q) = %q, want %q", tc.etag, got, tc.want)
		}
	}
}

func TestWriteResponse(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		etag       string
		body       []byte
		wantStatus int
		wantETag   string
	}{
		{
			name:       "success with ETag",
			status:     http.StatusOK,
			etag:       `"abc123"`,
			body:       []byte(`{"id":"123"}`),
			wantStatus: http.StatusOK,
			wantETag:   `"abc123"`,
		},
		{
			name:       "success without ETag",
			status:     http.StatusOK,
			etag:       "",
			body:       []byte(`{"id":"123"}`),
			wantStatus: http.StatusOK,
			wantETag:   "",
		},
		{
			name:       "created",
			status:     http.StatusCreated,
			etag:       `"new"`,
			body:       []byte(`{"id":"456"}`),
			wantStatus: http.StatusCreated,
			wantETag:   `"new"`,
		},
		{
			name:       "no content",
			status:     http.StatusNoContent,
			etag:       "",
			body:       nil,
			wantStatus: http.StatusNoContent,
			wantETag:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			WriteResponse(rec, tc.status, tc.etag, tc.body)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/scim+json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/scim+json")
			}

			gotETag := rec.Header().Get("ETag")
			if gotETag != tc.wantETag {
				t.Errorf("ETag = %q, want %q", gotETag, tc.wantETag)
			}

			if tc.body != nil {
				if rec.Body.String() != string(tc.body) {
					t.Errorf("body = %q, want %q", rec.Body.String(), string(tc.body))
				}
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input string
		sep   string
		want  []string
	}{
		{"a,b,c", ",", []string{"a", "b", "c"}},
		{" a , b , c ", ",", []string{"a", "b", "c"}},
		{"a", ",", []string{"a"}},
		{"", ",", []string{}},
		{"  ,  ,  ", ",", []string{}},
		{"a,,b", ",", []string{"a", "b"}},
	}

	for _, tc := range tests {
		got := splitAndTrim(tc.input, tc.sep)
		if len(got) != len(tc.want) {
			t.Errorf("splitAndTrim(%q, %q) = %v, want %v", tc.input, tc.sep, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitAndTrim(%q, %q)[%d] = %q, want %q", tc.input, tc.sep, i, got[i], tc.want[i])
			}
		}
	}
}
