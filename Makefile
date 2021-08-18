export KO_DOCKER_REPO = gcr.io/mammay-labs

.PHONY: deploy-metadata
deploy-metadata:
	gcloud run deploy metadata --image=$$(ko publish --preserve-import-paths ./cmd/metadata ) --allow-unauthenticated;

.PHONY: deploy-logging
deploy-logging:
	gcloud run deploy logging --image=$$(ko publish --preserve-import-paths ./cmd/structuredlogging ) --allow-unauthenticated;

