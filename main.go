// Command d-helper opens a folder of the Dahotecc Dropbox root in the Windows explorer.
// The browser starts it with a link: d-helper://open?path=<encoded relative path>.
//
// Where to find the details:
//   - link.go: the link format and all the security rules of the link.
//   - root.go: the Dahotecc root (variable D_HELPER_ROOT, then the Dropbox info.json file).
//   - install.go: install, update by double-click, repair, uninstall.
//   - update.go: the automatic update from GitHub (the only network connection).
//
// Test on Windows without opening a folder: the hidden option --check writes the result.
// The program has no console, thus use "| Write-Output" in PowerShell:
//
//	.\d-helper.exe --check "d-helper://open?path=Devis%202026" | Write-Output
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// version is set at build time: -ldflags "-X main.version=AAAA.MM.JJ.HHMM".
// The version "dev" never checks for an update.
var version = "dev"

// Texts that the user sees (French, from the D-HELPER contract).
const (
	msgRootNotFound = "D-HELPER ne trouve pas le dossier Dropbox Dahotecc sur ce poste."
	msgInstalled    = "D-HELPER est installé. Les liens « Ouvrir le dossier » de D-HUB ouvriront l'explorateur."
	msgUninstalled  = "D-HELPER est désinstallé."

	// Questions of a start without arguments (double-click on the program).
	msgAskInstall = "Installer D-HELPER sur ce poste ? Les liens « Ouvrir le dossier » de D-HUB ouvriront l'explorateur."
	msgAskUpdate  = "Mettre à jour D-HELPER sur ce poste ?"
	msgAskRepair  = "D-HELPER est déjà installé sur ce poste. Réparer l'installation ?"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run does the command of the arguments and gives the exit code.
func run(args []string) int {
	if len(args) == 0 {
		// Double-click on the program: install, update or repair.
		return doStart()
	}
	// The browser gives the link as the only argument.
	if len(args) == 1 && isLink(args[0]) {
		return openLink(args[0])
	}

	switch {
	case len(args) == 1 && (args[0] == "--help" || args[0] == "-h" || args[0] == "/?"):
		report(helpText())
		return 0
	case len(args) == 1 && args[0] == "--version":
		report("D-HELPER " + version)
		return 0
	case len(args) == 1 && args[0] == "--install":
		return doInstall()
	case len(args) == 1 && args[0] == "--uninstall":
		return doUninstall()
	case len(args) == 1 && args[0] == "--update":
		// Immediate check, without the 24-hour limit.
		return runUpdate(time.Now(), true)
	case len(args) == 2 && args[0] == "--check":
		// Hidden option for the tests: check a link, write the result, do not open a folder.
		return checkLink(args[1])
	}
	report("Arguments inconnus.\n\n" + helpText())
	return 2
}

// openLink opens the folder of a link, or shows the reason of the error.
func openLink(raw string) int {
	dir, rel, err := resolveLink(raw)
	if err != nil {
		showError(errorMessage(err, rel))
		return 1
	}
	if err := openFolder(dir); err != nil {
		showError("D-HELPER ne peut pas ouvrir l'explorateur : " + err.Error())
		return 1
	}
	// The folder is open. Then check for an update (one time in 24 hours at most).
	// No data of the link goes to the update.
	autoUpdate()
	return 0
}

// checkLink does the same checks as openLink, but writes the result on the standard output.
func checkLink(raw string) int {
	dir, rel, err := resolveLink(raw)
	if err != nil {
		fmt.Println(errorMessage(err, rel))
		return 1
	}
	fmt.Println("OK : " + dir)
	return 0
}

// resolveLink reads the link, finds the root and gives the real path of the folder.
// It also gives the relative path of the link, for the messages.
func resolveLink(raw string) (dir, rel string, err error) {
	rel, err = parseLink(raw)
	if err != nil {
		return "", "", err
	}
	root, err := findRoot()
	if err != nil {
		return "", rel, err
	}
	dir, err = resolveFolder(root, rel)
	return dir, rel, err
}

// errorMessage gives the French text of an error for the user.
func errorMessage(err error, rel string) string {
	var refused *linkError
	switch {
	case errors.As(err, &refused):
		return "Lien D-HELPER refusé : " + refused.reason + "."
	case errors.Is(err, errRootNotFound):
		return msgRootNotFound
	case errors.Is(err, errFolderNotFound):
		return "Dossier introuvable : " + rel + ". Il a peut-être été déplacé ; relisez le Dropbox dans D-HUB."
	default:
		return "Erreur D-HELPER : " + err.Error()
	}
}

// doStart is the start without arguments (double-click on the program).
// It asks the user, then installs, updates or repairs D-HELPER.
// If the user clicks "Non", it does nothing and shows no other message.
func doStart() int {
	running, err := os.Executable()
	if err != nil {
		showError("Installation de D-HELPER impossible : " + err.Error())
		return 1
	}
	dest, err := installPath()
	if err != nil {
		showError("Installation de D-HELPER impossible : " + err.Error())
		return 1
	}

	switch chooseAction(running, filepath.Dir(dest), isInstalled(dest)) {
	case actionRepair:
		if !askYesNo(msgAskRepair) {
			return 0
		}
		return showInstallResult("Réparation de D-HELPER impossible : ", repair)
	case actionUpdate:
		if !askYesNo(msgAskUpdate) {
			return 0
		}
		return showInstallResult("Mise à jour de D-HELPER impossible : ", install)
	default:
		if !askYesNo(msgAskInstall) {
			return 0
		}
		return doInstall()
	}
}

// doInstall installs D-HELPER and shows the result.
func doInstall() int {
	return showInstallResult("Installation de D-HELPER impossible : ", install)
}

// showInstallResult does the step and shows the result.
// If an error occurs, the message starts with failure.
func showInstallResult(failure string, step func() (string, error)) int {
	exe, err := step()
	if err != nil {
		showError(failure + err.Error())
		return 1
	}
	showInfo(msgInstalled + "\n\nProgramme installé : " + exe)
	return 0
}

// doUninstall removes the d-helper:// protocol and shows the result.
// The copy of the program stays. The message tells where it is.
func doUninstall() int {
	exe, err := uninstall()
	if err != nil {
		showError("Désinstallation de D-HELPER impossible : " + err.Error())
		return 1
	}
	showInfo(msgUninstalled + "\n\nVous pouvez supprimer le programme : " + exe)
	return 0
}

// report writes a text on the standard output.
// If there is no standard output (program started without a console), it shows a message box.
func report(text string) {
	if _, err := fmt.Fprintln(os.Stdout, text); err != nil {
		showInfo(text)
	}
}

// helpText gives the French help text.
func helpText() string {
	return "D-HELPER " + version + "\n" +
		"Ouvre dans l'explorateur un dossier Dahotecc depuis un lien d-helper:// de D-HUB.\n\n" +
		"Utilisation :\n" +
		"  d-helper.exe               (double-clic) installe, met à jour ou répare D-HELPER\n" +
		"  d-helper.exe --install     installe D-HELPER pour l'utilisateur courant\n" +
		"  d-helper.exe --uninstall   désinstalle D-HELPER\n" +
		"  d-helper.exe --update      cherche et propose la dernière version publiée sur GitHub\n" +
		"  d-helper.exe --version     affiche la version\n" +
		"  d-helper.exe --help        affiche cette aide"
}
