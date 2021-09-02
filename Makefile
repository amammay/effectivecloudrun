export KO_DOCKER_REPO = gcr.io/mammay-labs

.PHONY: deploy-metadata
deploy-metadata:
	gcloud run deploy metadata --image=$$(ko publish --preserve-import-paths ./cmd/metadata ) --allow-unauthenticated;

.PHONY: deploy-logging
deploy-logging:
	gcloud run deploy logging --image=$$(ko publish --preserve-import-paths ./cmd/structuredlogging ) --allow-unauthenticated;

.PHONY: deploy-opentelemetry
deploy-opentelemetry:
	gcloud run deploy opentelemetry --image=$$(ko publish --preserve-import-paths ./cmd/opentelemetry ) --allow-unauthenticated;


.PHONY: deploy-ko
deploy-ko:
	gcloud builds submit --config ./cmd/ko/cloudbuild.yaml;

.PHONY: setup-ko-gcr
setup-ko-gcr:
	git clone git@github.com:GoogleCloudPlatform/cloud-builders-community.git --depth=1;
	gcloud builds submit ./cloud-builders-community/ko --config=./cloud-builders-community/ko/cloudbuild.yaml
	rm -rf ./cloud-builders-community
