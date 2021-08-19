# Metadata Server

The metadata server is a simple way to retrieve key pieces of information in regards to the context of where your code
is running at.

The most common usages of the metadata server is retrieve some pieces of information around where your code is running
at, and the identity that is running the code. According
to [here](https://cloud.google.com/run/docs/reference/container-contract#sandbox) we have the following available for us

### Project id

### How to get it?

```go
package main

import (
	"cloud.google.com/go/compute/metadata"
	"log"
)

func main() {
	// get our gcp project id
	projectID, err := metadata.ProjectID()
	if err != nil {
		log.Fatalf("metadata.ProjectID(): %v", err)
	}
	log.Printf("our gcp project id is %s", projectID)
}
```

### How can I use it?

```go
package main

import (
	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"log"
)

func main() {
	// get our gcp project id
	projectID, err := metadata.ProjectID()
	if err != nil {
		log.Fatalf("metadata.ProjectID(): %v", err)
	}
	log.Printf("our gcp project id is %s", projectID)
	client, err := useProjectID(context.Background(), projectID)
	if err != nil {
		log.Fatalf("useProjectID(): %v", err)
	}
	// TODO use firestore client
}

func useProjectID(ctx context.Context, projectID string) (*firestore.Client, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("firestore.NewClient(): %v", err)
	}
	return client, err
}

```

### Project number

```go
package main

import (
	"cloud.google.com/go/compute/metadata"
	"log"
)

func main() {
	// get our numeric project id (auto generated at project creation from gcp)
	numericProjectID, err := metadata.NumericProjectID()
	if err != nil {
		log.Fatalf("metadata.NumericProjectID(): %v", err)
	}
	log.Printf("our gcp numeric project id %s", numericProjectID)
}
```

### Cloud run geo-graphical region

```go
package main

import (
	"cloud.google.com/go/compute/metadata"
	"log"
)

func main() {
	// get the region our code is running in
	region, err := metadata.Get("instance/region")
	if err != nil {
		log.Fatalf("metadata.Get(instance/region): %v", err)
	}
	log.Printf("our code is running in region %s", region)

}
```

### Cloud run instance id

```go
package main

import (
	"cloud.google.com/go/compute/metadata"
	"log"
)

func main() {
	// unique container instance id
	instanceID, err := metadata.InstanceID()
	if err != nil {
		log.Fatalf("metadata.InstanceID(): %v", err)
	}
	log.Printf("the instance of our cloud run instance is %s", instanceID)

}
```

### [Identity tokens (used to call other services that can validate an identity token)](https://cloud.google.com/run/docs/securing/service-identity#identity_tokens)

```go
package main

import (
	"cloud.google.com/go/compute/metadata"
	"log"
)

func main() {
	// get OIDC token to call other services that can validate an identity token
	identityToken, err := metadata.Get("instance/service-accounts/default/identity?audience=https://some.cloud.run.url.com")
	if err != nil {
		log.Fatalf("metadata.Get(instance/service-accounts/default/identity): %v", err)
	}
	log.Printf("recieve identity token that is %d bytes", len(identityToken))

}
```



### [Access Tokens (used to call GCP Api's) ](https://cloud.google.com/run/docs/securing/service-identity#access_tokens)

```go
package main

import (
	"cloud.google.com/go/compute/metadata"
	"log"
)

func main() {
	// get access token to call gcp api's with, can pass scopes as an query param
	accessToken, err := metadata.Get("instance/service-accounts/default/token?scopes=https://www.googleapis.com/auth/drive,https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("metadata.Get(instance/service-accounts/default/token): %v", err)
	}
	log.Printf("recieve access token that is %d bytes", len(accessToken))

}
```

## How to get it?

You can ping the metadata server in two separate ways, using the client library or using a standard http client. For
this example we are going to use the client library for ease of use.

You can install the metadata server client library with below

```shell
go get cloud.google.com/go/compute/metadata
```

or you can just use the http client and add a specific header [detailed here](https://cloud.google.com/compute/docs/metadata/overview#parts-of-a-request)

