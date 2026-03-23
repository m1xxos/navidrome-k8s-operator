# Navidrome Kubernetes Operator (for Playlists and Musical Chaos)

Этот оператор следит за двумя кастомными ресурсами в Kubernetes:

- `Playlist` - говорит, какой плейлист должен жить в Navidrome.
- `Track` - говорит, какой трек должен жить внутри конкретного `Playlist`.

Оператор автоматически:

- создает плейлисты,
- обновляет их,
- удаляет их,
- добавляет/переупорядочивает треки,
- удаляет треки из плейлиста при удалении `Track` ресурса.

Да, это тот самый случай, когда Kubernetes управляет музыкой. Мы живем в будущем.

## Как это работает

1. `Playlist` содержит URL Navidrome, имя плейлиста и Secret с логином/паролем.
2. `Track` ссылается на `Playlist` и описывает трек:
   - `trackID`, или
   - `filePath`, или
   - `artist + title`.
3. Контроллеры синхронизируют фактическое состояние в Navidrome с декларативным состоянием в Kubernetes.

## Быстрый старт (kind + helm)

Убедись, что установлены `kind`, `kubectl`, `helm`, `go`.

```bash
make tidy
chmod +x scripts/dev-kind-helm.sh
./scripts/dev-kind-helm.sh
```

После этого проверь:

```bash
kubectl get playlists,tracks -A
kubectl get pods -n navidrome-operator
kubectl logs -n navidrome-operator deploy/navidrome-operator-navidrome-operator -f
```

## Примеры ресурсов

Секрет авторизации:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: navidrome-auth
  namespace: default
type: Opaque
stringData:
  username: your-username
  password: your-password
```

Playlist:

```yaml
apiVersion: navidrome.m1xxos.dev/v1alpha1
kind: Playlist
metadata:
  name: coding-vibes
  namespace: default
spec:
  navidromeURL: "http://navidrome.local"
  name: "Coding Vibes"
  authSecret: "navidrome-auth"
```

Track:

```yaml
apiVersion: navidrome.m1xxos.dev/v1alpha1
kind: Track
metadata:
  name: coding-vibes-track-1
  namespace: default
spec:
  playlistRef:
    name: coding-vibes
  trackRef:
    artist: "Daft Punk"
    title: "Harder Better Faster Stronger"
  position: 0
```

## Важные заметки

- Navidrome уже должен быть доступен по URL из `Playlist.spec.navidromeURL`.
- Secret должен содержать ключи `username` и `password`.
- Если трек не находится по метаданным, попробуй сначала использовать `trackID`.

## Development

```bash
make fmt
make test
make build
```

## Почему это шуточный проект

Потому что это оператор для плейлистов.
Следующий шаг: CRD `Mood`, который меняет музыку по фазам луны и качеству кофе.
