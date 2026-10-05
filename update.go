package main

// This file updates D-HELPER with the latest release of the GitHub repository.
//
// Security:
//   - This is the only network connection of D-HELPER. It goes to GitHub only:
//     the address of the API is a constant (updateAPIURL), each download must start
//     with updateAssetPrefix, and a redirect can go only to github.com or
//     *.githubusercontent.com, in HTTPS.
//   - No data of the d-helper:// link has an effect on the update. The update starts
//     only after the folder of the link is open.
//   - D-HELPER installs a new version only if the user clicks "Oui". It never installs
//     an older version or the same version again.
//   - D-HELPER compares the SHA-256 of the downloaded file with the .sha256 file of the
//     release before it replaces the installed program. A corrupted file is deleted.
//   - The repository is public: no token and no secret are necessary.
//
// Why: the users get the new versions without a manual download.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Addresses of GitHub. Only the tests replace these values. There is no runtime configuration.
var (
	// updateAPIURL is the address of the latest release in the GitHub API.
	updateAPIURL = "https://api.github.com/repos/Dahotecc/D-HELPER/releases/latest"
	// updateAssetPrefix is the start of each download address.
	updateAssetPrefix = "https://github.com/Dahotecc/D-HELPER/releases/download/"
	// updateRedirectHost is a host that a redirect can go to.
	updateRedirectHost = "github.com"
	// updateRedirectHostSuffix is the end of the other hosts that a redirect can go to.
	updateRedirectHostSuffix = ".githubusercontent.com"
	// updateTransport is the HTTP transport. nil is the default transport.
	updateTransport http.RoundTripper
)

// Functions that the update uses. Only the tests replace them.
var (
	updateAsk      = askYesNo
	updateInfo     = showInfo
	updateError    = showError
	updateRegister = registerProtocol
)

const (
	// updateCheckInterval is the minimum time between two automatic checks.
	updateCheckInterval = 24 * time.Hour
	// updateRefusalDelay is the time during which D-HELPER does not show a refused version again.
	updateRefusalDelay = 7 * 24 * time.Hour
	// updateAPITimeout is the total time of the request to the API.
	updateAPITimeout = 10 * time.Second
	// updateDownloadTimeout is the total time of one download.
	updateDownloadTimeout = 5 * time.Minute
	// updateMaxSize is the maximum size of the downloaded program (50 MB).
	updateMaxSize = 50 << 20
	// updateMaxAPISize is the maximum size of the API response.
	updateMaxAPISize = 1 << 20
	// updateMaxChecksumSize is the maximum size of the .sha256 file.
	updateMaxChecksumSize = 1024
	// updateStateFileName is the state file in the install folder.
	updateStateFileName = "update.json"
	// checksumName is the name of the checksum file of the release.
	checksumName = exeName + ".sha256"
	// versionLayout is the format of a version: AAAA.MM.JJ.HHMM (UTC time of the build).
	versionLayout = "2006.01.02.1504"
	// dateLayout is the date that the user sees: JJ/MM/AAAA.
	dateLayout = "02/01/2006"
)

// Texts that the user sees.
const (
	msgUpdateAsk       = "Une mise à jour de D-HELPER est disponible (version du %s). Installer maintenant ?"
	msgUpdateDone      = "D-HELPER est à jour (version du %s)."
	msgUpToDate        = "D-HELPER est à jour."
	msgUpdateCorrupt   = "Mise à jour refusée : le fichier téléchargé est corrompu."
	msgUpdateFailed    = "Mise à jour de D-HELPER impossible : %s. La version installée n'a pas changé."
	msgUpdateCheckFail = "Vérification de la mise à jour impossible : %s."
	msgUpdateNoVersion = "Cette version de D-HELPER (%s) ne se met pas à jour."
	msgUpdateRegister  = "La nouvelle version de D-HELPER est copiée, mais le protocole d-helper:// n'est pas enregistré : %s. Double-cliquez sur le programme installé pour réparer l'installation."
)

// versionPattern is the format of a version, without the "v" of the tag.
var versionPattern = regexp.MustCompile(`^[0-9]{4}\.[0-9]{2}\.[0-9]{2}\.[0-9]{4}$`)

// errUpdateCorrupt tells that the SHA-256 of the downloaded file is not correct.
var errUpdateCorrupt = errors.New("corrupted download")

// updateState is the content of update.json.
type updateState struct {
	// LastCheck is the time of the last check.
	LastCheck time.Time `json:"lastCheck"`
	// RefusedVersion is the tag of the last version that the user refused.
	RefusedVersion string `json:"refusedVersion,omitempty"`
	// RefusedAt is the time of the refusal.
	RefusedAt time.Time `json:"refusedAt,omitempty"`
}

