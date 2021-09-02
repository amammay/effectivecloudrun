## What is ko?

`ko` is a technology that allows for building simple go applications into a container.

As long as the application does not need any os level package dependencies, ko is an easy and streamlined way to create
simple go applications as container. To read more check out the [github repo](https://github.com/google/ko) for ko.

## Using ko on gcp

We will focus on using cloud build, gcr and cloud run for our usage of ko in this example. There is a couple extra steps
involved in being able to use ko on cloud build, but the process should be straight forward.

Using the cloud-builders-community repo, we will grab the `ko` builder and run it through our projects cloud build
pipeline. I have this step in my makefile

```makefile
.PHONY: setup-ko-gcr
setup-ko-gcr:
    git clone git@github.com:GoogleCloudPlatform/cloud-builders-community.git --depth=1;
    gcloud builds submit ./cloud-builders-community/ko --config=./cloud-builders-community/ko/cloudbuild.yaml
    rm -rf ./cloud-builders-community
```

Now when this is done, we will have the ko builder available to us under our private gcr repo
of `gcr.io/[project-id]/ko`.

Next we will look at our applications `cloudbuild.yaml` file.

```yaml
steps:
  #  using our ko builder we will build our application that lives in ./cmd/ko
  - name: gcr.io/$PROJECT_ID/ko¬
    entrypoint: /bin/sh
    env:
      - 'KO_DOCKER_REPO=gcr.io/$PROJECT_ID'
    # we write the result of ko publish to a txt file so we can persist the variable between steps
    args:
      - -c
      - |
        echo $(/ko publish --preserve-import-paths ./cmd/ko) > ./ko_container.txt || exit 1

  # Deploy container image to Cloud Run
  - name: 'gcr.io/google.com/cloudsdktool/cloud-sdk'
    entrypoint: /bin/bash
    args:
      - -c
      - |
        gcloud run deploy ko \
        --image=$(cat ./ko_container.txt) \
        --region=us-central1 \
        --platform=managed

```

From us running `gcloud builds submit --config=./cmd/ko/cloudbuild.yaml` we will have our image uploaded to gcr and
deployed to cloud run!





