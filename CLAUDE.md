# Comment travailler avec moi

## Mon profil
- Je ne suis pas développeur, mais je sais utiliser VS Code, un terminal et git.
- Je dois pouvoir relire et maintenir le code seul plus tard : privilégie toujours la solution la plus simple et la plus lisible, plutôt que la plus élégante ou la plus complète.

## Méthode de travail
- Explique chaque décision technique en une ou deux phrases simples avant de coder.
- Avance par petites étapes vérifiables. Après chaque étape, dis-moi exactement quoi tester et comment (commande, lien, résultat attendu).
- Ne t'enchaîne pas sur plusieurs étapes non testées sous prétexte que la suite semble évidente - arrête-toi, fais-moi valider, puis continue.

## Autorisations
- Les commandes locales courantes (installer une dépendance, pousser du code vers un environnement de test/dev) sont autorisées sans demander.
- Ne lance jamais une action de déploiement en production, irréversible, ou visible publiquement (publier une Release, envoyer un email, toucher à des données réelles, etc.) sans me le demander explicitement - même si je t'y ai déjà autorisé une fois pour un cas précis, ça ne vaut pas blanc-seing pour la suite.
- Ne lance jamais `d-helper.exe --install` ou `--uninstall` sur mon poste : je le fais moi-même. N'ouvre aucun dossier réel pendant les tests (utilise l'option cachée `--check`).

## Code
- Go, un seul exécutable `d-helper.exe`, Windows uniquement. Seule dépendance autorisée : `golang.org/x/sys`. Toute nouvelle dépendance se discute avec moi avant.
- Commentaires et identifiants en anglais conforme à ASD-STE100 (Simplified Technical English). Référence : `docs/conventions.md` de D-OPS, section 12. Documentation et textes affichés à l'utilisateur en français.
- Code propre à Windows dans les fichiers `*_windows.go` ; leur remplacement pour les tests sous Linux dans les fichiers `*_other.go` (`//go:build !windows`).
- Avant chaque commit : `gofmt`, `go vet` (Linux et `GOOS=windows`), `go test ./...` et build Windows verts (commandes Docker dans le README).
- Le contrat D-HELPER (lien, règles de sécurité, textes affichés) est résumé dans le README. Toute règle de sécurité ajoutée ou modifiée a son test.
- Jamais de tiret cadratin ni demi-cadratin : dans le code, la doc, partout. Utilise un tiret simple (-) à la place.

## Sécurité
- N'importe quel site web peut déclencher un lien `d-helper://` : D-HELPER ne fait confiance à rien de ce qu'il reçoit. Ne relâche jamais une règle de validation sans mon accord.
- Si un secret (mot de passe, clé API, token) doit exister, il ne doit jamais être committé en clair. D-HELPER n'en a aucun. Sa seule connexion réseau est la vérification des mises à jour, vers une adresse GitHub fixe dans le code (`update.go`) ; aucune donnée d'un lien d-helper:// ne l'influence.
- Signale-moi explicitement toute décision qui a un impact sécurité, même mineur, avant de l'implémenter.

## Quand tu vois plusieurs options
- Donne-moi ta recommandation avec une phrase de justification, pas juste une liste neutre d'options.
- Pose-moi la question seulement quand le choix change vraiment ce que tu vas faire ; sinon prends la décision par défaut et dis-le en une ligne.

## Git : commit, push et PR autonomes, merge sur demande (skill `post-pr`)
- Le travail se fait sur `dev`. Commits atomiques et push de `dev` autorisés à tout moment, sans demander (un push sur `dev` lance seulement le workflow `ci`).
- La PR `dev` -> `main` est créée et tenue à jour de façon autonome avec la skill `post-pr` (`.claude/skills/post-pr/SKILL.md`), après les pushs significatifs.
- Ne clôture jamais une session de ta propre initiative.
- Pas d'autorisation permanente pour le merge : merger la PR met en production (ou publie une Release). Tu merges uniquement si je le demande explicitement dans la session.
- Jamais de commit ni de push direct sur `main`.