// release is the part of the GitHub API response that D-HELPER uses.
type release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// parseVersion reads a version AAAA.MM.JJ.HHMM. The result is a time, thus the comparison is numerical.
// It gives false for "dev" and for all other formats.
func parseVersion(text string) (time.Time, bool) {
	if !versionPattern.MatchString(text) {
		return time.Time{}, false
	}
	t, err := time.Parse(versionLayout, text)
	return t, err == nil
}

// parseTag reads a tag vAAAA.MM.JJ.HHMM.
func parseTag(tag string) (time.Time, bool) {
	text, found := strings.CutPrefix(tag, "v")
	if !found {
		return time.Time{}, false
	}
	return parseVersion(text)
}

// autoUpdate is the automatic check after a link opened a folder.
// It does a check one time in 24 hours at most.
func autoUpdate() {
	runUpdate(time.Now(), false)
}

// runUpdate checks the latest release and proposes it to the user.
// manual is true for --update: no 24-hour limit, no 7-day limit, and a message for each result.
// In the automatic mode, a problem before the question gives no message.
func runUpdate(now time.Time, manual bool) int {
	current, ok := parseVersion(version)
	if !ok {
		// A version "dev" or a version in an other format never checks.
		if manual {
			updateInfo(fmt.Sprintf(msgUpdateNoVersion, version))
		}
		return 0
	}
	exe, err := installPath()
	if err != nil {
		if manual {
			updateError(fmt.Sprintf(msgUpdateCheckFail, err))
			return 1
		}
		return 0
	}
	dir := filepath.Dir(exe)

	state := readUpdateState(dir)
	if !manual && !state.checkIsDue(now) {
		return 0
	}
	state.LastCheck = now
	// Record the check before the connection. Thus a network error does not cause a check at each link.
	_ = writeUpdateState(dir, state)

	latest, err := fetchLatestRelease()
	if err != nil {
		if manual {
			updateError(fmt.Sprintf(msgUpdateCheckFail, err))
			return 1
		}
		return 0
	}
	latestTime, ok := parseTag(latest.TagName)
	if !ok || !latestTime.After(current) {
		// Never go back to an older version.
		if manual {
			updateInfo(msgUpToDate)
		}
		return 0
	}
	if !manual && state.isRefused(latest.TagName, now) {
		return 0
	}

	date := latestTime.Format(dateLayout)
	if !updateAsk(fmt.Sprintf(msgUpdateAsk, date)) {
		state.RefusedVersion = latest.TagName
		state.RefusedAt = now
		_ = writeUpdateState(dir, state)
		return 0
	}

	installed, err := installRelease(latest, exe)
	switch {
	case errors.Is(err, errUpdateCorrupt):
		updateError(msgUpdateCorrupt)
		return 1
	case err != nil && installed:
		updateError(fmt.Sprintf(msgUpdateRegister, err))
		return 1
	case err != nil:
		updateError(fmt.Sprintf(msgUpdateFailed, err))
		return 1
	}
	updateInfo(fmt.Sprintf(msgUpdateDone, date))
	return 0
}

// checkIsDue tells if the last check is older than 24 hours.
// A last check in the future (wrong clock) does not stop the check.
func (s updateState) checkIsDue(now time.Time) bool {
	age := now.Sub(s.LastCheck)
	return age < 0 || age >= updateCheckInterval
}

// isRefused tells if the user refused this version less than 7 days ago.
func (s updateState) isRefused(tag string, now time.Time) bool {
	age := now.Sub(s.RefusedAt)
	return tag == s.RefusedVersion && age >= 0 && age < updateRefusalDelay
}

// readUpdateState reads update.json. A missing or unreadable file gives an empty state.
func readUpdateState(dir string) updateState {
	var state updateState
	data, err := os.ReadFile(filepath.Join(dir, updateStateFileName))
	if err != nil {
		return updateState{}
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return updateState{}
	}
	return state
}

// writeUpdateState writes update.json.
func writeUpdateState(dir string, state updateState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, updateStateFileName), data, 0o644)
}

// newUpdateClient makes an HTTP client with a total timeout and the redirect rules.
func newUpdateClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:       timeout,
		Transport:     updateTransport,
		CheckRedirect: checkUpdateRedirect,
	}
}

// checkUpdateRedirect lets a redirect go only to github.com or *.githubusercontent.com, in HTTPS.
func checkUpdateRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return errors.New("trop de redirections")
	}
	if !isAllowedRedirect(req.URL) {
		return errors.New("redirection refusée")
	}
	return nil
}

