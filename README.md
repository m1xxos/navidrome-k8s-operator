# Navidrome Kubernetes Operator

Оператор синхронизирует кастомные ресурсы Kubernetes с Navidrome:

- `Playlist` - управляет удаленным плейлистом в Navidrome
- `Track` - управляет треком внутри конкретного `Playlist`

## Что умеет сейчас

- Создавать/обновлять/удалять плейлисты в Navidrome
- Добавлять трек в плейлист
- Перемещать трек в нужную позицию
- Удалять трек из плейлиста при удалении ресурса `Track`
- Работать идемпотентно (без повторного "добавления того же самого" при стабильном состоянии)
- Логировать синк в стиле Kubernetes logging conventions (структурированные `Info`/`Error`)

## Как работает синк

1. `Playlist` содержит:
   - `spec.navidromeURL`
   - `spec.name`
   - `spec.authSecret` (Secret с `username`/`password`)
2. `Track` ссылается на `Playlist` и определяет трек через одно из полей:
   - `trackRef.trackID`
   - `trackRef.filePath`
   - `trackRef.artist + trackRef.title`
3. Порядок треков:
   - приоритетно используется `spec.priority`
   - `spec.position` поддерживается как fallback для обратной совместимости

## Быстрый старт (kind + helm)

Требования: `kind`, `kubectl`, `helm`, `go`, `docker`.

```bash
make tidy
chmod +x scripts/dev-kind-helm.sh
./scripts/dev-kind-helm.sh
```

Проверка:

```bash
kubectl get playlists,tracks -A
kubectl get pods -n navidrome-operator
kubectl logs -n navidrome-operator deploy/navidrome-operator-navidrome-operator -f
```

## Установка Helm chart (локально)

```bash
helm upgrade --install navidrome-operator ./charts/navidrome-operator \
  -n navidrome-operator \
  --create-namespace
```

Удаление:

```bash
helm uninstall navidrome-operator -n navidrome-operator
```

## Установка через Helm Repo (`helm repo add` + `helm install`)

Если chart опубликован как Helm repository (например, через GitHub Pages), установка выглядит так:

```bash
helm repo add m1xxos https://m1xxos.github.io/navidrome-k8s-operator
helm repo update

helm install navidrome-operator m1xxos/navidrome-operator \
  -n navidrome-operator \
  --create-namespace
```

Обновление:

```bash
helm upgrade navidrome-operator m1xxos/navidrome-operator -n navidrome-operator
```

Примечание: если chart repo развернут по другому URL, замени адрес в `helm repo add`.

## Примеры ресурсов

### Secret авторизации

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

### Playlist

```yaml
apiVersion: navidrome.m1xxos.dev/v1alpha1
kind: Playlist
metadata:
  name: coding-vibes
  namespace: default
spec:
  navidromeURL: "https://music.example.com"
  name: "Coding Vibes"
  authSecret: "navidrome-auth"
```

### Track

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
  priority: 0
```

## Полезные команды

```bash
kubectl get playlist coding-vibes -n default -o yaml
kubectl get tracks -n default
kubectl describe track coding-vibes-track-1 -n default
kubectl logs -n navidrome-operator deploy/navidrome-operator-navidrome-operator --since=10m
```

## Development

```bash
make fmt
make test
make build
```
