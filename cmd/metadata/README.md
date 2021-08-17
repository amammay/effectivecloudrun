# Metadata Server

The metadata server is a simple way to retrieve key pieces of information in regards to the context of where your code
is running at.

The most common usages of the metadata server is retrieve some pieces of information around where your code is running
at, and the identity that is running the code. According
to [here](https://cloud.google.com/run/docs/reference/container-contract#sandbox) we have the following available for us

1. Project id
2. Project number
3. Cloud run geo-graphical region
4. Cloud run instance id
5. [Identity tokens (used to call other services that can validate an identity token)](https://cloud.google.com/run/docs/securing/service-identity#identity_tokens)
6. [Access Tokens (used to call GCP Api's) ](https://cloud.google.com/run/docs/securing/service-identity#access_tokens)

## How to use it?

You can ping the metadata server in two separate ways, using the client library or using a standard http client. For
this example we are going to use the client library for ease of use.

You can install the metadata server client library with below

```shell
go get cloud.google.com/go/compute/metadata
```

From there we can start using it in our code like so...