// isAllowedRedirect tells if a redirect can go to this address.
func isAllowedRedirect(u *url.URL) bool {
	if u.Scheme != "https" || u.User != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == updateRedirectHost || strings.HasSuffix(host, updateRedirectHostSuffix)
}

// isAllowedAsset tells if a download address is a file of a D-HELPER release.
func isAllowedAsset(address string) bool {
	return strings.HasPrefix(address, updateAssetPrefix) && !strings.Contains(address, "..")
}

// updateGet sends a GET request with the D-HELPER headers.
// It gives an error if the response is not 200. The caller closes the body.
func updateGet(client *http.Client, address, accept string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "D-HELPER/"+version)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("connexion à GitHub impossible")
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("réponse %d de GitHub", resp.StatusCode)
	}
	return resp, nil
}

// fetchLatestRelease reads the latest release in the GitHub API.
func fetchLatestRelease() (release, error) {
	var latest release
	resp, err := updateGet(newUpdateClient(updateAPITimeout), updateAPIURL, "application/vnd.github+json")
	if err != nil {
		return latest, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(io.LimitReader(resp.Body, updateMaxAPISize)).Decode(&latest); err != nil {
		return latest, errors.New("réponse de GitHub illisible")
	}
	return latest, nil
}

// assetURLs gives the download addresses of the program and of its checksum file.
func assetURLs(latest release) (exeURL, sumURL string, err error) {
	for _, asset := range latest.Assets {
		switch asset.Name {
		case exeName:
			exeURL = asset.URL
		case checksumName:
			sumURL = asset.URL
		}
	}
	if exeURL == "" || sumURL == "" {
		return "", "", errors.New("fichier absent de la version publiée")
	}
	if !isAllowedAsset(exeURL) || !isAllowedAsset(sumURL) {
		return "", "", errors.New("adresse de téléchargement refusée")
	}
	return exeURL, sumURL, nil
}

// installRelease downloads the program of the release, checks its SHA-256 and replaces exe.
// Then it registers the protocol again. installed is true if exe is replaced.
func installRelease(latest release, exe string) (installed bool, err error) {
	exeURL, sumURL, err := assetURLs(latest)
	if err != nil {
		return false, err
	}
	client := newUpdateClient(updateDownloadTimeout)
	want, err := downloadChecksum(client, sumURL)
	if err != nil {
		return false, err
	}
	temp, got, err := downloadProgram(client, exeURL, filepath.Dir(exe))
	if err != nil {
		return false, err
	}
	if !bytes.Equal(got, want) {
		_ = os.Remove(temp)
		return false, errUpdateCorrupt
	}
	if err := replaceFile(temp, exe); err != nil {
		_ = os.Remove(temp)
		return false, err
	}
	if err := updateRegister(exe); err != nil {
		return true, err
	}
	return true, nil
}

// downloadChecksum downloads the .sha256 file and gives the SHA-256 that it contains.
// The format is "<hex>  d-helper.exe" (output of sha256sum).
func downloadChecksum(client *http.Client, address string) ([]byte, error) {
	resp, err := updateGet(client, address, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, updateMaxChecksumSize))
	if err != nil {
		return nil, errors.New("téléchargement interrompu")
	}
	fields := strings.Fields(string(data))
	if len(fields) != 2 || (fields[1] != exeName && fields[1] != "*"+exeName) {
		return nil, errors.New("fichier d'empreinte illisible")
	}
	sum, err := hex.DecodeString(fields[0])
	if err != nil || len(sum) != sha256.Size {
		return nil, errors.New("fichier d'empreinte illisible")
	}
	return sum, nil
}

// downloadProgram downloads the program into a temporary file of dir.
// It gives the path of the temporary file and the SHA-256 of its content.
// If an error occurs, it deletes the temporary file.
func downloadProgram(client *http.Client, address, dir string) (string, []byte, error) {
	resp, err := updateGet(client, address, "")
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.ContentLength > updateMaxSize {
		return "", nil, errors.New("fichier trop grand")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, err
	}
	file, err := os.CreateTemp(dir, exeName+".*.download")
	if err != nil {
		return "", nil, err
	}
	temp := file.Name()
	hash := sha256.New()
	// Read one byte more than the maximum to find a file that is too large.
	n, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(resp.Body, updateMaxSize+1))
	closeErr := file.Close()
	switch {
	case copyErr != nil:
		err = errors.New("téléchargement interrompu")
	case n > updateMaxSize:
		err = errors.New("fichier trop grand")
	case closeErr != nil:
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(temp)
		return "", nil, err
	}
	return temp, hash.Sum(nil), nil
}
