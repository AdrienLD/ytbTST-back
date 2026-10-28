# ytbTST-back

Application Go avec PostgreSQL et les intégrations YouTube existantes.
Les réglages restent dans le fichier local `.env`, jamais dans Git ou l’archive.

## Déploiement

Le workflow appelle `fulgurmotors/deploy-action/.github/workflows/deploy.yml@v1`.
Le hub doit d’abord avoir branché le dépôt sur la CT cible en mode archive.
Le code est livré sans `.git` et le manifeste protège les fichiers gérés ;
`.env` et les autres fichiers locaux restent hors manifeste.

`deploy.sh` valide la liste `CHANGED_FILES` fournie par le hub, verrouille la
compilation, télécharge et vérifie les modules, lance `go test -short ./...`,
puis compile et remplace atomiquement le binaire local `myapp`. Une erreur de
test ou de compilation conserve le binaire précédent. Le script accepte aussi
l’absence de `CHANGED_FILES` pendant la transition depuis le relais historique.

L’exploitation actuelle est manuelle : aucun service applicatif systemd n’est
installé et ce script ne lance pas le serveur. Il ne démarre pas PostgreSQL,
ne lance pas de migration SQL et ne renouvelle aucun abonnement YouTube.
Le démarrage normal de l’application conserve ses actions existantes ; il doit
être effectué séparément par l’opérateur lorsqu’il le souhaite.

Go est déjà installé sur la CT et sa version reste celle indiquée par `go.mod`.
Le test d’intégration `TestIsShort`, qui appelle réellement YouTube, est ignoré
en mode court. La CI vérifie compilation, modules et protections du déploiement
sans base, identifiant YouTube ou fichier `.env` réel.

La première migration demande un instantané, une sauvegarde du binaire local
et des réglages sur place, l’agent archive protégé, puis le branchement avant
fusion pour éviter le double relais. L’ancien Git est conservé sous
`.git.avant-deploy-hub-2`. Aucun démarrage de CT ou installation ne découle de
la seule ouverture de cette PR.
