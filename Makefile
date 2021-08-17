export KO_DOCKER_REPO = gcr.io/mammay-labs

deploy-metadata:
	gcloud run deploy metadata --image=$$(ko publish --preserve-import-paths ./cmd/metadata ) --allow-unauthenticated;

#thing:
#	export thing=$$(cat go.mod); \

