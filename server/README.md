# SimplePresent sync server (minimal scaffold)

Minimal Go scaffold for a Linux-only headless sync server using SQLite and JSON config.

Quick start:

1. Copy `config.json.example` to `config.json` and edit paths.
2. Build:

```bash
cd server
go build -o simplepresent
```

3. Run:

```bash
./simplepresent -config config.json
```
