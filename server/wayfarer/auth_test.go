package wayfarer

import (
	"net/http"
	"testing"
)

func TestIsWayfarerAuthFailure(t *testing.T) {
	cases := []struct {
		name   string
		status int
		loc    string
		ct     string
		body   string
		want   bool
	}{
		{name: "401", status: 401, want: true},
		{name: "403", status: 403, want: true},
		{name: "302 login", status: 302, loc: "https://wayfarer.scopely.com/login", want: true},
		{name: "html 200", status: 200, ct: "text/html; charset=utf-8", body: "<!DOCTYPE html>", want: true},
		{name: "json ok", status: 200, ct: "application/json", body: `{"result":{}}`, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tc.status,
				Header:     http.Header{},
			}
			if tc.loc != "" {
				resp.Header.Set("Location", tc.loc)
			}
			if tc.ct != "" {
				resp.Header.Set("Content-Type", tc.ct)
			}
			got := isWayfarerAuthFailure(resp, []byte(tc.body))
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
