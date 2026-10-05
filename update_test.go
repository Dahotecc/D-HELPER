package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// testNow is the time of the checks in the tests.
var testNow = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

// fakeGitHub is a TLS server that replaces the GitHub API and the downloads.
type fakeGitHub struct {
	server   *httptest.Server
	requests atomic.Int32
	// apiStatus and apiBody are the API response.
	apiStatus int
	apiBody   string
	// program and checksum are the files of the release.
	program  []byte
	checksum string
	// handlers replace the default response for some paths.
	handlers map[string]http.HandlerFunc
	// userAgent is the User-Agent of the last request.
	userAgent atomic.Value
}

// updateResult records the messages of the update.
type updateResult struct {
	questions  []string
	infos      []string
	errors     []string
	registered []string
}

// newFakeGitHub starts the server and replaces the addresses, the transport and the dialogs.
// answer is the answer of the user to the question.
func newFakeGitHub(t *testing.T, tag string, answer bool) (*fakeGitHub, *updateResult) {
	t.Helper()
	fake := &fakeGitHub{apiStatus: http.StatusOK, program: []byte("nouveau programme"), handlers: map[string]http.HandlerFunc{}}
	sum := sha256.Sum256(fake.program)
	fake.checksum = hex.EncodeToString(sum[:]) + "  d-helper.exe\n"

	fake.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.requests.Add(1)
		fake.userAgent.Store(r.Header.Get("User-Agent"))
		if handler, ok := fake.handlers[r.URL.Path]; ok {
			handler(w, r)
			return
		}
		switch r.URL.Path {
		case "/api":
			w.WriteHeader(fake.apiStatus)
			fmt.Fprint(w, fake.apiBody)
		case "/download/d-helper.exe":
			// GitHub sends a redirect to a file server.
			http.Redirect(w, r, fake.server.URL+"/objects/d-helper.exe", http.StatusFound)
		case "/objects/d-helper.exe":
			_, _ = w.Write(fake.program)
		case "/download/d-helper.exe.sha256":
			fmt.Fprint(w, fake.checksum)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(fake.server.Close)
	fake.apiBody = releaseJSON(tag, fake.server.URL+"/download/")

	saved := []any{updateAPIURL, updateAssetPrefix, updateRedirectHost, updateTransport, updateAsk, updateInfo, updateError, updateRegister, version}
	t.Cleanup(func() {
		updateAPIURL = saved[0].(string)
		updateAssetPrefix = saved[1].(string)
		updateRedirectHost = saved[2].(string)
		updateTransport, _ = saved[3].(http.RoundTripper)
		updateAsk = saved[4].(func(string) bool)
		updateInfo = saved[5].(func(string))
		updateError = saved[6].(func(string))
		updateRegister = saved[7].(func(string) error)
		version = saved[8].(string)
	})
	updateAPIURL = fake.server.URL + "/api"
	updateAssetPrefix = fake.server.URL + "/download/"
	updateRedirectHost = "127.0.0.1"
	updateTransport = fake.server.Client().Transport

	result := &updateResult{}
	updateAsk = func(text string) bool {
		result.questions = append(result.questions, text)
		return answer
	}
	updateInfo = func(text string) { result.infos = append(result.infos, text) }
	updateError = func(text string) { result.errors = append(result.errors, text) }
	updateRegister = func(exe string) error {
		result.registered = append(result.registered, exe)
		return nil
	}

	version = "2026.10.01.0800"
	t.Setenv("LOCALAPPDATA", t.TempDir())
	return fake, result
}

// releaseJSON gives an API response with the two files of the release.
func releaseJSON(tag, prefix string) string {
	return fmt.Sprintf(`{"tag_name":%q,"assets":[`+
		`{"name":"d-helper.exe","browser_download_url":%q},`+
		`{"name":"d-helper.exe.sha256","browser_download_url":%q}]}`,
		tag, prefix+"d-helper.exe", prefix+"d-helper.exe.sha256")
}

// installedExe writes an installed program and gives its path.
func installedExe(t *testing.T) string {
	t.Helper()
	exe, err := installPath()
	if err != nil {
		t.Fatal(err)
	}
	mustMkdir(t, filepath.Dir(exe))
	if err := os.WriteFile(exe, []byte("ancien programme"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe
}

// assertInstalled compares the content of the installed program.
func assertInstalled(t *testing.T, exe, want string) {
	t.Helper()
	data, err := os.ReadFile(exe)
	if err != nil || string(data) != want {
		t.Errorf("installed program = %q, %v: want %q", data, err, want)
	}
}

// assertNoTempFile makes sure that no download stays in the install folder.
func assertNoTempFile(t *testing.T, dir string) {
	t.Helper()
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".download") {
			t.Errorf("temporary file %s must not stay", entry.Name())
		}
	}
}

func TestParseVersion(t *testing.T) {
	good := map[string]string{
		"2026.10.02.1432": "2026-10-02 14:32",
		"2027.01.31.0005": "2027-01-31 00:05",
	}
	for text, want := range good {
		got, ok := parseVersion(text)
		if !ok || got.Format("2006-01-02 15:04") != want {
			t.Errorf("parseVersion(%q) = %v, %v: want %s", text, got, ok, want)
		}
	}
	for _, text := range []string{"dev", "ci", "sha-89783a0", "", "2026.10.2.1432", "2026.13.02.1432", "2026.10.02.2560", "2026.10.02.1432 ", "v2026.10.02.1432", "２026.10.02.1432"} {
		if _, ok := parseVersion(text); ok {
			t.Errorf("parseVersion(%q): want false", text)
		}
	}
	if _, ok := parseTag("v2026.10.02.1432"); !ok {
		t.Error("parseTag with v: want true")
	}
	for _, tag := range []string{"2026.10.02.1432", "V2026.10.02.1432", "sha-89783a0", "vdev"} {
		if _, ok := parseTag(tag); ok {
			t.Errorf("parseTag(%q): want false", tag)
		}
	}
}

func TestUpdateVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		tag     string
		ask     bool
	}{
		{"current version dev", "dev", "v2026.10.02.0800", false},
		{"current version in an other format", "sha-89783a0", "v2026.10.02.0800", false},
		{"tag in a bad format", "2026.10.01.0800", "sha-1234567", false},
		{"tag without v", "2026.10.01.0800", "2026.10.02.0800", false},
		{"older version", "2026.10.01.0800", "v2026.09.30.2359", false},
		{"same version", "2026.10.01.0800", "v2026.10.01.0800", false},
		{"newer version", "2026.10.01.0800", "v2026.10.01.0801", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake, result := newFakeGitHub(t, tt.tag, false)
			version = tt.current
			if got := runUpdate(testNow, false); got != 0 {
				t.Errorf("runUpdate = %d, want 0", got)
			}
			if asked := len(result.questions) == 1; asked != tt.ask {
				t.Errorf("question asked = %v, want %v", asked, tt.ask)
			}
			if len(result.infos)+len(result.errors) != 0 {
				t.Errorf("automatic check: want no message, got %v %v", result.infos, result.errors)
			}
			// A version that does not have the format never connects.
			if _, ok := parseVersion(tt.current); !ok && fake.requests.Load() != 0 {
				t.Error("version out of format: want no connection")
			}
		})
	}
}

