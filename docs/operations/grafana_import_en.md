# Grafana Import

For a better overview, it might be preferred to view some contents (e.g. node metrics) of the support archive in Grafana
instead of tediously scrolling through the file.

For that purpose, a docker-compose has been created in the `grafana`-Directory.
Simply drop your metrics in a subfolder of `grafana/archives` and spin up the docker-compose with

```shell
docker compose up
```

The dashboards are available at http://localhost:3000, username/password are `admin`/`admin`.