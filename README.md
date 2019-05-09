# stockercm

Sentiment processor using `Cloud Functions`, `Cloud PubSub`, `Natural Language Processing API`, and `Cloud Dataflow`.

## Setup

Create PubSub topics

```shell
gcloud pubsub topics create stocker-source
gcloud pubsub topics create stocker-processed
```

Create Cloud Dataflow job to drain processed topic to BigQuery

```shell
gcloud dataflow jobs run stocker-processed-to-bq-pump \
  --gcs-location gs://dataflow-templates/latest/PubSub_to_BigQuery \
  --parameters topic=projects/${GCP_PROJECT}/topics/stocker-processed,\
    table=${GCP_PROJECT}:stocker.content \
  --region us-central1
```

## Deploy

Deploy a Cloud Function to process the events from `stocker-source` topic and push results to `stocker-processed`.

```shell
gcloud functions deploy stocker-process \
  --entry-point ProcessorSentiment \
  --set-env-vars "PID=${GCP_PROJECT}" \
  --memory 256MB \
  --region us-central1 \
  --runtime go112 \
  --trigger-topic stocker-source \
  --timeout=300s
```

## Cleanup

Delete the Cloud Function

```shell
gcloud functions delete stocker-process --region us-central1
```

Delete the Cloud Dataflow job

```shell
gcloud dataflow jobs cancel stocker-processed-to-bq-pump --region us-central1
```