func TestUpdateQuestionAndHeaders(t *testing.T) {
	fake, result := newFakeGitHub(t, "v2026.10.02.0930", false)
	runUpdate(testNow, false)
	want := "Une mise à jour de D-HELPER est disponible (version du 02/10/2026). Installer maintenant ?"
	if len(result.questions) != 1 || result.questions[0] != want {
		t.Errorf("questions = %q, want %q", result.questions, want)
	}
	if got := fake.userAgent.Load(); got != "D-HELPER/2026.10.01.0800" {
		t.Errorf("User-Agent = %v", got)
	}
}

func TestUpdateSilentAbandon(t *testing.T) {
	tests := map[string]func(f *fakeGitHub){
		"404":          func(f *fakeGitHub) { f.apiStatus = http.StatusNotFound },
		"500":          func(f *fakeGitHub) { f.apiStatus = http.StatusInternalServerError },
		"invalid JSON": func(f *fakeGitHub) { f.apiBody = `{"tag_name":` },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			fake, result := newFakeGitHub(t, "v2026.10.02.0800", true)
			change(fake)
			if got := runUpdate(testNow, false); got != 0 {
				t.Errorf("runUpdate = %d, want 0", got)
			}
			if len(result.questions)+len(result.infos)+len(result.errors) != 0 {
				t.Errorf("want no dialog, got %v %v %v", result.questions, result.infos, result.errors)
			}
		})
	}
}

