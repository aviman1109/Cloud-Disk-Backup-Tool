package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	auth "golang.org/x/oauth2/google"
	"google.golang.org/protobuf/proto"
)

type SnapshotList struct {
	Kind     string         `json:"kind"`
	ID       string         `json:"id"`
	Items    []SnapshotItem `json:"items"`
	SelfLink string         `json:"selfLink"`
}
type SnapshotItem struct {
	Kind                           string   `json:"kind"`
	ID                             string   `json:"id"`
	CreationTimestamp              string   `json:"creationTimestamp"`
	Name                           string   `json:"name"`
	Status                         string   `json:"status"`
	SourceDisk                     string   `json:"sourceDisk"`
	SourceDiskID                   string   `json:"sourceDiskId"`
	DiskSizeGb                     string   `json:"diskSizeGb"`
	StorageBytes                   string   `json:"storageBytes"`
	StorageBytesStatus             string   `json:"storageBytesStatus"`
	SelfLink                       string   `json:"selfLink"`
	LabelFingerprint               string   `json:"labelFingerprint"`
	StorageLocations               []string `json:"storageLocations"`
	DownloadBytes                  string   `json:"downloadBytes"`
	CreationSizeBytes              string   `json:"creationSizeBytes"`
	AutoCreated                    bool     `json:"autoCreated,omitempty"`
	SourceSnapshotSchedulePolicy   string   `json:"sourceSnapshotSchedulePolicy,omitempty"`
	SourceSnapshotSchedulePolicyID string   `json:"sourceSnapshotSchedulePolicyId,omitempty"`
}

func getAuthToken() string {
	ctx := context.Background()
	scopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
	}
	credentials, err := auth.FindDefaultCredentials(ctx, scopes...)
	if err != nil {
		log.Fatal(err)
	}
	token, err := credentials.TokenSource.Token()
	if err != nil {
		log.Fatal(err)
	}

	return (fmt.Sprintf("Bearer %v", string(token.AccessToken)))
}
func GetLatestSnapshot(project string, region string, disk string) (string, error) {
	url, err := url.Parse(fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/snapshots", project))
	method := "GET"

	queryParams := url.Query()
	queryParams.Add("orderBy", "creationTimestamp desc")
	url.RawQuery = queryParams.Encode()

	client := &http.Client{}
	req, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", getAuthToken())

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var snapshot SnapshotList
	err = json.NewDecoder(res.Body).Decode(&snapshot)
	if err != nil {
		return "", err
	}

	for _, item := range snapshot.Items {
		if strings.Contains(item.SourceDisk, disk) {
			return item.SelfLink, nil
		}
	}
	return "", errors.New("snapshot not found")
}

// createSnapshot creates a snapshot of a disk.
func createSnapshot(
	w io.Writer,
	projectID, diskName, snapshotName, zone, region, location, diskProjectID string,
) (string, error) {
	// projectID := "your_project_id"
	// diskName := "your_disk_name"
	// snapshotName := "your_snapshot_name"
	// zone := "europe-central2-b"
	// region := "eupore-central2"
	// location = "eupore-central2"
	// diskProjectID = "YOUR_DISK_PROJECT_ID"

	ctx := context.Background()

	snapshotsClient, err := compute.NewSnapshotsRESTClient(ctx)
	if err != nil {
		return "", fmt.Errorf("NewSnapshotsRESTClient: %w", err)
	}
	defer snapshotsClient.Close()

	if zone == "" && region == "" {
		return "", fmt.Errorf("you need to specify `zone` or `region` for this function to work")
	}

	if zone != "" && region != "" {
		return "", fmt.Errorf("you can't set both `zone` and `region` parameters")
	}

	if diskProjectID == "" {
		diskProjectID = projectID
	}

	disk := &computepb.Disk{}
	locations := []string{}
	if location != "" {
		locations = append(locations, location)
	}

	if zone != "" {
		disksClient, err := compute.NewDisksRESTClient(ctx)
		if err != nil {
			return "", fmt.Errorf("NewDisksRESTClient: %w", err)
		}
		defer disksClient.Close()

		getDiskReq := &computepb.GetDiskRequest{
			Project: projectID,
			Zone:    zone,
			Disk:    diskName,
		}

		disk, err = disksClient.Get(ctx, getDiskReq)
		if err != nil {
			return "", fmt.Errorf("unable to get disk: %w", err)
		}
	} else {
		regionDisksClient, err := compute.NewRegionDisksRESTClient(ctx)
		if err != nil {
			return "", fmt.Errorf("NewRegionDisksRESTClient: %w", err)
		}
		defer regionDisksClient.Close()

		getDiskReq := &computepb.GetRegionDiskRequest{
			Project: projectID,
			Region:  region,
			Disk:    diskName,
		}

		disk, err = regionDisksClient.Get(ctx, getDiskReq)
		if err != nil {
			return "", fmt.Errorf("unable to get disk: %w", err)
		}
	}

	req := &computepb.InsertSnapshotRequest{
		Project: projectID,
		SnapshotResource: &computepb.Snapshot{
			Name:             proto.String(snapshotName),
			SourceDisk:       proto.String(disk.GetSelfLink()),
			StorageLocations: locations,
		},
	}

	op, err := snapshotsClient.Insert(ctx, req)
	if err != nil {
		return "", fmt.Errorf("unable to create snapshot: %w", err)
	}

	if err = op.Wait(ctx); err != nil {
		return "", fmt.Errorf("unable to wait for the operation: %w", err)
	}

	fmt.Fprintf(w, "Snapshot created\n")

	return op.Proto().GetTargetLink(), nil
}
func writeToFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}
func generateSnapshotName(diskName string) string {
	currentTime := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-snapshot-%s", diskName, currentTime)
}
func main() {
	log.SetFlags(0)

	createBackup := flag.Bool("create-backup", false, "Set to true to create a backup, or leave it empty/false to get the latest backup.")
	project := flag.String("project", "", "Project ID for create snapshot")
	region := flag.String("region", "", "The name of the region for create snapshot")
	disk := flag.String("disk", "", "Name of the persistent disk to snapshot")
	flag.Parse()

	if *project == "" || *region == "" || *disk == "" {
		log.Fatal("Error: Missing required argument")
	}

	var snapshotUrl string
	var err error
	if *createBackup {
		snapshotName := generateSnapshotName(*disk)
		snapshotUrl, err = createSnapshot(
			os.Stdout,
			*project,
			*disk,
			snapshotName,
			"",
			*region,
			*region,
			"",
		)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		snapshotUrl, err = GetLatestSnapshot(*project, *region, *disk)
	}

	log.Println("Snapshot Url:", snapshotUrl)
	if err := writeToFile("snapshot.url", snapshotUrl); err != nil {
		log.Fatal(err)
	}
}
