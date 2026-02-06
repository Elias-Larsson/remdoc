# remdoc

remdoc is a CLI tool for deploying and managing Docker containers on a remote
server via the Portainer API.

## Status

Still in development and early release. Some features and stuff still in planning.

## Requirements

- Go 1.21+ (for installation)
- Access to a Portainer instance

## Install

### Install with Go

```sh
go install github.com/Elias-Larsson/remdoc/cmd/remdoc@latest
```

Make sure `$GOPATH/bin` (usually `~/go/bin`) is in your `PATH`.

### Install from source

```sh
git clone https://github.com/Elias-Larsson/remdoc.git
cd remdoc
go build -o remdoc ./cmd/remdoc
```

## Login (recommended)

Use the CLI to authenticate with your Portainer username and password. This will
store a JWT in `~/.remdoc/config.json` with secure permissions:

```sh
remdoc login --username admin
```

You can also pass a password directly (not recommended on shared systems):

```sh
remdoc login -u admin -p yourpassword
```

## Configure (manual)

You can also create/edit the config file manually if you already have a JWT:

- Linux/macOS: `$HOME/.config/remdoc/config.json`
- Windows: `%APPDATA%\remdoc\config.json`

Example `config.json`:

```json
{
  "portainer_url": "https://your-portainer.example.com",
  "jwt": "<YOUR_PORTAINER_JWT>"
}
```

## Usage

Deploy a container:

```sh
remdoc deploy --image nginx:latest --name my-nginx --port 8080:80
```

List containers:

```sh
remdoc status
```

Start/stop/remove containers:

```sh
remdoc start <container>
remdoc stop <container>
remdoc rm <container>
```

Deploy a local compose file as a stack:

```sh
remdoc compose --file ./docker-compose.yml --name my-stack
```

## Commands

- `login` – authenticate and store JWT (recommended)
- `deploy` – deploy a single container
- `status` – list containers
- `start` – start a container
- `stop` – stop a container
- `rm` – remove a container
- `compose` – deploy a Docker Compose file as a stack

## Notes

- Stack names are required by Portainer; if you omit `--name`, the file name is used.
- Compose deployments currently use the Portainer stack API with the compose file content.

## License

MIT License - see [LICENSE](LICENSE) for details.
