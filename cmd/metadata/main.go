package main

import (
	"cloud.google.com/go/compute/metadata"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type metaDataResponse struct {
	MetadataResults map[string]string `json:"metadata_results"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("run(): %v", err)
	}
}

func run() error {

	response := metaDataResponse{MetadataResults: make(map[string]string)}

	// nice util for deciding if our code is running in the cloud or not
	// it's always useful to make this function call first before pinging the metadata server for further information
	onGCE := metadata.OnGCE()
	log.Printf("is our code running on Google Cloud? %v", onGCE)

	// wrap metatdata server calls around our check if we are on the cloud
	if onGCE {

		// get our gcp project id
		projectID, err := metadata.ProjectID()
		if err != nil {
			return fmt.Errorf("metadata.ProjectID(): %v", err)
		}
		response.MetadataResults["projectID"] = projectID
		log.Printf("our gcp project id is %s", projectID)

		// get our numeric project id (auto generated at project creation from gcp)
		numericProjectID, err := metadata.NumericProjectID()
		if err != nil {
			return fmt.Errorf("metadata.NumericProjectID(): %v", err)
		}
		response.MetadataResults["numericProjectID"] = numericProjectID
		log.Printf("our gcp numeric project id %s", numericProjectID)

		// unique container instance id
		instanceID, err := metadata.InstanceID()
		if err != nil {
			return fmt.Errorf("metadata.InstanceID(): %v", err)
		}
		response.MetadataResults["instanceID"] = instanceID
		log.Printf("the instance of our cloud run instance is %s", instanceID)

		// get the region our code is running in
		region, err := metadata.Get("instance/region")
		if err != nil {
			return fmt.Errorf("metadata.InstanceAttributeValue(): %v", err)
		}
		response.MetadataResults["region"] = region
		log.Printf("our code is running in region %s", region)

		// get access token to call gcp api's with, can pass scopes as an query param
		accessToken, err := metadata.Get("instance/service-accounts/default/token?scopes=https://www.googleapis.com/auth/drive,https://www.googleapis.com/auth/spreadsheets")
		if err != nil {
			return fmt.Errorf("metadata.Get(): %v", err)
		}
		log.Printf("recieve access token that is %d bytes", len(accessToken))

		// get OIDC token to call other services that can validate an identity token
		identityToken, err := metadata.Get("instance/service-accounts/default/identity?audience=https://some.cloud.run.url.com")
		if err != nil {
			return fmt.Errorf("metadata.Get(): %v", err)
		}
		log.Printf("recieve identity token that is %d bytes", len(identityToken))

	}

	// serve out some of the instance metadata
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(writer).Encode(&response); err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	})

	// cloud run will set a PORT env for us
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("starting server on %q", port)

	return http.ListenAndServe(":"+port, nil)
}
