package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	redis "cloud.google.com/go/redis/apiv1beta1"
	redispb "google.golang.org/genproto/googleapis/cloud/redis/v1beta1"
)

var (
	err error
)

// getInstanceProjectID fetches the project id from instance metadata
func getInstanceProjectID(client *metadata.Client) (string, error) {
	projectID, err := client.ProjectID()
	if err != nil {
		return "", err
	}

	return projectID, nil
}

// getInstanceRegion gets the region for the current instance from metadata
func getInstanceRegion(client *metadata.Client) (string, error) {
	zone, err := client.Zone()
	if err != nil {
		return "", err
	}

	// get the current region by dropping the ending letter from the zone
	zoneArr := strings.Split(zone, "-")
	region := strings.Join(zoneArr[:len(zoneArr)-1], "-")

	return region, nil
}

// redisInstancePath generates the gcp style resource path for the redis instance
func redisInstancePath(project string, region string, redisName string) string {
	return fmt.Sprintf("projects/%s/locations/%s/instances/%s", project, region, redisName)
}

// getRedisInstanceDetails fetches configuration information for the redis instance
func getRedisInstanceDetails(ctx context.Context, client *redis.CloudRedisClient, path string) (*redispb.Instance, error) {
	// fetch redis instance details using the instance path
	req := &redispb.GetInstanceRequest{Name: path}
	resp, err := client.GetInstance(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// writeCAFile creates a crt file from list of certificates on a redis instance
func writeCAFile(certs []*redispb.TlsCertificate, file *os.File) error {
	for _, cert := range certs {
		log.Printf(
			"writting ca cert %s to %s expiring on %s",
			cert.Sha1Fingerprint,
			file.Name(),
			time.Unix(cert.ExpireTime.Seconds, int64(cert.ExpireTime.Nanos)),
		)

		_, err := file.WriteString(cert.Cert)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// get cli parameters
	cliCAFile := flag.String("caFile", "/certs/redis_ca.crt", "File to write redis CA certificates to.")
	cliRedisInstanceName := flag.String("redisInstance", "redis-test-01", "Redis instance name.")
	cliProjectID := flag.String("project", "metadata", "GCP project name of redis instance.")
	cliRegion := flag.String("region", "metadata", "GCP region of redis instance.")
	flag.Parse()

	// create client for instance metadata
	cm := metadata.NewClient(&http.Client{})

	// use metadata for project name if not set on cli
	projectID := *cliProjectID
	if projectID == "metadata" {
		projectID, err = getInstanceProjectID(cm)
		if err != nil {
			log.Fatalln(err)
		}
	}

	// use metadata for region if not set on cli
	region := *cliRegion
	if region == "metadata" {
		region, err = getInstanceRegion(cm)
		if err != nil {
			log.Fatalln(err)
		}
	}

	// get the redis instance path in the gcp project
	resourcePath := redisInstancePath(projectID, region, *cliRedisInstanceName)

	// create gcp redis client
	ctx := context.Background()
	cr, err := redis.NewCloudRedisClient(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer cr.Close()

	// fetch info of the redis instance
	resp, err := getRedisInstanceDetails(ctx, cr, resourcePath)
	if err != nil {
		log.Fatalln(err)
	}

	// append redis CA to file
	f, err := os.Create(path.Clean(*cliCAFile))
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer f.Close()

	err = writeCAFile(resp.ServerCaCerts, f)
	if err != nil {
		log.Fatalln(err)
	}
}
