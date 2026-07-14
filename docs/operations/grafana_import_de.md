# Grafana-Import

Um einen besseren Überblick zu erhalten, ist es möglicherweise vorteilhaft, bestimmte Inhalte (z. B. die Metriken von Nodes) des Support-Archivs in Grafana anzuzeigen,
anstatt mühsam durch die Datei zu scrollen.

Zu diesem Zweck wurde im Verzeichnis `grafana` eine Docker-Compose-Datei erstellt.
Legen Sie Ihre Metriken einfach in einem Unterordner von `grafana/archives` ab und starten Sie die Docker-Compose-Datei mit

```shell
docker compose up
```

<!-- markdown-link-check-disable-next-line -->
Die Dashboards sind unter http://localhost:3000 verfügbar, Benutzername und Passwort lauten `admin`/`admin`.