func TestUpdateFullPath(t *testing.T) {
	fake, result := newFakeGitHub(t, "v2026.10.02.0800", true)
	exe := installedExe(t)

	if got := runUpdate(testNow, false); got != 0 {
		t.Fatalf("runUpdate = %d, want 0 (errors %v)", got, result.errors)
	}
	assertInstalled(t, exe, string(fake.program))
	assertNoTempFile(t, filepath.Dir(exe))
	if len(result.registered) != 1 || result.registered[0] != exe {
		t.Errorf("registered = %v, want the installed program", result.registered)
	}
	if want := "D-HELPER est à jour (version du 02/10/2026)."; len(result.infos) != 1 || result.infos[0] != want {
		t.Errorf("infos = %q, want %q", result.infos, want)
	}
	if state := readUpdateState(filepath.Dir(exe)); !state.LastCheck.Equal(testNow) {
		t.Errorf("lastCheck = %v, want %v", state.LastCheck, testNow)
	}
}

func TestUpdateRefusesAssetOutsidePrefix(t *testing.T) {
	fake, result := newFakeGitHub(t, "v2026.10.02.0800", true)
	exe := installedExe(t)
	for _, prefix := range []string{
		"https://example.com/Dahotecc/D-HELPER/releases/download/",
		fake.server.URL + "/other/",
		fake.server.URL + "/download/../other/",
	} {
		result.errors = nil
		fake.apiBody = releaseJSON("v2026.10.02.0800", prefix)
		if got := runUpdate(testNow, true); got != 1 {
			t.Errorf("%s: runUpdate = %d, want 1", prefix, got)
		}
		if len(result.errors) != 1 || !strings.Contains(result.errors[0], "adresse de téléchargement refusée") {
			t.Errorf("%s: errors = %q", prefix, result.errors)
		}
	}
	assertInstalled(t, exe, "ancien programme")
}

func TestUpdateRefusesBadRedirect(t *testing.T) {
	for name, target := range map[string]string{
		"other host": "https://example.com/d-helper.exe",
		"HTTP":       "http://127.0.0.1/d-helper.exe",
	} {
		t.Run(name, func(t *testing.T) {
			fake, result := newFakeGitHub(t, "v2026.10.02.0800", true)
			exe := installedExe(t)
			fake.handlers["/download/d-helper.exe"] = func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, target, http.StatusFound)
			}
			if got := runUpdate(testNow, false); got != 1 {
				t.Errorf("runUpdate = %d, want 1", got)
			}
			if len(result.errors) != 1 || !strings.Contains(result.errors[0], "La version installée n'a pas changé.") {
				t.Errorf("errors = %q", result.errors)
			}
			assertInstalled(t, exe, "ancien programme")
			assertNoTempFile(t, filepath.Dir(exe))
		})
	}

	allowed := []string{"https://github.com/a", "https://objects.githubusercontent.com/a", "https://release-assets.githubusercontent.com/a"}
	refused := []string{"http://github.com/a", "https://github.com.example.com/a", "https://evilgithubusercontent.com/a", "https://user@github.com/a", "https://api.github.com/a"}
	for _, address := range append(allowed, refused...) {
		req, _ := http.NewRequest(http.MethodGet, address, nil)
		err := checkUpdateRedirect(req, nil)
		isAllowed := address == allowed[0] || address == allowed[1] || address == allowed[2]
		if (err == nil) != isAllowed {
			t.Errorf("redirect to %s: error %v", address, err)
		}
	}
}

