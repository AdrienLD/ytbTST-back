#!/usr/bin/env bash
set -euo pipefail
export PATH=/usr/local/go/bin:/usr/local/bin:/usr/local/sbin:/usr/sbin:/usr/bin:/sbin:/bin
export GOTOOLCHAIN=local
cd "$(dirname "$0")"
test "$(hostname)" = ytst-back
exec 9>"${YTST_DEPLOY_LOCK:-/run/lock/ytst-back-deploy.lock}"
flock -n 9 || { echo 'Un déploiement est déjà en cours.' >&2; exit 1; }
if [[ -n "${CHANGED_FILES:-}" ]]; then
  test -f "$CHANGED_FILES" || { echo 'Liste CHANGED_FILES introuvable.' >&2; exit 1; }
  echo "Fichiers changés : $(wc -l < "$CHANGED_FILES")"
fi
test ! -L myapp && { test ! -e myapp || test -f myapp; } || {
  echo 'myapp doit être un fichier ordinaire, jamais un lien ou un dossier.' >&2
  exit 1
}
command -v go >/dev/null || { echo 'Le runtime Go doit être installé par un opérateur.' >&2; exit 1; }
go version
go mod download
go mod verify
# Le test YouTube existant ignore explicitement son appel réel en mode court.
go test -short ./...
build=$(mktemp -d .ytst-build.XXXXXX)
trap 'rm -f -- "$build/myapp"; rmdir -- "$build"' EXIT
go build -trimpath -o "$build/myapp" .
chmod 755 "$build/myapp"
mv -f -- "$build/myapp" myapp
echo 'Binaire myapp compilé. Le démarrage manuel de l’application reste inchangé.'
