SECRET_NAME=gcp-get-secret-php-demo-pgpassword
demo:
	docker build -t gcp-get-secret-php .
	gcloud secrets describe  $(SECRET_NAME) || \
	gcloud secrets create $(SECRET_NAME) --replication-policy automatic && \
	uuidgen | gcloud secrets versions add $(SECRET_NAME) --data-file=- 
	ID=$$(docker run \
		-v $$HOME/.config/gcloud:/root/.config/gcloud \
		-e GOOGLE_CLOUD_PROJECT=$$(gcloud config get-value core/project) \
		-d \
		-P gcp-get-secret-php) ; \
	echo "http://$$(docker port $$ID 80)"
