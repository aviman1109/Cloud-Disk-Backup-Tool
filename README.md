# Cloud Disk Backup Tool

This tool creates backups of GCP persistent disks by taking snapshots. It can both create new backups and fetch the latest existing backup.

## Usage

To create a new snapshot:

```sh
./cloud-disk-backup --create-backup=true --project=[gcp project id] --region=[region] --disk=[disk name]
```

To get the latest existing snapshot:

```sh
./cloud-disk-backup --project=[gcp project id] --region=[region] --disk=[disk name]
```

The snapshot URL will be printed to stdout and saved to the file `snapshot.url`.

## Authentication

The tool uses your default GCP credentials to authenticate. Make sure you have authenticated with `gcloud auth application-default login` before running.

## Implementation

The tool uses the GCP Compute API to create snapshots and fetch disk metadata.

The main logic is in `main.go`. It handles command line flags, calls the snapshot creation and fetching functions, and saves the result.

`api.go` contains the API interaction functions for making HTTP requests and JSON parsing.

Let me know if you would like me to expand or modify anything in this draft README!
