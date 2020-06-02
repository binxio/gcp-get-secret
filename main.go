// Sample quickstart is a basic program that uses Secret Manager.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

func main() {
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)

		if strings.HasPrefix(pair[1], "secret:") {
			fmt.Println("export " + pair[0] + "=" + getSecret(strings.TrimPrefix(pair[1], "secret:")))
		}
	}
}

func getSecret(secretName string) string {
	// Create the client.
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to setup client: %v", err)
	}

	// Build the request.
	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	}

	// Call the API.
	result, err := client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		log.Fatalf("failed to access secret version: %v", err)
	}

	return string(result.Payload.Data)
}