func TestUpdateWrongChecksum(t *testing.T) {
	fake, result := newFakeGitHub(t, "v2026.10.02.0800", true)
	exe := installedExe(t)
	sum := sha256.Sum256([]byte("autre contenu"))
	fake.checksum = hex.EncodeToString(sum[:]) + "  d-helper.exe\n"

	if got := runUpdate(testNow, false); got != 1 {
		t.Errorf("runUpdate = %d, want 1", got)
	}
	if want := "Mise à jour refusée : le fichier téléchargé est corrompu."; len(result.errors) != 1 || result.errors[0] != want {
		t.Errorf("errors = %q, want %q", result.errors, want)
	}
	assertInstalled(t, exe, "ancien programme")
	assertNoTempFile(t, filepath.Dir(exe))
	if len(result.registered) != 0 {
		t.Error("corrupted file: the protocol must not change")
	}
}

func TestUpdateTooLarge(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		// The size is in the header.
		"announced size": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", fmt.Sprint(updateMaxSize+1))
		},
		// The size is not in the header: the server sends 50 MB + 1 byte.
		"streamed size": func(w http.ResponseWriter, r *http.Request) {
			block := make([]byte, 1<<20)
			for i := 0; i < 50; i++ {
				_, _ = w.Write(block)
				w.(http.Flusher).Flush()
			}
			_, _ = w.Write([]byte{0})
		},
	}
	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			fake, result := newFakeGitHub(t, "v2026.10.02.0800", true)
			exe := installedExe(t)
			fake.handlers["/objects/d-helper.exe"] = handler
			if got := runUpdate(testNow, false); got != 1 {
				t.Errorf("runUpdate = %d, want 1", got)
			}
			if len(result.errors) != 1 || !strings.Contains(result.errors[0], "fichier trop grand") {
				t.Errorf("errors = %q", result.errors)
			}
			assertInstalled(t, exe, "ancien programme")
			assertNoTempFile(t, filepath.Dir(exe))
		})
	}
}

func TestUpdateOncePerDay(t *testing.T) {
	fake, result := newFakeGitHub(t, "v2026.10.01.0800", false)
	runUpdate(testNow, false)
	runUpdate(testNow.Add(23*time.Hour), false)
	if got := fake.requests.Load(); got != 1 {
		t.Errorf("requests before 24 hours = %d, want 1", got)
	}
	runUpdate(testNow.Add(24*time.Hour), false)
	if got := fake.requests.Load(); got != 2 {
		t.Errorf("requests after 24 hours = %d, want 2", got)
	}
	// --update does not have the 24-hour limit.
	runUpdate(testNow.Add(24*time.Hour), true)
	if got := fake.requests.Load(); got != 3 {
		t.Errorf("requests with --update = %d, want 3", got)
	}
	if want := "D-HELPER est à jour."; len(result.infos) != 1 || result.infos[0] != want {
		t.Errorf("infos = %q, want %q", result.infos, want)
	}
}

func TestUpdateRefusedVersion(t *testing.T) {
	_, result := newFakeGitHub(t, "v2026.10.02.0800", false)
	runUpdate(testNow, false)
	if len(result.questions) != 1 {
		t.Fatalf("questions = %d, want 1", len(result.questions))
	}
	// Next day: the refused version is not shown again.
	runUpdate(testNow.Add(25*time.Hour), false)
	runUpdate(testNow.Add(6*24*time.Hour), false)
	if len(result.questions) != 1 {
		t.Errorf("before 7 days: questions = %d, want 1", len(result.questions))
	}
	runUpdate(testNow.Add(7*24*time.Hour), false)
	if len(result.questions) != 2 {
		t.Errorf("after 7 days: questions = %d, want 2", len(result.questions))
	}
}

