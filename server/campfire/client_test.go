package campfire

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchRetriesPlatform400ThenSucceeds(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			writePlatform400(w)
			return
		}
		writeEmptyMap(w)
	}))
	defer srv.Close()

	if _, err := testClient(srv).FetchWithDropTypes(context.Background(), []string{"1", "2"}, 15, nil); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls %d, want 2", calls)
	}
}

func TestFetchSplitsPlatform400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Count(string(body), `"s2CellId"`) > 1 {
			writePlatform400(w)
			return
		}
		writeEmptyMap(w)
	}))
	defer srv.Close()

	_, err := testClient(srv).FetchWithDropTypes(context.Background(), []string{"1", "2", "3"}, 15, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFetchPlatform400OnEveryCell(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writePlatform400(w)
	}))
	defer srv.Close()

	_, err := testClient(srv).FetchWithDropTypes(context.Background(), []string{"1", "2"}, 15, nil)
	if err == nil || !isPlatformBadRequest(err) {
		t.Fatalf("got %v", err)
	}
}

func testClient(srv *httptest.Server) *Client {
	return &Client{
		url:              srv.URL,
		http:             srv.Client(),
		retries:          1,
		realityChannelID: DefaultRealityChannelID,
		s2CellLevel:      15,
	}
}

func writePlatform400(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"errors":[{"message":"fetch map objects for game map: PgoPlatformRequest GET_MAP_OBJECTS_FOR_CAMPFIRE: HTTP Mapping: 400 Bad Request"}]}`)
}

func writeEmptyMap(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"data":{"realityChannelMapObjectsByS2Cells":{"mapObjectsByS2CellsAndTypes":[]}}}`)
}
