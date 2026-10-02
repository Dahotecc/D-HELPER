# D-HELPER

Petit programme Windows installé sur chaque poste : il ouvre dans l'explorateur un dossier du Dropbox Dahotecc quand on clique sur « Ouvrir le dossier » dans D-HUB.

D-HELPER tourne uniquement sur les postes Windows, jamais sur le NUC. Il ne fait qu'une chose : ouvrir un dossier. Il n'ouvre jamais de fichier, n'exécute rien, ne se connecte à rien.

## Fonctionnement

1. Dans D-HUB, l'utilisateur clique sur « Ouvrir le dossier ».
2. Si le navigateur n'est pas sur le serveur, D-HUB navigue vers un lien `d-helper://` (et copie le chemin dans le presse-papiers).
3. Windows lance `d-helper.exe` avec ce lien (protocole enregistré à l'installation).
4. D-HELPER vérifie le lien, trouve le dossier Dahotecc du poste, vérifie que le dossier demandé y est bien, puis lance `explorer.exe` sur ce dossier.
5. En cas de problème, une fenêtre de message explique pourquoi. Aucune fenêtre console ne s'affiche.

### Schéma du lien

```
d-helper://open?path=<chemin relatif encodé>
```

- `open` : seule action existante.
- `path` : chemin relatif à la racine Dahotecc, séparateur `/`, encodé par le navigateur avec `encodeURIComponent`. C'est la valeur `chemin` renvoyée par `POST /api/projets/:id/ouvrir` de D-HUB.
- Exemple : `Devis 2026/260915-Client-Nom/FAB` donne `d-helper://open?path=Devis%202026%2F260915-Client-Nom%2FFAB`.
- Aucun autre paramètre n'est accepté.

### Racine Dahotecc sur le poste

- Par défaut : lue dans `%LOCALAPPDATA%\Dropbox\info.json` (fichier tenu par le client Dropbox). Compte `business` s'il existe, sinon compte `personal`, puis sous-dossier `Dahotecc`.
- Surcharge : la variable d'environnement `D_HELPER_ROOT` (chemin absolu du dossier Dahotecc) passe avant `info.json`. Utile pour les tests ou un poste atypique.
- Racine introuvable : message « D-HELPER ne trouve pas le dossier Dropbox Dahotecc sur ce poste. », aucun dossier ouvert.

### Messages affichés

| Situation | Message |
|---|---|
| Racine introuvable | D-HELPER ne trouve pas le dossier Dropbox Dahotecc sur ce poste. |
| Lien refusé | Lien D-HELPER refusé : <raison courte>. |
| Dossier absent | Dossier introuvable : <chemin relatif>. Il a peut-être été déplacé ; relisez le Dropbox dans D-HUB. |
| Double-clic, pas encore installé | Installer D-HELPER sur ce poste ? Les liens « Ouvrir le dossier » de D-HUB ouvriront l'explorateur. (Oui/Non) |
| Double-clic, déjà installé | Mettre à jour D-HELPER sur ce poste ? (Oui/Non) |
| Double-clic sur le programme installé | D-HELPER est déjà installé sur ce poste. Réparer l'installation ? (Oui/Non) |
| Installation, mise à jour, réparation | D-HELPER est installé. Les liens « Ouvrir le dossier » de D-HUB ouvriront l'explorateur. |
| Programme installé en cours d'utilisation | ... impossible : le programme installé est en cours d'utilisation. Fermez les fenêtres D-HELPER puis relancez. |
| Désinstallation | D-HELPER est désinstallé. (avec l'emplacement du programme à supprimer) |

## Installation sur un poste

À faire une fois par poste et par utilisateur Windows. Pas besoin de droits administrateur, pas besoin de ligne de commande.

1. Télécharger `d-helper.exe` depuis D-HUB : après un clic sur « Ouvrir le dossier », l'alerte « Rien ne s'est ouvert ? » propose « Télécharger D-HELPER ». (Autres sources : la dernière Release GitHub du dépôt, ou la copie déposée dans le Dropbox Dahotecc.)
2. Le navigateur peut avertir que ce type de fichier peut être dangereux (programme non signé, peu téléchargé). Choisir « Conserver » (Edge : « ... » > « Conserver », puis « Afficher plus » > « Conserver quand même » ; Chrome : « Conserver »).
3. Double-cliquer sur `d-helper.exe` (dans le dossier Téléchargements, ou depuis la liste des téléchargements du navigateur).
4. Windows SmartScreen peut afficher « Windows a protégé votre ordinateur » (le programme n'est pas signé). Cliquer sur « Informations complémentaires », puis sur « Exécuter quand même ».
5. D-HELPER demande « Installer D-HELPER sur ce poste ? ». Cliquer sur « Oui ». (« Non » ne fait rien.)
6. Le message « D-HELPER est installé... » s'affiche.

L'installation copie le programme dans `%LOCALAPPDATA%\Programs\d-helper\d-helper.exe` et enregistre le protocole `d-helper://` pour l'utilisateur courant (registre `HKEY_CURRENT_USER\Software\Classes\d-helper`). Le fichier téléchargé peut ensuite être supprimé.

(Facultatif) Vérifier l'empreinte d'un fichier de la Release, dans PowerShell, dans le dossier du fichier : `Get-FileHash .\d-helper.exe` doit donner la valeur du fichier `d-helper.exe.sha256`.

### Mise à jour

Même geste : télécharger la nouvelle version depuis D-HUB, double-cliquer dessus, puis répondre « Oui » à « Mettre à jour D-HELPER sur ce poste ? ». La copie installée est remplacée.

Si la copie installée est en cours d'utilisation (une fenêtre de message D-HELPER est encore ouverte), D-HELPER la met de côté (`d-helper.exe.old`, supprimé à la mise à jour suivante) avant de la remplacer. Si ce n'est pas possible, le message demande de fermer les fenêtres D-HELPER puis de relancer.

### Réparation

Un double-clic sur le programme installé (`%LOCALAPPDATA%\Programs\d-helper\d-helper.exe`) demande « D-HELPER est déjà installé sur ce poste. Réparer l'installation ? ». « Oui » enregistre à nouveau le protocole `d-helper://`, sans copier le programme.

### En ligne de commande

`d-helper.exe --install` fait la même installation sans question. `d-helper.exe --help` affiche les options.

## Première utilisation

Au premier clic sur « Ouvrir le dossier », le navigateur demande « Ouvrir D-HELPER ? » (ou « Ouvrir d-helper.exe ? »). Cocher « Toujours autoriser » (case proposée quand D-HUB est en HTTPS ; sinon la question revient à chaque clic), puis cliquer sur « Ouvrir ».

## Désinstallation

Touches Windows + R, puis coller la ligne suivante et valider :

```
%LOCALAPPDATA%\Programs\d-helper\d-helper.exe --uninstall
```

Ou, dans PowerShell :

```powershell
& "$env:LOCALAPPDATA\Programs\d-helper\d-helper.exe" --uninstall
```

La clé de registre du protocole est supprimée. Le programme reste sur le disque : supprimer ensuite le dossier `%LOCALAPPDATA%\Programs\d-helper` (le message rappelle son emplacement).

## Sécurité

N'importe quel site web peut déclencher un lien `d-helper://`. D-HELPER ne fait donc confiance à rien de ce qu'il reçoit.

Il refuse :

- un lien qui n'a pas exactement la forme `d-helper://open?path=...` : autre action, paramètre inconnu ou répété, utilisateur, port ou fragment ;
- un chemin vide, absolu (`/x`, `\x`, `C:...`, `\\serveur\...`), ou avec un segment `.`, `..` ou vide (`a//b`) : il pourrait sortir du dossier Dahotecc ;
- les caractères `:`, `\`, `< > " | ? *`, NUL et caractères de contrôle, et les noms finissant par un point ou un espace : Windows les interprète de façon particulière ;
- les noms réservés par Windows (`CON`, `PRN`, `AUX`, `NUL`, `COM1`, `LPT1`...) : ce sont des périphériques, pas des dossiers ;
- un dossier qui, une fois les liens symboliques et jonctions résolus, se trouve hors du dossier Dahotecc ;
- une cible qui n'est pas un dossier : D-HELPER n'ouvre jamais de fichier et n'exécute jamais rien.

Le dossier est ouvert uniquement par `explorer.exe "<chemin absolu>"`, lancé directement (pas de `cmd`, pas de shell, pas de `ShellExecute` sur le chemin). D-HELPER ne fait aucune connexion réseau, n'a aucun secret et n'écrit aucun journal.

## Développement

Go n'est pas installé sur le poste de dev : tout passe par le container Go officiel, depuis le dossier du dépôt dans WSL.

Contrôles complets (format, `go vet` Linux et Windows, tests, build Windows) :

```bash
docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp -e CGO_ENABLED=0 -v "$PWD":/src -w /src golang:1.27.1-alpine3.24 sh -ec '
  test -z "$(gofmt -l .)"
  go vet ./...
  GOOS=windows GOARCH=amd64 go vet ./...
  go test ./...
  GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H=windowsgui -X main.version=dev" -o dist/d-helper.exe .
'
```

Le programme compilé est dans `dist/d-helper.exe` (ignoré par git). `-H=windowsgui` évite la fenêtre console ; `-X main.version=...` fixe la version affichée par `--version`.

Organisation du code :

| Fichier | Rôle |
|---|---|
| `main.go` | arguments (aucun = double-clic, `--install`, `--uninstall`, `--version`, `--help`, lien) et messages |
| `link.go` | lecture du lien et toutes les règles de sécurité |
| `root.go` | recherche de la racine Dahotecc (`D_HELPER_ROOT`, puis `info.json`) |
| `install.go` | choix de l'action au double-clic, copie du programme, appel du registre |
| `*_windows.go` | registre, fenêtre de message, `explorer.exe`, chemin réel (jonctions) |
| `*_other.go` | remplacements pour lancer les tests sous Linux |
| `*_test.go` | tests |

### Tester sur un poste Windows sans rien ouvrir

L'option cachée `--check <lien>` fait toutes les vérifications et écrit le résultat au lieu d'ouvrir le dossier. Le programme n'ayant pas de console, passer par `| Write-Output` pour voir le texte dans PowerShell :

```powershell
.\d-helper.exe --version | Write-Output
.\d-helper.exe --check "d-helper://open?path=Devis%202026" | Write-Output
.\d-helper.exe --check "d-helper://open?path=..%2FWindows" | Write-Output
```

Si les accents s'affichent mal, lancer d'abord `[Console]::OutputEncoding = [Text.Encoding]::UTF8`.

Pour tester sur un dossier de test plutôt que sur le vrai Dropbox : `$env:D_HELPER_ROOT = "C:\chemin\vers\un\dossier"` avant la commande.

## Publication

1. Travail sur `dev` ; chaque push lance le workflow `ci` (format, `go vet`, tests, build Windows).
2. PR `dev` -> `main`, CI verte, merge par merge commit.
3. Le push sur `main` lance le workflow `release` : tests, build de `d-helper.exe` (version `sha-<court>`), fichier `d-helper.exe.sha256`, puis Release GitHub avec le tag `sha-<court>`. Les notes de version reprennent les sujets de commit, sans les domaines techniques (`CI`, `Docker`, `Git`, `Doc`, `Santé`, `Sauvegarde`, `Config`, `Style`).
4. Rendre la nouvelle version disponible dans D-HUB (et dans le Dropbox Dahotecc) : chaque poste la télécharge et fait un double-clic dessus (« Mettre à jour D-HELPER sur ce poste ? » > « Oui »).