func TestUpdateRefusalOfOtherVersion(t *testing.T) {
	fake, result := newFakeGitHub(t, "v2026.10.02.0800", false)
	runUpdate(testNow, false)
	fake.apiBody = releaseJSON("v2026.10.03.0800", fake.server.URL+"/download/")
	runUpdate(testNow.Add(25*time.Hour), false)
	if len(result.questions) != 2 {
		t.Errorf("new version after a refusal: questions = %d, want 2", len(result.questions))
	}
}

func TestUpdateStateUnreadable(t *testing.T) {
	fake, _ := newFakeGitHub(t, "v2026.10.01.0800", false)
	exe := installedExe(t)
	file := filepath.Join(filepath.Dir(exe), updateStateFileName)
	if err := os.WriteFile(file, []byte("{pas du JSON"), 0o644); err != nil {
		t.Fatal(err)
	}
	runUpdate(testNow, false)
	if got := fake.requests.Load(); got != 1 {
		t.Errorf("unreadable state: requests = %d, want 1", got)
	}
	if state := readUpdateState(filepath.Dir(exe)); !state.LastCheck.Equal(testNow) {
		t.Errorf("lastCheck = %v, want %v", state.LastCheck, testNow)
	}
}

func TestUpdateNetworkError(t *testing.T) {
	fake, result := newFakeGitHub(t, "v2026.10.02.0800", true)
	exe := installedExe(t)
	fake.handlers["/download/d-helper.exe.sha256"] = func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "erreur", http.StatusBadGateway)
	}
	if got := runUpdate(testNow, false); got != 1 {
		t.Errorf("runUpdate = %d, want 1", got)
	}
	want := "Mise à jour de D-HELPER impossible : réponse 502 de GitHub. La version installée n'a pas changé."
	if len(result.errors) != 1 || result.errors[0] != want {
		t.Errorf("errors = %q, want %q", result.errors, want)
	}
	assertInstalled(t, exe, "ancien programme")

	// --update shows a message if GitHub does not answer.
	fake.server.Close()
	result.errors = nil
	if got := runUpdate(testNow, true); got != 1 {
		t.Errorf("--update without network = %d, want 1", got)
	}
	if len(result.errors) != 1 || !strings.HasPrefix(result.errors[0], "Vérification de la mise à jour impossible") {
		t.Errorf("errors = %q", result.errors)
	}
}

func TestRunUpdateDevVersion(t *testing.T) {
	fake, result := newFakeGitHub(t, "v2026.10.02.0800", true)
	version = "dev"
	if got := run([]string{"--update"}); got != 0 {
		t.Errorf("run --update = %d, want 0", got)
	}
	if fake.requests.Load() != 0 {
		t.Error("version dev: want no connection")
	}
	if want := "Cette version de D-HELPER (dev) ne se met pas à jour."; len(result.infos) != 1 || result.infos[0] != want {
		t.Errorf("infos = %q, want %q", result.infos, want)
	}
}

func TestChecksumFormat(t *testing.T) {
	fake, _ := newFakeGitHub(t, "v2026.10.02.0800", true)
	client := newUpdateClient(updateAPITimeout)
	address := fake.server.URL + "/download/d-helper.exe.sha256"
	sum := strings.Repeat("ab", 32)
	for content, ok := range map[string]bool{
		sum + "  d-helper.exe\n":    true,
		sum + " *d-helper.exe\n":    true,
		sum + "  autre.exe\n":       false,
		sum + "\n":                  false,
		"zz  d-helper.exe\n":        false,
		sum[:62] + "  d-helper.exe": false,
	} {
		fake.checksum = content
		_, err := downloadChecksum(client, address)
		if (err == nil) != ok {
			t.Errorf("checksum %q: error %v", content, err)
		}
	}
}
