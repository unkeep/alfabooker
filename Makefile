deploy:
	gcloud config set project ontrack-384219 && gcloud functions deploy on-track-func \
	--source=${PWD} \
	--region=europe-central2 \
	--entry-point=ServeHTTP \
	--source=. \
	--trigger-http
