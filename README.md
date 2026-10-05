# D-HELPER

Petit programme Windows installé sur chaque poste : il ouvre dans l'explorateur le dossier Dahotecc demandé par « Ouvrir le dossier » dans D-HUB.
Il n'ouvre jamais de fichier et n'exécute rien. Le détail du lien `d-helper://` et des règles est dans les commentaires du code.

## Installation

1. Télécharger https://github.com/Dahotecc/D-HELPER/releases/latest/download/d-helper.exe (lien aussi affiché par D-HUB).
2. Double-cliquer sur le fichier, puis répondre « Oui ». Si SmartScreen bloque : « Informations complémentaires » > « Exécuter quand même ».

## Mises à jour

Après l'ouverture d'un dossier, D-HELPER vérifie au plus une fois par jour s'il existe une version plus récente sur GitHub.
Si oui, il propose de l'installer ; une version refusée n'est pas reproposée avant 7 jours.
`d-helper.exe --update` vérifie tout de suite.

## Désinstallation

`%LOCALAPPDATA%\Programs\d-helper\d-helper.exe --uninstall` (Windows + R), puis supprimer ce dossier.

## Sécurité

- N'importe quel site peut déclencher un lien `d-helper://` : D-HELPER refuse tout chemin qui pourrait sortir du dossier Dahotecc ou viser autre chose qu'un dossier.
- Une seule connexion réseau, vers GitHub, pour les mises à jour : aucune donnée du lien n'y est utilisée.
- Une mise à jour est installée seulement après un « Oui » et si son empreinte SHA-256 est correcte ; jamais de retour à une version plus ancienne.
- Aucun secret, aucun journal, aucun droit administrateur.

## Développement

Depuis le dossier du dépôt dans WSL (contrôles, tests, puis build dans `dist/d-helper.exe`) :

```bash
docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp -e CGO_ENABLED=0 -v "$PWD":/src -w /src golang:1.27.1-alpine3.24 sh -ec '
  test -z "$(gofmt -l .)"; go vet ./...; GOOS=windows GOARCH=amd64 go vet ./...; go test ./...
  GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H=windowsgui -X main.version=dev" -o dist/d-helper.exe .'
```

## Publication

Merge de la PR `dev` -> `main`, puis le workflow `release` publie la Release `vAAAA.MM.JJ.HHMM`. Les postes se voient proposer la mise à jour.
